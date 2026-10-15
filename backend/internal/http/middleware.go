package httpserver

import (
	"context"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5/middleware"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
)

// RateLimiter decides whether a request identified by key may proceed.
type RateLimiter interface {
	Allow(ctx context.Context, key string) (bool, error)
}

// accessLog writes one line per request. The query string is left out because
// the WebSocket endpoint carries the access token in it.
func accessLog(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)
			level := slog.LevelInfo
			if ww.Status() >= 500 {
				level = slog.LevelError
			}
			log.Log(r.Context(), level, "request",
				"method", r.Method, "path", r.URL.Path, "status", ww.Status(),
				"bytes", ww.BytesWritten(), "duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()), "remote", r.RemoteAddr)
		})
	}
}

// recoverer turns a panic into a 500 response in the API error format.
func recoverer(log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if v := recover(); v != nil {
					if v == http.ErrAbortHandler {
						panic(v)
					}
					log.ErrorContext(r.Context(), "panic", "value", v, "path", r.URL.Path)
					writeJSON(w, http.StatusInternalServerError, map[string]string{
						"code": domain.KindInternal.Code(), "message": "Внутренняя ошибка сервера",
					})
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// securityHeaders sets defensive headers on every API response.
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cross-Origin-Resource-Policy", "same-origin")
		if !strings.HasPrefix(r.URL.Path, "/docs") {
			h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
			h.Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

// limitBody caps request bodies: uploads at the attachment limit plus room
// for the multipart framing, everything else at 1 MiB.
func limitBody(maxUpload int64) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			limit := int64(1 << 20)
			if strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/") {
				limit = maxUpload + 1<<20
			}
			r.Body = http.MaxBytesReader(w, r.Body, limit)
			next.ServeHTTP(w, r)
		})
	}
}

// authenticate puts the actor from a valid Bearer token into the context.
// Requests without a valid token stay anonymous; the OpenAPI validator then
// rejects them on operations that require authentication.
func authenticate(verify func(string) (domain.Actor, error)) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer "); ok {
				if actor, err := verify(strings.TrimSpace(token)); err == nil {
					r = r.WithContext(withActor(r.Context(), actor))
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

var rateLimitedPaths = map[string]bool{
	"/api/v1/auth/login":    true,
	"/api/v1/auth/register": true,
}

// rateLimitAuth slows down password guessing and mass registration per client
// address. If Redis is unavailable the request is let through: a degraded
// limiter must not lock everybody out.
func rateLimitAuth(limiter RateLimiter, log *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost || !rateLimitedPaths[r.URL.Path] {
				next.ServeHTTP(w, r)
				return
			}
			ip, _, err := net.SplitHostPort(r.RemoteAddr)
			if err != nil {
				ip = r.RemoteAddr
			}
			ok, err := limiter.Allow(r.Context(), r.URL.Path+":"+ip)
			if err != nil {
				log.WarnContext(r.Context(), "rate limiter unavailable", "err", err)
			} else if !ok {
				w.Header().Set("Retry-After", "60")
				writeError(w, r, log, domain.E(domain.KindRateLimited, "Слишком много попыток. Повторите через минуту"))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
