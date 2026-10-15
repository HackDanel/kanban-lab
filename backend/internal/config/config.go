// Package config reads the server configuration from environment variables.
package config

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/HackDanel/kanban-lab/backend/internal/auth"
)

// Config is the complete server configuration.
type Config struct {
	Env             string
	HTTPAddr        string
	DatabaseURL     string
	DBMaxConns      int32
	RedisURL        string
	JWTSecret       string
	AccessTTL       time.Duration
	RefreshTTL      time.Duration
	BcryptCost      int
	CookieSecure    bool
	CORSOrigins     []string
	AuthRateLimit   int
	UploadDir       string
	UploadMaxBytes  int64
	AdminEmail      string
	AdminPassword   string
	ShutdownTimeout time.Duration
}

// Production reports whether the server runs in the production environment.
func (c Config) Production() bool { return c.Env == "production" }

// Load reads the configuration using getenv (os.Getenv in production) and
// reports every problem at once.
func Load(getenv func(string) string) (Config, error) {
	l := loader{getenv: getenv}
	cfg := Config{
		Env:             l.str("APP_ENV", "development"),
		HTTPAddr:        l.str("HTTP_ADDR", ":8080"),
		DatabaseURL:     l.required("DATABASE_URL"),
		DBMaxConns:      int32(l.integer("DB_MAX_CONNS", 50, 1, 1000)),
		RedisURL:        l.required("REDIS_URL"),
		JWTSecret:       l.required("JWT_SECRET"),
		AccessTTL:       l.duration("JWT_ACCESS_TTL", 15*time.Minute),
		RefreshTTL:      l.duration("REFRESH_TTL", 30*24*time.Hour),
		BcryptCost:      l.integer("BCRYPT_COST", 12, 4, 31),
		CookieSecure:    l.boolean("COOKIE_SECURE", true),
		CORSOrigins:     l.list("CORS_ORIGINS"),
		AuthRateLimit:   l.integer("AUTH_RATE_LIMIT", 20, 1, 1_000_000),
		UploadDir:       l.str("UPLOAD_DIR", "./uploads"),
		UploadMaxBytes:  int64(l.integer("UPLOAD_MAX_BYTES", 10<<20, 1, 1<<30)),
		AdminEmail:      l.str("ADMIN_EMAIL", ""),
		AdminPassword:   l.str("ADMIN_PASSWORD", ""),
		ShutdownTimeout: l.duration("SHUTDOWN_TIMEOUT", 15*time.Second),
	}

	if cfg.JWTSecret != "" && len(cfg.JWTSecret) < auth.MinSecretBytes {
		l.fail("JWT_SECRET", fmt.Sprintf("must be at least %d bytes", auth.MinSecretBytes))
	}
	if (cfg.AdminEmail == "") != (cfg.AdminPassword == "") {
		l.fail("ADMIN_EMAIL/ADMIN_PASSWORD", "must be set together")
	}
	if cfg.Production() && !cfg.CookieSecure {
		l.fail("COOKIE_SECURE", "must be true in production")
	}
	return cfg, errors.Join(l.errs...)
}

type loader struct {
	getenv func(string) string
	errs   []error
}

func (l *loader) fail(key, reason string) {
	l.errs = append(l.errs, fmt.Errorf("%s: %s", key, reason))
}

func (l *loader) str(key, fallback string) string {
	if v := strings.TrimSpace(l.getenv(key)); v != "" {
		return v
	}
	return fallback
}

func (l *loader) required(key string) string {
	v := l.str(key, "")
	if v == "" {
		l.fail(key, "is required")
	}
	return v
}

func (l *loader) integer(key string, fallback, lo, hi int) int {
	raw := l.str(key, "")
	if raw == "" {
		return fallback
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < lo || n > hi {
		l.fail(key, fmt.Sprintf("must be an integer between %d and %d", lo, hi))
		return fallback
	}
	return n
}

func (l *loader) duration(key string, fallback time.Duration) time.Duration {
	raw := l.str(key, "")
	if raw == "" {
		return fallback
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		l.fail(key, "must be a positive duration such as 15m")
		return fallback
	}
	return d
}

func (l *loader) boolean(key string, fallback bool) bool {
	raw := l.str(key, "")
	if raw == "" {
		return fallback
	}
	b, err := strconv.ParseBool(raw)
	if err != nil {
		l.fail(key, "must be true or false")
		return fallback
	}
	return b
}

func (l *loader) list(key string) []string {
	var out []string
	for _, item := range strings.Split(l.getenv(key), ",") {
		if item = strings.TrimSpace(item); item != "" {
			out = append(out, item)
		}
	}
	return out
}
