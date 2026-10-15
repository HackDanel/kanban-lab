package httpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/getkin/kin-openapi/openapi3filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/http/api"
)

var discard = slog.New(slog.NewTextHandler(io.Discard, nil))

func asDomain(t *testing.T, err error) *domain.Error {
	t.Helper()
	var de *domain.Error
	require.ErrorAs(t, err, &de)
	return de
}

func TestValidationErrorFields(t *testing.T) {
	schemaErr := &openapi3.SchemaError{Reason: "minimum string length is 1"}
	err := openapi3.MultiError{
		&openapi3filter.RequestError{Parameter: &openapi3.Parameter{Name: "limit"}, Reason: "out of range"},
		&openapi3filter.RequestError{Err: schemaErr},
		errors.New("something else"),
	}
	de := asDomain(t, validationError(err, http.StatusBadRequest))
	assert.Equal(t, domain.KindValidation, de.Kind)
	assert.Equal(t, []domain.FieldError{
		{Field: "limit", Message: "out of range"},
		{Field: "", Message: "minimum string length is 1"},
		{Field: "", Message: "something else"},
	}, de.Fields)
}

func TestValidationErrorParameterBinding(t *testing.T) {
	cases := []error{
		&api.InvalidParamFormatError{ParamName: "p", Err: errors.New("x")},
		&api.UnmarshalingParamError{ParamName: "p", Err: errors.New("x")},
		&api.RequiredParamError{ParamName: "p"},
		&api.RequiredHeaderError{ParamName: "p", Err: errors.New("x")},
		&api.TooManyValuesForParamError{ParamName: "p", Count: 2},
	}
	for _, err := range cases {
		de := asDomain(t, validationError(err, http.StatusBadRequest))
		assert.Equal(t, []domain.FieldError{{Field: "p", Message: "Некорректное значение параметра"}}, de.Fields, "%T", err)
	}
}

func TestValidationErrorKinds(t *testing.T) {
	assert.Equal(t, domain.KindUnauthorized, domain.KindOf(validationError(&openapi3filter.SecurityRequirementsError{}, http.StatusForbidden)))
	assert.Equal(t, domain.KindNotFound, domain.KindOf(validationError(errors.New("no route"), http.StatusNotFound)))
	assert.Equal(t, domain.KindNotFound, domain.KindOf(validationError(errors.New("method"), http.StatusMethodNotAllowed)))
	assert.Equal(t, domain.KindPayloadTooLarge, domain.KindOf(validationError(&http.MaxBytesError{Limit: 1}, http.StatusBadRequest)))
	assert.Equal(t, domain.KindPayloadTooLarge, domain.KindOf(validationError(fmt.Errorf("reading failed: %s", "http: request body too large"), http.StatusBadRequest)))
}

func TestWriteError(t *testing.T) {
	cases := []struct {
		err    error
		status int
		code   string
	}{
		{domain.ErrForbidden, http.StatusForbidden, "FORBIDDEN"},
		{domain.E(domain.KindWipLimit, "full"), http.StatusConflict, "WIP_LIMIT_EXCEEDED"},
		{domain.E(domain.KindInternal, "odd"), http.StatusInternalServerError, "INTERNAL"},
		{errors.New("db down"), http.StatusInternalServerError, "INTERNAL"},
	}
	for _, tc := range cases {
		rec := httptest.NewRecorder()
		writeError(rec, httptest.NewRequest(http.MethodGet, "/", nil), discard, tc.err)
		assert.Equal(t, tc.status, rec.Code)
		var body api.Error
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
		assert.Equal(t, tc.code, body.Code)
		assert.Nil(t, body.Details)
	}

	rec := httptest.NewRecorder()
	writeError(rec, httptest.NewRequest(http.MethodGet, "/", nil), discard, errors.New("secret connection string"))
	assert.NotContains(t, rec.Body.String(), "secret", "internal details are not leaked")
}

func TestRecoverer(t *testing.T) {
	h := recoverer(discard)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic("boom") }))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.Contains(t, rec.Body.String(), "INTERNAL")

	abort := recoverer(discard)(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { panic(http.ErrAbortHandler) }))
	assert.PanicsWithValue(t, http.ErrAbortHandler, func() {
		abort.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
	})
}

