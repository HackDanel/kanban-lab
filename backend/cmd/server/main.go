// Command server runs the Kanban API.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/HackDanel/kanban-lab/backend/internal/auth"
	"github.com/HackDanel/kanban-lab/backend/internal/config"
	httpserver "github.com/HackDanel/kanban-lab/backend/internal/http"
	"github.com/HackDanel/kanban-lab/backend/internal/platform/metrics"
	"github.com/HackDanel/kanban-lab/backend/internal/platform/postgres"
	"github.com/HackDanel/kanban-lab/backend/internal/platform/ratelimit"
	"github.com/HackDanel/kanban-lab/backend/internal/platform/storage"
	"github.com/HackDanel/kanban-lab/backend/internal/realtime"
	"github.com/HackDanel/kanban-lab/backend/internal/repository"
	"github.com/HackDanel/kanban-lab/backend/internal/service"
)

func main() {
	// The runtime image has no shell or curl, so the container health check
	// calls the binary itself.
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(healthcheck())
	}
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if err := run(log); err != nil {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}

func run(log *slog.Logger) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.Connect(ctx, cfg.DatabaseURL, cfg.DBMaxConns)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err := postgres.Migrate(ctx, pool, log); err != nil {
		return err
	}

	redisOpts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return err
	}
	rdb := redis.NewClient(redisOpts)
	defer func() { _ = rdb.Close() }()

	hasher, err := auth.NewHasher(cfg.BcryptCost)
	if err != nil {
		return err
	}
	tokens, err := auth.NewTokens(cfg.JWTSecret, cfg.AccessTTL, time.Now)
	if err != nil {
		return err
	}
	files, err := storage.NewDisk(cfg.UploadDir)
	if err != nil {
		return err
	}
	hub := realtime.NewHub(rdb, log)

	svc := service.New(service.Deps{
		Store: repository.NewStore(pool), Hasher: hasher, Tokens: tokens, RefreshTTL: cfg.RefreshTTL,
		Now: time.Now, Events: hub, Files: files, UploadMaxBytes: cfg.UploadMaxBytes, Log: log,
	})
	if cfg.AdminEmail != "" {
		if err := svc.EnsureAdmin(ctx, cfg.AdminEmail, cfg.AdminPassword); err != nil {
			return err
		}
	}

	handler, err := httpserver.NewRouter(httpserver.Config{
		Service:        svc,
		Hub:            hub,
		Metrics:        metrics.New(pool, hub.Clients),
		Limiter:        ratelimit.New(rdb, cfg.AuthRateLimit, time.Minute, time.Now),
		CORSOrigins:    cfg.CORSOrigins,
		CookieSecure:   cfg.CookieSecure,
		UploadMaxBytes: cfg.UploadMaxBytes,
		Log:            log,
		Ready: map[string]httpserver.ReadyCheck{
			"postgres": pool.Ping,
			"redis":    func(ctx context.Context) error { return rdb.Ping(ctx).Err() },
		},
	})
	if err != nil {
		return err
	}

	go hub.Run(ctx)
	go purgeTokens(ctx, svc, log)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		log.Info("listening", "addr", cfg.HTTPAddr, "env", cfg.Env)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}
	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func purgeTokens(ctx context.Context, svc *service.Service, log *slog.Logger) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := svc.PurgeExpiredTokens(ctx); err != nil {
				log.Warn("purge expired refresh tokens", "err", err)
			}
		}
	}
}

func healthcheck() int {
	addr := os.Getenv("HTTP_ADDR")
	if addr == "" || addr[0] == ':' {
		addr = "localhost" + addr
	}
	if addr == "localhost" {
		addr = "localhost:8080"
	}
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + addr + "/healthz")
	if err != nil {
		return 1
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
