package httpserver

import (
	"context"
	"net/http"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/http/api"
)

type actorKey struct{}

type exchangeKey struct{}

// exchange gives strict handlers access to the raw request and response,
// which they need for cookies.
type exchange struct {
	w http.ResponseWriter
	r *http.Request
}

func withActor(ctx context.Context, actor domain.Actor) context.Context {
	return context.WithValue(ctx, actorKey{}, actor)
}

// actorFrom returns the authenticated user, or the zero Actor for anonymous requests.
func actorFrom(ctx context.Context) domain.Actor {
	actor, _ := ctx.Value(actorKey{}).(domain.Actor)
	return actor
}

func exchangeFrom(ctx context.Context) exchange {
	ex, _ := ctx.Value(exchangeKey{}).(exchange)
	return ex
}

// withExchange is a strict-server middleware storing the request and response in the context.
func withExchange(f api.StrictHandlerFunc, _ string) api.StrictHandlerFunc {
	return func(ctx context.Context, w http.ResponseWriter, r *http.Request, request any) (any, error) {
		return f(context.WithValue(ctx, exchangeKey{}, exchange{w: w, r: r}), w, r, request)
	}
}
