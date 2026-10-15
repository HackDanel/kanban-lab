package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"net/url"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/google/uuid"

	"github.com/HackDanel/kanban-lab/backend/internal/http/api"
	"github.com/HackDanel/kanban-lab/backend/internal/platform/metrics"
	"github.com/HackDanel/kanban-lab/backend/internal/realtime"
	"github.com/HackDanel/kanban-lab/backend/internal/service"
)

// ReadyCheck reports whether a dependency is reachable.
type ReadyCheck func(ctx context.Context) error

// Config collects what the router needs.
type Config struct {
	Service        *service.Service
	Hub            *realtime.Hub
	Metrics        *metrics.Metrics
	Limiter        RateLimiter
	Ready          map[string]ReadyCheck
	CORSOrigins    []string
	CookieSecure   bool
	UploadMaxBytes int64
	Log            *slog.Logger
}

// NewRouter builds the complete HTTP handler.
func NewRouter(cfg Config) (http.Handler, error) {
	validator, err := requestValidator(cfg.Log)
	if err != nil {
		return nil, err
	}

	r := chi.NewRouter()
	r.Use(realIP, middleware.RequestID, accessLog(cfg.Log), recoverer(cfg.Log), securityHeaders)
	if cfg.Metrics != nil {
		r.Use(cfg.Metrics.Middleware)
		r.Handle("/metrics", cfg.Metrics.Handler())
	}
	if len(cfg.CORSOrigins) > 0 {
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   cfg.CORSOrigins,
			AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE"},
			AllowedHeaders:   []string{"Authorization", "Content-Type", "X-CSRF-Token"},
			AllowCredentials: true,
			MaxAge:           600,
		}))
	}

	r.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
	r.Get("/readyz", readiness(cfg.Ready))
	r.Get("/docs", docsPage)
	r.Get("/docs/init.js", docsInit)
	r.Get("/docs/openapi.json", docsSpec)

	r.Get(apiBase+"/ws", cfg.Hub.Handler(func(ctx context.Context, token string, boardID uuid.UUID) error {
		actor, err := cfg.Service.Authenticate(token)
		if err != nil {
			return err
		}
		_, err = cfg.Service.GetBoard(ctx, actor, boardID)
		return err
	}, wsOrigins(cfg.CORSOrigins)).ServeHTTP)

	h := &handlers{svc: cfg.Service, cookies: cookies{secure: cfg.CookieSecure}}
	strict := api.NewStrictHandlerWithOptions(h, []api.StrictMiddlewareFunc{withExchange}, api.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			writeError(w, r, cfg.Log, validationError(err, http.StatusBadRequest))
		},
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			writeError(w, r, cfg.Log, err)
		},
	})
	r.Group(func(g chi.Router) {
		g.Use(limitBody(cfg.UploadMaxBytes), authenticate(cfg.Service.Authenticate), rateLimitAuth(cfg.Limiter, cfg.Log), validator)
		api.HandlerWithOptions(strict, api.ChiServerOptions{
			BaseURL: apiBase, BaseRouter: g,
			ErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
				writeError(w, r, cfg.Log, validationError(err, http.StatusBadRequest))
			},
		})
		// Unknown /api paths still go through the validator, which answers 404 in the API format.
		g.HandleFunc(apiBase+"/*", http.NotFound)
	})
	return r, nil
}

// wsOrigins lists the hosts allowed to open a WebSocket in addition to the
// server's own host, which the library always allows.
func wsOrigins(origins []string) []string {
	out := make([]string, 0, len(origins))
	for _, o := range origins {
		if u, err := url.Parse(o); err == nil && u.Host != "" {
			out = append(out, u.Host)
		}
	}
	return out
}

func readiness(checks map[string]ReadyCheck) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		status := map[string]string{}
		code := http.StatusOK
		for name, check := range checks {
			if err := check(ctx); err != nil {
				status[name] = "down"
				code = http.StatusServiceUnavailable
			} else {
				status[name] = "up"
			}
		}
		writeJSON(w, code, status)
	}
}
