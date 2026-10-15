package realtime

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/service"
)

func newHub() *Hub {
	return NewHub(nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

func dial(t *testing.T, srv *httptest.Server, query string) *websocket.Conn {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	conn, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/?"+query, nil)
	require.NoError(t, err)
	t.Cleanup(func() { conn.CloseNow() })
	return conn
}

func waitForClients(t *testing.T, h *Hub, n float64) {
	t.Helper()
	require.Eventually(t, func() bool { return h.Clients() == n }, 5*time.Second, 10*time.Millisecond)
}

func TestHubDeliversEventsToWatchersOfTheBoard(t *testing.T) {
	hub := newHub()
	board, other := uuid.New(), uuid.New()
	allow := func(context.Context, string, uuid.UUID) error { return nil }
	srv := httptest.NewServer(hub.Handler(allow, nil))
	defer srv.Close()

	watcher := dial(t, srv, "boardId="+board.String()+"&token=t")
	bystander := dial(t, srv, "boardId="+other.String()+"&token=t")
	waitForClients(t, hub, 2)

	hub.Publish(context.Background(), service.Event{Type: service.EventCardMoved, BoardID: board, Payload: map[string]string{"cardId": "c1"}})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, data, err := watcher.Read(ctx)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(data, &got))
	assert.Equal(t, "card.moved", got["type"])
	assert.Equal(t, board.String(), got["boardId"])

	watcher.Close(websocket.StatusNormalClosure, "")
	waitForClients(t, hub, 1)

	// A read that times out closes the connection, so this check goes last.
	quiet, cancelQuiet := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancelQuiet()
	_, _, err = bystander.Read(quiet)
	assert.ErrorIs(t, err, context.DeadlineExceeded, "clients of another board receive nothing")
}

func TestHandlerRejectsBadRequests(t *testing.T) {
	hub := newHub()
	board := uuid.New()
	authorize := func(_ context.Context, token string, _ uuid.UUID) error {
		switch token {
		case "ok":
			return nil
		case "member-of-nothing":
			return domain.ErrForbidden
		case "gone":
			return domain.NotFound("Доска")
		case "broken":
			return assert.AnError
		}
		return domain.ErrUnauthorized
	}
	srv := httptest.NewServer(hub.Handler(authorize, nil))
	defer srv.Close()

	cases := map[string]int{
		"token=ok":                  http.StatusBadRequest,
		"boardId=nope&token=ok":     http.StatusBadRequest,
		"boardId=" + board.String(): http.StatusUnauthorized,
		"boardId=" + board.String() + "&token=member-of-nothing": http.StatusForbidden,
		"boardId=" + board.String() + "&token=gone":              http.StatusNotFound,
		"boardId=" + board.String() + "&token=broken":            http.StatusInternalServerError,
	}
	for query, status := range cases {
		resp, err := http.Get(srv.URL + "/?" + query)
		require.NoError(t, err)
		_ = resp.Body.Close()
		assert.Equal(t, status, resp.StatusCode, query)
	}
	assert.Zero(t, hub.Clients())
}

func TestHubDropsSlowClients(t *testing.T) {
	hub := newHub()
	board := uuid.New()
	c := hub.subscribe(board)

	for i := 0; i <= clientBuffer; i++ {
		hub.Publish(context.Background(), service.Event{Type: "x", BoardID: board})
	}

	assert.True(t, c.closed.Load(), "a client with a full buffer is dropped")
	assert.Zero(t, hub.Clients())
	hub.unsubscribe(board, c) // second unsubscribe is harmless
	assert.Zero(t, hub.Clients())
}

func TestRunWithoutRedisStopsOnCancel(t *testing.T) {
	hub := newHub()
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() { hub.Run(ctx); close(done) }()
	cancel()
	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("Run did not stop")
	}
}
