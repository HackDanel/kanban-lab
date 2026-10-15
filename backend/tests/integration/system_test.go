package integration

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/http/api"
)

func get(t *testing.T, path string) (*http.Response, string) {
	t.Helper()
	resp, err := http.Get(env.srv.URL + path)
	require.NoError(t, err)
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)
	return resp, string(body)
}

func TestHealthAndReadiness(t *testing.T) {
	resp, _ := get(t, "/healthz")
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	resp, body := get(t, "/readyz")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.JSONEq(t, `{"postgres":"up","redis":"up"}`, body)
}

func TestDocsAndMetrics(t *testing.T) {
	resp, body := get(t, "/docs")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, "swagger-ui")
	assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "script-src 'self' https://unpkg.com")

	resp, body = get(t, "/docs/init.js")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, "/docs/openapi.json")

	resp, body = get(t, "/docs/openapi.json")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	var spec map[string]any
	require.NoError(t, json.Unmarshal([]byte(body), &spec))
	assert.Equal(t, "Kanban Lab API", spec["info"].(map[string]any)["title"])

	newClient(t).do(http.MethodGet, "/projects", nil)
	resp, body = get(t, "/metrics")
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, body, `http_requests_total{method="GET",route="/api/v1/projects",status="401"}`)
	assert.Contains(t, body, "db_pool_max_conns")
	assert.Contains(t, body, "websocket_clients")
}

func TestUnknownRoutesAndHeaders(t *testing.T) {
	c := newUser(t, domain.RoleDeveloper)
	r := c.do(http.MethodGet, "/no-such-endpoint", nil)
	expectError(t, r, http.StatusNotFound, "NOT_FOUND")
	assert.Equal(t, "default-src 'none'; frame-ancestors 'none'", r.header.Get("Content-Security-Policy"))
	assert.Equal(t, "DENY", r.header.Get("X-Frame-Options"))
	assert.Equal(t, "no-store", r.header.Get("Cache-Control"))

	expectError(t, c.do(http.MethodPut, "/projects", nil), http.StatusNotFound, "NOT_FOUND")
	expectError(t, c.send(http.MethodPost, "/projects", strings.NewReader("{not json"), "application/json", nil), http.StatusBadRequest, "VALIDATION_ERROR")
	expectError(t, c.send(http.MethodPost, "/projects", strings.NewReader("key=A"), "text/plain", nil), http.StatusBadRequest, "VALIDATION_ERROR")

	huge := `{"key":"AB","name":"` + strings.Repeat("x", 2<<20) + `"}`
	expectError(t, c.send(http.MethodPost, "/projects", strings.NewReader(huge), "application/json", nil), http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE")
}

// TC-27
func TestRealtimeEvents(t *testing.T) {
	f := newFixture(t)
	watcher := f.member(t, domain.RoleDeveloper)
	wsURL := "ws" + strings.TrimPrefix(env.srv.URL, "http") + base + "/ws?boardId=" + f.board.Id.String()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	_, resp, err := websocket.Dial(ctx, wsURL+"&token=bad", nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	outsider := newUser(t, domain.RoleDeveloper)
	_, resp, err = websocket.Dial(ctx, wsURL+"&token="+outsider.token, nil)
	require.Error(t, err)
	assert.Equal(t, http.StatusForbidden, resp.StatusCode)

	conn, _, err := websocket.Dial(ctx, wsURL+"&token="+watcher.token, nil)
	require.NoError(t, err)
	defer func() { _ = conn.CloseNow() }()
	require.Eventually(t, func() bool { return env.hub.Clients() >= 1 }, 5*time.Second, 10*time.Millisecond)

	card := f.card(t, f.owner, f.todo, "live")
	decode[api.Card](t, f.owner.do(http.MethodPost, cardPath(card, "/move"), map[string]any{"columnId": f.doing.Id}), http.StatusOK)

	var types []string
	for len(types) < 2 {
		_, data, err := conn.Read(ctx)
		require.NoError(t, err)
		var event struct {
			Type    string            `json:"type"`
			BoardID string            `json:"boardId"`
			Payload map[string]string `json:"payload"`
		}
		require.NoError(t, json.Unmarshal(data, &event))
		assert.Equal(t, f.board.Id.String(), event.BoardID)
		assert.Equal(t, card.Id.String(), event.Payload["cardId"])
		types = append(types, event.Type)
	}
	assert.Equal(t, []string{"card.created", "card.moved"}, types)
}
