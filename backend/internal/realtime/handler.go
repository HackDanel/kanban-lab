package realtime

import (
	"context"
	"net/http"
	"time"

	"github.com/coder/websocket"
	"github.com/google/uuid"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
)

const writeTimeout = 10 * time.Second

// Authorizer checks the token and that the user may watch the board.
type Authorizer func(ctx context.Context, token string, boardID uuid.UUID) error

// Handler upgrades GET /api/v1/ws?boardId=...&token=... to a WebSocket that
// streams the board's events. Browsers cannot set headers on a WebSocket
// handshake, so the access token travels in the query string.
func (h *Hub) Handler(authorize Authorizer, originPatterns []string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		boardID, err := uuid.Parse(r.URL.Query().Get("boardId"))
		if err != nil {
			http.Error(w, "boardId is required", http.StatusBadRequest)
			return
		}
		if err := authorize(r.Context(), r.URL.Query().Get("token"), boardID); err != nil {
			status := http.StatusInternalServerError
			switch domain.KindOf(err) {
			case domain.KindUnauthorized:
				status = http.StatusUnauthorized
			case domain.KindForbidden:
				status = http.StatusForbidden
			case domain.KindNotFound:
				status = http.StatusNotFound
			}
			http.Error(w, http.StatusText(status), status)
			return
		}

		conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{OriginPatterns: originPatterns})
		if err != nil {
			return // Accept has already written the response
		}
		c := h.subscribe(boardID)
		defer h.unsubscribe(boardID, c)

		// Clients only listen; CloseRead handles their close frames and pings.
		ctx := conn.CloseRead(r.Context())
		for {
			select {
			case <-ctx.Done():
				_ = conn.Close(websocket.StatusNormalClosure, "")
				return
			case data, ok := <-c.send:
				if !ok {
					_ = conn.Close(websocket.StatusTryAgainLater, "too slow")
					return
				}
				writeCtx, cancel := context.WithTimeout(ctx, writeTimeout)
				err := conn.Write(writeCtx, websocket.MessageText, data)
				cancel()
				if err != nil {
					return
				}
			}
		}
	})
}