func TestWsOrigins(t *testing.T) {
	assert.Equal(t, []string{"app.example.com", "localhost:5173"},
		wsOrigins([]string{"https://app.example.com", "http://localhost:5173", "not a url", "::"}))
	assert.Empty(t, wsOrigins(nil))
}

func TestReadiness(t *testing.T) {
	h := readiness(map[string]ReadyCheck{
		"postgres": func(context.Context) error { return nil },
		"redis":    func(context.Context) error { return errors.New("down") },
	})
	rec := httptest.NewRecorder()
	h(rec, httptest.NewRequest(http.MethodGet, "/readyz", nil))
	assert.Equal(t, http.StatusServiceUnavailable, rec.Code)
	assert.JSONEq(t, `{"postgres":"up","redis":"down"}`, rec.Body.String())
}

type limiterFunc func(ctx context.Context, key string) (bool, error)

func (f limiterFunc) Allow(ctx context.Context, key string) (bool, error) { return f(ctx, key) }

func TestRateLimitFailsOpen(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusNoContent) })
	broken := rateLimitAuth(limiterFunc(func(context.Context, string) (bool, error) { return false, errors.New("redis down") }), discard)(ok)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	req.RemoteAddr = "no-port"
	broken.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusNoContent, rec.Code, "an unavailable limiter lets requests through")

	var keys []string
	counting := rateLimitAuth(limiterFunc(func(_ context.Context, key string) (bool, error) {
		keys = append(keys, key)
		return true, nil
	}), discard)(ok)
	counting.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/v1/auth/login", nil))
	counting.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodPost, "/api/v1/projects", nil))
	assert.Empty(t, keys, "only POST to the auth endpoints is limited")
}

func TestAuthenticateIgnoresBadTokens(t *testing.T) {
	var seen domain.Actor
	h := authenticate(func(token string) (domain.Actor, error) {
		if token == "good" {
			return domain.Actor{Role: domain.RoleAdmin}, nil
		}
		return domain.Actor{}, domain.ErrUnauthorized
	})(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = actorFrom(r.Context()) }))

	for header, want := range map[string]domain.Role{"Bearer good": domain.RoleAdmin, "Bearer bad": "", "Basic good": "", "": ""} {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("Authorization", header)
		h.ServeHTTP(httptest.NewRecorder(), req)
		assert.Equal(t, want, seen.Role, header)
	}
}

func TestOptional(t *testing.T) {
	var absent api.ColumnPatch
	v, set := optional(absent.WipLimit)
	assert.Nil(t, v)
	assert.False(t, set)

	var null api.ColumnPatch
	require.NoError(t, json.Unmarshal([]byte(`{"wipLimit":null}`), &null))
	v, set = optional(null.WipLimit)
	assert.Nil(t, v)
	assert.True(t, set)

	var value api.ColumnPatch
	require.NoError(t, json.Unmarshal([]byte(`{"wipLimit":3}`), &value))
	v, set = optional(value.WipLimit)
	require.NotNil(t, v)
	assert.Equal(t, 3, *v)
	assert.True(t, set)
}

func TestRealIPTrustsOnlyPrivateProxies(t *testing.T) {
	var seen string
	h := realIP(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) { seen = r.RemoteAddr }))
	cases := []struct{ remote, header, want string }{
		{"172.18.0.5:1234", "203.0.113.7", "203.0.113.7:1234"},
		{"127.0.0.1:1234", "198.51.100.1", "198.51.100.1:1234"},
		{"203.0.113.9:1234", "10.0.0.1", "203.0.113.9:1234"},
		{"172.18.0.5:1234", "not-an-ip", "172.18.0.5:1234"},
		{"172.18.0.5:1234", "", "172.18.0.5:1234"},
		{"garbage", "203.0.113.7", "garbage"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = tc.remote
		req.Header.Set("X-Real-IP", tc.header)
		h.ServeHTTP(httptest.NewRecorder(), req)
		assert.Equal(t, tc.want, seen, "%s via %q", tc.remote, tc.header)
	}
}
