// Package realtime pushes board events to browsers over WebSocket. Events
// travel through a Redis channel so that every server instance delivers them
// to its own clients.
package realtime

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"github.com/HackDanel/kanban-lab/backend/internal/service"
)

const channel = "kanban:events"

// clientBuffer is how many events may queue for a slow client before it is dropped.
const clientBuffer = 32

type client struct {
	send chan []byte
	// closed is set once the hub stops delivering to the client.
	closed atomic.Bool
}

// Hub tracks connected clients per board.
type Hub struct {
	rdb *redis.Client
	log *slog.Logger

	mu      sync.RWMutex
	boards  map[uuid.UUID]map[*client]struct{}
	clients atomic.Int64
}

// NewHub returns a Hub. With a nil Redis client events are delivered only locally.
func NewHub(rdb *redis.Client, log *slog.Logger) *Hub {
	return &Hub{rdb: rdb, log: log, boards: make(map[uuid.UUID]map[*client]struct{})}
}

// Clients reports the number of connected clients.
func (h *Hub) Clients() float64 { return float64(h.clients.Load()) }

// Publish implements service.Publisher.
func (h *Hub) Publish(ctx context.Context, event service.Event) {
	data, err := json.Marshal(event)
	if err != nil {
		h.log.Error("marshal event", "err", err)
		return
	}
	if h.rdb == nil {
		h.deliver(event.BoardID, data)
		return
	}
	if err := h.rdb.Publish(ctx, channel, data).Err(); err != nil {
		h.log.Warn("publish event", "err", err)
	}
}

// Run relays events from Redis to local clients until ctx is cancelled.
func (h *Hub) Run(ctx context.Context) {
	if h.rdb == nil {
		<-ctx.Done()
		return
	}
	sub := h.rdb.Subscribe(ctx, channel)
	defer func() { _ = sub.Close() }()
	messages := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-messages:
			if !ok {
				return
			}
			var event struct {
				BoardID uuid.UUID `json:"boardId"`
			}
			if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
				h.log.Warn("malformed event", "err", err)
				continue
			}
			h.deliver(event.BoardID, []byte(msg.Payload))
		}
	}
}

func (h *Hub) deliver(boardID uuid.UUID, data []byte) {
	h.mu.RLock()
	var slow []*client
	for c := range h.boards[boardID] {
		select {
		case c.send <- data:
		default:
			slow = append(slow, c)
		}
	}
	h.mu.RUnlock()
	// A client that cannot keep up is disconnected; it reloads the board on reconnect.
	for _, c := range slow {
		h.unsubscribe(boardID, c)
	}
}

func (h *Hub) subscribe(boardID uuid.UUID) *client {
	c := &client{send: make(chan []byte, clientBuffer)}
	h.mu.Lock()
	if h.boards[boardID] == nil {
		h.boards[boardID] = make(map[*client]struct{})
	}
	h.boards[boardID][c] = struct{}{}
	h.mu.Unlock()
	h.clients.Add(1)
	return c
}

func (h *Hub) unsubscribe(boardID uuid.UUID, c *client) {
	if !c.closed.CompareAndSwap(false, true) {
		return
	}
	h.mu.Lock()
	delete(h.boards[boardID], c)
	if len(h.boards[boardID]) == 0 {
		delete(h.boards, boardID)
	}
	h.mu.Unlock()
	close(c.send)
	h.clients.Add(-1)
}
