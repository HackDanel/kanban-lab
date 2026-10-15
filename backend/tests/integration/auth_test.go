package integration

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HackDanel/kanban-lab/backend/internal/http/api"
)

func TestRegister(t *testing.T) {
	c := newClient(t)
	email := uniqueEmail()

	// TC-01
	user := decode[api.User](t, c.do(http.MethodPost, "/auth/register", map[string]string{
		"email": "  " + strings.ToUpper(email) + " ", "name": " Иван ", "password": "password1",
	}), http.StatusCreated)
	assert.Equal(t, email, string(user.Email), "email is trimmed and lower-cased")
	assert.Equal(t, "Иван", user.Name)
	assert.Equal(t, api.Developer, user.Role, "new accounts are developers")

	// TC-10
	expectError(t, c.do(http.MethodPost, "/auth/register", map[string]string{
		"email": email, "name": "Again", "password": "password1",
	}), http.StatusConflict, "CONFLICT")
}

func TestRegisterValidation(t *testing.T) {
	c := newClient(t)
	cases := map[string]struct {
		body  map[string]string
		field string
	}{
		"bad email":         {map[string]string{"email": "not-an-email", "name": "A", "password": "password1"}, "email"},
		"short password":    {map[string]string{"email": uniqueEmail(), "name": "A", "password": "1234567"}, "password"},
		"too long password": {map[string]string{"email": uniqueEmail(), "name": "A", "password": strings.Repeat("я", 37)}, "password"},
		"blank name":        {map[string]string{"email": uniqueEmail(), "name": "   ", "password": "password1"}, "name"},
		"missing name":      {map[string]string{"email": uniqueEmail(), "password": "password1"}, ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			e := expectError(t, c.do(http.MethodPost, "/auth/register", tc.body), http.StatusBadRequest, "VALIDATION_ERROR")
			if tc.field != "" {
				assert.Contains(t, fieldErrors(e), tc.field)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	user := newUser(t, "developer")
	assert.NotEmpty(t, user.token)
	assert.NotEmpty(t, user.cookie("refresh_token"))
	assert.NotEmpty(t, user.cookie("csrf_token"))

	me := decode[api.User](t, user.do(http.MethodGet, "/auth/me", nil), http.StatusOK)
	assert.Equal(t, user.user.Id, me.Id)

	// TC-09: a wrong password and an unknown email look the same.
	c := newClient(t)
	wrong := expectError(t, c.do(http.MethodPost, "/auth/login", map[string]string{"email": string(user.user.Email), "password": "wrong-password"}), http.StatusUnauthorized, "UNAUTHORIZED")
	unknown := expectError(t, c.do(http.MethodPost, "/auth/login", map[string]string{"email": uniqueEmail(), "password": "wrong-password"}), http.StatusUnauthorized, "UNAUTHORIZED")
	assert.Equal(t, wrong.Message, unknown.Message)
}

func TestLoginSetsHardenedCookies(t *testing.T) {
	user := newUser(t, "developer")
	r := user.do(http.MethodPost, "/auth/login", map[string]string{"email": string(user.user.Email), "password": "password1"})
	require.Equal(t, http.StatusOK, r.status)

	cookies := map[string]*http.Cookie{}
	for _, ck := range (&http.Response{Header: r.header}).Cookies() {
		cookies[ck.Name] = ck
	}
	refresh, csrf := cookies["refresh_token"], cookies["csrf_token"]
	require.NotNil(t, refresh)
	require.NotNil(t, csrf)
	assert.True(t, refresh.HttpOnly, "refresh token is not readable by scripts")
	assert.Equal(t, "/api/v1/auth", refresh.Path)
	assert.Equal(t, http.SameSiteStrictMode, refresh.SameSite)
	assert.False(t, csrf.HttpOnly, "the page must read the CSRF token")
	assert.Equal(t, http.SameSiteStrictMode, csrf.SameSite)
}

func TestProtectedEndpointsRequireToken(t *testing.T) {
	c := newClient(t)
	// TC-13
	expectError(t, c.do(http.MethodGet, "/projects", nil), http.StatusUnauthorized, "UNAUTHORIZED")
	c.token = "garbage"
	expectError(t, c.do(http.MethodGet, "/auth/me", nil), http.StatusUnauthorized, "UNAUTHORIZED")
}

func TestRefreshRotatesTokens(t *testing.T) {
	user := newUser(t, "developer")
	csrf := user.cookie("csrf_token")
	firstRefresh := user.cookie("refresh_token")

	refreshed := decode[api.AuthResponse](t, user.send(http.MethodPost, "/auth/refresh", nil, "", map[string]string{"X-CSRF-Token": csrf}), http.StatusOK)
	assert.Equal(t, user.user.Id, refreshed.User.Id)
	assert.NotEqual(t, firstRefresh, user.cookie("refresh_token"), "refresh token is rotated")

	user.token = refreshed.AccessToken
	decode[api.User](t, user.do(http.MethodGet, "/auth/me", nil), http.StatusOK)
}

func TestRefreshTokenReuseRevokesSession(t *testing.T) {
	user := newUser(t, "developer")
	stolen := user.cookie("refresh_token")
	csrf := user.cookie("csrf_token")

	// The legitimate client refreshes first...
	decode[api.AuthResponse](t, user.send(http.MethodPost, "/auth/refresh", nil, "", map[string]string{"X-CSRF-Token": csrf}), http.StatusOK)

	// ...then the stolen, already used token is presented.
	attacker := newClient(t)
	r := attacker.send(http.MethodPost, "/auth/refresh", nil, "", map[string]string{
		"X-CSRF-Token": "x", "Cookie": "refresh_token=" + stolen + "; csrf_token=x",
	})
	expectError(t, r, http.StatusUnauthorized, "UNAUTHORIZED")

	// The whole session family is revoked, including the legitimate client's new token.
	expectError(t, user.send(http.MethodPost, "/auth/refresh", nil, "", map[string]string{"X-CSRF-Token": user.cookie("csrf_token")}), http.StatusUnauthorized, "UNAUTHORIZED")
}

func TestRefreshRequiresCSRFToken(t *testing.T) {
	user := newUser(t, "developer")
	expectError(t, user.send(http.MethodPost, "/auth/refresh", nil, "", nil), http.StatusBadRequest, "VALIDATION_ERROR")
	expectError(t, user.send(http.MethodPost, "/auth/refresh", nil, "", map[string]string{"X-CSRF-Token": "forged"}), http.StatusForbidden, "FORBIDDEN")
}

func TestRefreshWithoutCookie(t *testing.T) {
	c := newClient(t)
	r := c.send(http.MethodPost, "/auth/refresh", nil, "", map[string]string{"X-CSRF-Token": "x", "Cookie": "csrf_token=x"})
	expectError(t, r, http.StatusUnauthorized, "UNAUTHORIZED")
}

func TestExpiredRefreshToken(t *testing.T) {
	user := newUser(t, "developer")
	_, err := pool.Exec(context.Background(), `UPDATE refresh_tokens SET expires_at = $1 WHERE user_id = $2`, time.Now().Add(-time.Minute), user.user.Id)
	require.NoError(t, err)

	r := user.send(http.MethodPost, "/auth/refresh", nil, "", map[string]string{"X-CSRF-Token": user.cookie("csrf_token")})
	expectError(t, r, http.StatusUnauthorized, "UNAUTHORIZED")
	assert.Empty(t, user.cookie("refresh_token"), "cookies are cleared")

	require.NoError(t, env.svc.PurgeExpiredTokens(context.Background()))
	var left int
	require.NoError(t, pool.QueryRow(context.Background(), `SELECT count(*) FROM refresh_tokens WHERE user_id = $1`, user.user.Id).Scan(&left))
	assert.Zero(t, left)
}

func TestLogout(t *testing.T) {
	user := newUser(t, "developer")
	csrf := user.cookie("csrf_token")

	expectError(t, user.send(http.MethodPost, "/auth/logout", nil, "", map[string]string{"X-CSRF-Token": "forged"}), http.StatusForbidden, "FORBIDDEN")

	r := user.send(http.MethodPost, "/auth/logout", nil, "", map[string]string{"X-CSRF-Token": csrf})
	require.Equal(t, http.StatusNoContent, r.status)
	assert.Empty(t, user.cookie("refresh_token"))

	// TC-12: the refresh token no longer works even if it was copied before logout.
	fresh := newClient(t)
	expectError(t, fresh.send(http.MethodPost, "/auth/refresh", nil, "", map[string]string{"X-CSRF-Token": "x", "Cookie": "csrf_token=x"}), http.StatusUnauthorized, "UNAUTHORIZED")

	// Logging out twice is harmless.
	again := newClient(t)
	r = again.send(http.MethodPost, "/auth/logout", nil, "", map[string]string{"X-CSRF-Token": "x", "Cookie": "csrf_token=x; refresh_token=unknown"})
	assert.Equal(t, http.StatusNoContent, r.status)
}

// TC-29
func TestAuthRateLimit(t *testing.T) {
	limited, err := newStack(t.Context(), t.TempDir(), 3)
	require.NoError(t, err)
	defer limited.srv.Close()

	c := newClient(t)
	c.srv = limited.srv.URL
	body := map[string]string{"email": uniqueEmail(), "password": "wrong-password"}
	for range 3 {
		expectError(t, c.do(http.MethodPost, "/auth/login", body), http.StatusUnauthorized, "UNAUTHORIZED")
	}
	r := c.do(http.MethodPost, "/auth/login", body)
	expectError(t, r, http.StatusTooManyRequests, "RATE_LIMITED")
	assert.Equal(t, "60", r.header.Get("Retry-After"))
}

func TestEnsureAdminIsIdempotent(t *testing.T) {
	require.NoError(t, env.svc.EnsureAdmin(context.Background(), adminEmail, "other-password"))
	admin(t) // the original password still works
}
