// Package integration tests the API end to end: real HTTP requests through the
// router, the service layer and PostgreSQL and Redis running in containers.
package integration

import (
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	tcredis "github.com/testcontainers/testcontainers-go/modules/redis"
	"golang.org/x/crypto/bcrypt"

	"github.com/HackDanel/kanban-lab/backend/internal/auth"
	httpserver "github.com/HackDanel/kanban-lab/backend/internal/http"
	"github.com/HackDanel/kanban-lab/backend/internal/platform/metrics"
	"github.com/HackDanel/kanban-lab/backend/internal/platform/postgres"
	"github.com/HackDanel/kanban-lab/backend/internal/platform/ratelimit"
	"github.com/HackDanel/kanban-lab/backend/internal/platform/storage"
	"github.com/HackDanel/kanban-lab/backend/internal/realtime"
	"github.com/HackDanel/kanban-lab/backend/internal/repository"
	"github.com/HackDanel/kanban-lab/backend/internal/service"
)

const (
	jwtSecret  = "integration-secret-integration-secret"
	adminEmail = "root@example.com"
	adminPass  = "root-password"
	// uploadLimit is small so that the size check is cheap to exercise.
	uploadLimit = 64 << 10
)

// stack is one fully wired server.
type stack struct {
	srv   *httptest.Server
	svc   *service.Service
	store *repository.Store
	hub   *realtime.Hub
}

var (
	env   *stack
	pool  *pgxpool.Pool
	rdb   *redis.Client
	quiet = slog.New(slog.NewTextHandler(io.Discard, nil))
)

func TestMain(m *testing.M) {
	os.Exit(run(m))
}

func run(m *testing.M) int {
	ctx := context.Background()

	pg, err := tcpostgres.Run(ctx, "postgres:17-alpine",
		tcpostgres.WithDatabase("kanban"), tcpostgres.WithUsername("kanban"), tcpostgres.WithPassword("kanban"),
		tcpostgres.BasicWaitStrategies())
	if err != nil {
		log.Printf("start postgres: %v", err)
		return 1
	}
	defer func() { _ = testcontainers.TerminateContainer(pg) }()

	rc, err := tcredis.Run(ctx, "redis:8-alpine")
	if err != nil {
		log.Printf("start redis: %v", err)
		return 1
	}
	defer func() { _ = testcontainers.TerminateContainer(rc) }()

	dsn, err := pg.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		log.Print(err)
		return 1
	}
	if pool, err = postgres.Connect(ctx, dsn, 40); err != nil {
		log.Print(err)
		return 1
	}
	defer pool.Close()
	if err := postgres.Migrate(ctx, pool, quiet); err != nil {
		log.Print(err)
		return 1
	}

	redisURL, err := rc.ConnectionString(ctx)
	if err != nil {
		log.Print(err)
		return 1
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Print(err)
		return 1
	}
	rdb = redis.NewClient(opts)
	defer func() { _ = rdb.Close() }()

	uploads, err := os.MkdirTemp("", "kanban-uploads-")
	if err != nil {
		log.Print(err)
		return 1
	}
	defer func() { _ = os.RemoveAll(uploads) }()

	hubCtx, stopHub := context.WithCancel(ctx)
	defer stopHub()
	env, err = newStack(hubCtx, uploads, 1000)
	if err != nil {
		log.Print(err)
		return 1
	}
	defer env.srv.Close()

	if err := env.svc.EnsureAdmin(ctx, adminEmail, adminPass); err != nil {
		log.Print(err)
		return 1
	}
	return m.Run()
}

// newStack wires a server on the shared database and Redis.
func newStack(ctx context.Context, uploads string, authRateLimit int) (*stack, error) {
	hasher, err := auth.NewHasher(bcrypt.MinCost)
	if err != nil {
		return nil, err
	}
	tokens, err := auth.NewTokens(jwtSecret, 15*time.Minute, time.Now)
	if err != nil {
		return nil, err
	}
	files, err := storage.NewDisk(uploads)
	if err != nil {
		return nil, err
	}
	store := repository.NewStore(pool)
	hub := realtime.NewHub(rdb, quiet)
	go hub.Run(ctx)

	svc := service.New(service.Deps{
		Store: store, Hasher: hasher, Tokens: tokens, RefreshTTL: 24 * time.Hour, Now: time.Now,
		Events: hub, Files: files, UploadMaxBytes: uploadLimit, Log: quiet,
	})
	handler, err := httpserver.NewRouter(httpserver.Config{
		Service: svc, Hub: hub, Metrics: metrics.New(pool, hub.Clients),
		Limiter:        ratelimit.New(rdb, authRateLimit, time.Minute, time.Now).WithPrefix("ratelimit-" + uuid.NewString()),
		UploadMaxBytes: uploadLimit, Log: quiet,
		Ready: map[string]httpserver.ReadyCheck{
			"postgres": pool.Ping,
			"redis":    func(ctx context.Context) error { return rdb.Ping(ctx).Err() },
		},
	})
	if err != nil {
		return nil, fmt.Errorf("build router: %w", err)
	}
	return &stack{srv: httptest.NewServer(handler), svc: svc, store: store, hub: hub}, nil
}
