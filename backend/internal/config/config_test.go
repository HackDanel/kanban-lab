package config

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func env(overrides map[string]string) func(string) string {
	values := map[string]string{
		"DATABASE_URL": "postgres://kanban:kanban@localhost:5432/kanban",
		"REDIS_URL":    "redis://localhost:6379/0",
		"JWT_SECRET":   strings.Repeat("s", 32),
	}
	for k, v := range overrides {
		values[k] = v
	}
	return func(key string) string { return values[key] }
}

func TestLoadDefaults(t *testing.T) {
	cfg, err := Load(env(nil))
	require.NoError(t, err)

	assert.Equal(t, "development", cfg.Env)
	assert.False(t, cfg.Production())
	assert.Equal(t, ":8080", cfg.HTTPAddr)
	assert.Equal(t, int32(50), cfg.DBMaxConns)
	assert.Equal(t, 15*time.Minute, cfg.AccessTTL)
	assert.Equal(t, 30*24*time.Hour, cfg.RefreshTTL)
	assert.Equal(t, 12, cfg.BcryptCost)
	assert.True(t, cfg.CookieSecure, "cookies are Secure unless explicitly disabled")
	assert.Empty(t, cfg.CORSOrigins)
	assert.Equal(t, 20, cfg.AuthRateLimit)
	assert.Equal(t, int64(10<<20), cfg.UploadMaxBytes)
}

func TestLoadOverrides(t *testing.T) {
	cfg, err := Load(env(map[string]string{
		"APP_ENV":          "production",
		"HTTP_ADDR":        " :9000 ",
		"DB_MAX_CONNS":     "10",
		"JWT_ACCESS_TTL":   "5m",
		"REFRESH_TTL":      "24h",
		"BCRYPT_COST":      "10",
		"COOKIE_SECURE":    "true",
		"CORS_ORIGINS":     "https://a.example, https://b.example,,",
		"AUTH_RATE_LIMIT":  "5",
		"UPLOAD_MAX_BYTES": "1024",
		"ADMIN_EMAIL":      "admin@example.com",
		"ADMIN_PASSWORD":   "secret-password",
	}))
	require.NoError(t, err)

	assert.True(t, cfg.Production())
	assert.Equal(t, ":9000", cfg.HTTPAddr)
	assert.Equal(t, int32(10), cfg.DBMaxConns)
	assert.Equal(t, 5*time.Minute, cfg.AccessTTL)
	assert.Equal(t, 24*time.Hour, cfg.RefreshTTL)
	assert.Equal(t, 10, cfg.BcryptCost)
	assert.Equal(t, []string{"https://a.example", "https://b.example"}, cfg.CORSOrigins)
	assert.Equal(t, 5, cfg.AuthRateLimit)
	assert.Equal(t, int64(1024), cfg.UploadMaxBytes)
	assert.Equal(t, "admin@example.com", cfg.AdminEmail)
}

func TestLoadReportsAllProblems(t *testing.T) {
	_, err := Load(func(string) string { return "" })
	require.Error(t, err)
	for _, key := range []string{"DATABASE_URL", "REDIS_URL", "JWT_SECRET"} {
		assert.Contains(t, err.Error(), key+": is required")
	}
}

func TestLoadRejectsInvalidValues(t *testing.T) {
	cases := map[string]struct {
		key, value, want string
	}{
		"short secret":       {"JWT_SECRET", "short", "JWT_SECRET: must be at least 32 bytes"},
		"non-numeric int":    {"DB_MAX_CONNS", "many", "DB_MAX_CONNS: must be an integer"},
		"int below range":    {"DB_MAX_CONNS", "0", "DB_MAX_CONNS: must be an integer"},
		"int above range":    {"BCRYPT_COST", "32", "BCRYPT_COST: must be an integer"},
		"bad duration":       {"JWT_ACCESS_TTL", "soon", "JWT_ACCESS_TTL: must be a positive duration"},
		"negative duration":  {"REFRESH_TTL", "-1h", "REFRESH_TTL: must be a positive duration"},
		"bad bool":           {"COOKIE_SECURE", "maybe", "COOKIE_SECURE: must be true or false"},
		"admin without pass": {"ADMIN_EMAIL", "admin@example.com", "must be set together"},
		"admin without mail": {"ADMIN_PASSWORD", "secret", "must be set together"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Load(env(map[string]string{tc.key: tc.value}))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.want)
		})
	}
}

func TestLoadRequiresSecureCookiesInProduction(t *testing.T) {
	_, err := Load(env(map[string]string{"APP_ENV": "production", "COOKIE_SECURE": "false"}))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "COOKIE_SECURE: must be true in production")

	_, err = Load(env(map[string]string{"APP_ENV": "development", "COOKIE_SECURE": "false"}))
	assert.NoError(t, err)
}
