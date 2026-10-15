package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand/v2"
	"mime/multipart"
	"net/http"
	"net/http/cookiejar"
	"net/textproto"
	"net/url"
	"testing"

	"github.com/google/uuid"
	openapi_types "github.com/oapi-codegen/runtime/types"
	"github.com/stretchr/testify/require"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/http/api"
	"github.com/HackDanel/kanban-lab/backend/internal/repository"
)

const base = "/api/v1"

// client is an API user with its own cookie jar.
type client struct {
	t     *testing.T
	srv   string
	http  *http.Client
	token string
	user  api.User
}

type response struct {
	status int
	header http.Header
	body   []byte
}

func newClient(t *testing.T) *client {
	t.Helper()
	jar, err := cookiejar.New(nil)
	require.NoError(t, err)
	return &client{t: t, srv: env.srv.URL, http: &http.Client{Jar: jar}}
}

func (c *client) send(method, path string, body io.Reader, contentType string, headers map[string]string) response {
	c.t.Helper()
	req, err := http.NewRequest(method, c.srv+base+path, body)
	require.NoError(c.t, err)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.token != "" {
		req.Header.Set("Authorization", "Bearer "+c.token)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := c.http.Do(req)
	require.NoError(c.t, err)
	defer func() { _ = resp.Body.Close() }()
	data, err := io.ReadAll(resp.Body)
	require.NoError(c.t, err)
	return response{status: resp.StatusCode, header: resp.Header, body: data}
}

// do sends body (if any) as JSON.
func (c *client) do(method, path string, body any) response {
	c.t.Helper()
	if body == nil {
		return c.send(method, path, nil, "", nil)
	}
	data, err := json.Marshal(body)
	require.NoError(c.t, err)
	return c.send(method, path, bytes.NewReader(data), "application/json", nil)
}

func (c *client) upload(cardID uuid.UUID, filename, contentType string, content []byte) response {
	c.t.Helper()
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="file"; filename=%q`, filename))
	h.Set("Content-Type", contentType)
	part, err := w.CreatePart(h)
	require.NoError(c.t, err)
	_, err = part.Write(content)
	require.NoError(c.t, err)
	require.NoError(c.t, w.Close())
	return c.send(http.MethodPost, fmt.Sprintf("/cards/%s/attachments", cardID), &buf, w.FormDataContentType(), nil)
}

func (c *client) cookie(name string) string {
	for _, ck := range c.http.Jar.Cookies(mustURL(c.srv + base + "/auth/")) {
		if ck.Name == name {
			return ck.Value
		}
	}
	return ""
}

// decode asserts the status and unmarshals the body.
func decode[T any](t *testing.T, r response, status int) T {
	t.Helper()
	require.Equal(t, status, r.status, "body: %s", r.body)
	var v T
	require.NoError(t, json.Unmarshal(r.body, &v), "body: %s", r.body)
	return v
}

// expectError asserts an API error response and returns it.
func expectError(t *testing.T, r response, status int, code string) api.Error {
	t.Helper()
	e := decode[api.Error](t, r, status)
	require.Equal(t, code, e.Code, "message: %s", e.Message)
	return e
}

func fieldErrors(e api.Error) map[string]string {
	out := map[string]string{}
	if e.Details != nil {
		for _, d := range *e.Details {
			out[d.Field] = d.Message
		}
	}
	return out
}

func uniqueEmail() string { return fmt.Sprintf("user-%s@example.com", uuid.NewString()[:8]) }

// uniqueKey returns a fresh project key.
func uniqueKey() string {
	const letters = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	b := make([]byte, 8)
	for i := range b {
		b[i] = letters[rand.IntN(len(letters))]
	}
	return string(b)
}

// login signs in an existing account.
func login(t *testing.T, email, password string) *client {
	t.Helper()
	c := newClient(t)
	auth := decode[api.AuthResponse](t, c.do(http.MethodPost, "/auth/login", api.LoginRequest{Email: email_(email), Password: password}), http.StatusOK)
	c.token, c.user = auth.AccessToken, auth.User
	return c
}

// newUser registers an account with the given role and signs it in.
func newUser(t *testing.T, role domain.Role) *client {
	t.Helper()
	email := uniqueEmail()
	anon := newClient(t)
	user := decode[api.User](t, anon.do(http.MethodPost, "/auth/register", api.RegisterRequest{
		Email: email_(email), Name: "User " + string(role), Password: "password1",
	}), http.StatusCreated)
	if role != domain.RoleDeveloper {
		_, err := env.store.SetUserRole(context.Background(), repository.SetUserRoleParams{ID: user.Id, Role: string(role)})
		require.NoError(t, err)
	}
	return login(t, email, "password1")
}

func admin(t *testing.T) *client { return login(t, adminEmail, adminPass) }

// fixture is a project with a board, owned by a manager.
type fixture struct {
	owner   *client
	project api.Project
	board   api.BoardDetail
	todo    api.Column
	doing   api.Column
	done    api.Column
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	owner := newUser(t, domain.RoleManager)
	project := decode[api.Project](t, owner.do(http.MethodPost, "/projects", api.ProjectInput{Key: uniqueKey(), Name: "Project"}), http.StatusCreated)
	board := decode[api.BoardDetail](t, owner.do(http.MethodPost, "/projects/"+project.Id.String()+"/boards", api.BoardInput{Name: "Board"}), http.StatusCreated)
	require.Len(t, board.Columns, 3)
	return fixture{owner: owner, project: project, board: board, todo: board.Columns[0], doing: board.Columns[1], done: board.Columns[2]}
}

// member adds a fresh user with the given role to the project.
func (f fixture) member(t *testing.T, role domain.Role) *client {
	t.Helper()
	c := newUser(t, role)
	r := f.owner.do(http.MethodPost, "/projects/"+f.project.Id.String()+"/members", api.AddMemberRequest{UserId: c.user.Id})
	require.Equal(t, http.StatusNoContent, r.status, "body: %s", r.body)
	return c
}

func (f fixture) card(t *testing.T, c *client, column api.Column, title string) api.Card {
	t.Helper()
	return decode[api.Card](t, c.do(http.MethodPost, "/columns/"+column.Id.String()+"/cards", map[string]any{"title": title}), http.StatusCreated)
}

func (f fixture) listCards(t *testing.T, c *client, query string) api.CardPage {
	t.Helper()
	path := "/boards/" + f.board.Id.String() + "/cards"
	if query != "" {
		path += "?" + query
	}
	return decode[api.CardPage](t, c.do(http.MethodGet, path, nil), http.StatusOK)
}

func titles(cards []api.Card) []string {
	out := make([]string, len(cards))
	for i, c := range cards {
		out[i] = c.Title
	}
	return out
}

func email_(s string) openapi_types.Email { return openapi_types.Email(s) }

func mustURL(raw string) *url.URL {
	u, err := url.Parse(raw)
	if err != nil {
		panic(err)
	}
	return u
}
