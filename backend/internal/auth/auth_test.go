package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
)

const secret = "0123456789abcdef0123456789abcdef"

func TestHasher(t *testing.T) {
	h, err := NewHasher(bcrypt.MinCost)
	require.NoError(t, err)

	hash, err := h.Hash("correct horse")
	require.NoError(t, err)
	assert.NotContains(t, hash, "correct horse")
	assert.True(t, strings.HasPrefix(hash, "$2"), "hash must be bcrypt")

	ok, err := h.Verify(hash, "correct horse")
	require.NoError(t, err)
	assert.True(t, ok)

	ok, err = h.Verify(hash, "wrong")
	require.NoError(t, err)
	assert.False(t, ok)

	other, err := h.Hash("correct horse")
	require.NoError(t, err)
	assert.NotEqual(t, hash, other, "hashes must be salted")

	_, err = h.Verify("not-a-hash", "correct horse")
	assert.Error(t, err)

	_, err = h.Hash(strings.Repeat("a", MaxPasswordBytes+1))
	assert.Error(t, err, "bcrypt rejects passwords over 72 bytes")

	h.BurnTime("anything")
}

func TestNewHasherRejectsBadCost(t *testing.T) {
	_, err := NewHasher(bcrypt.MaxCost + 1)
	assert.Error(t, err)
}

func TestNewTokensValidation(t *testing.T) {
	_, err := NewTokens("short", time.Minute, time.Now)
	assert.Error(t, err)
	_, err = NewTokens(secret, 0, time.Now)
	assert.Error(t, err)
}

func TestTokensRoundTrip(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tokens, err := NewTokens(secret, 15*time.Minute, func() time.Time { return now })
	require.NoError(t, err)
	assert.Equal(t, 15*time.Minute, tokens.TTL())

	actor := domain.Actor{ID: uuid.New(), Role: domain.RoleManager}
	raw, err := tokens.Issue(actor)
	require.NoError(t, err)

	got, err := tokens.Parse(raw)
	require.NoError(t, err)
	assert.Equal(t, actor, got)
}

func TestTokensExpiry(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	tokens, err := NewTokens(secret, 15*time.Minute, func() time.Time { return now })
	require.NoError(t, err)
	raw, err := tokens.Issue(domain.Actor{ID: uuid.New(), Role: domain.RoleDeveloper})
	require.NoError(t, err)

	now = now.Add(14*time.Minute + 59*time.Second)
	_, err = tokens.Parse(raw)
	assert.NoError(t, err, "token is valid until the TTL elapses")

	now = now.Add(2 * time.Second)
	_, err = tokens.Parse(raw)
	assert.ErrorIs(t, err, ErrInvalidToken)
}

func TestTokensRejectForgeries(t *testing.T) {
	now := func() time.Time { return time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC) }
	tokens, err := NewTokens(secret, time.Hour, now)
	require.NoError(t, err)
	actor := domain.Actor{ID: uuid.New(), Role: domain.RoleDeveloper}

	sign := func(method jwt.SigningMethod, key any, c jwt.Claims) string {
		raw, err := jwt.NewWithClaims(method, c).SignedString(key)
		require.NoError(t, err)
		return raw
	}
	registered := func(subject string, withExpiry bool) jwt.RegisteredClaims {
		rc := jwt.RegisteredClaims{Issuer: issuer, Subject: subject}
		if withExpiry {
			rc.ExpiresAt = jwt.NewNumericDate(now().Add(time.Hour))
		}
		return rc
	}

	cases := map[string]string{
		"garbage":     "not.a.jwt",
		"empty":       "",
		"other key":   sign(jwt.SigningMethodHS256, []byte(strings.Repeat("x", 32)), claims{Role: actor.Role, RegisteredClaims: registered(actor.ID.String(), true)}),
		"alg none":    sign(jwt.SigningMethodNone, jwt.UnsafeAllowNoneSignatureType, claims{Role: domain.RoleAdmin, RegisteredClaims: registered(actor.ID.String(), true)}),
		"other alg":   sign(jwt.SigningMethodHS512, []byte(secret), claims{Role: actor.Role, RegisteredClaims: registered(actor.ID.String(), true)}),
		"no expiry":   sign(jwt.SigningMethodHS256, []byte(secret), claims{Role: actor.Role, RegisteredClaims: registered(actor.ID.String(), false)}),
		"bad subject": sign(jwt.SigningMethodHS256, []byte(secret), claims{Role: actor.Role, RegisteredClaims: registered("42", true)}),
		"bad role":    sign(jwt.SigningMethodHS256, []byte(secret), claims{Role: "root", RegisteredClaims: registered(actor.ID.String(), true)}),
		"bad issuer": sign(jwt.SigningMethodHS256, []byte(secret), claims{Role: actor.Role, RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "someone-else", Subject: actor.ID.String(), ExpiresAt: jwt.NewNumericDate(now().Add(time.Hour)),
		}}),
	}
	for name, raw := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := tokens.Parse(raw)
			assert.ErrorIs(t, err, ErrInvalidToken)
		})
	}

	valid, err := tokens.Issue(actor)
	require.NoError(t, err)
	t.Run("tampered payload", func(t *testing.T) {
		parts := strings.Split(valid, ".")
		parts[1] = parts[1][:len(parts[1])-2] + "AA"
		_, err := tokens.Parse(strings.Join(parts, "."))
		assert.ErrorIs(t, err, ErrInvalidToken)
	})
}

func TestOpaqueTokens(t *testing.T) {
	a, err := NewOpaqueToken()
	require.NoError(t, err)
	b, err := NewOpaqueToken()
	require.NoError(t, err)

	assert.NotEqual(t, a, b)
	assert.Len(t, a, 43, "32 random bytes in unpadded base64url")
	assert.Equal(t, HashOpaqueToken(a), HashOpaqueToken(a))
	assert.NotEqual(t, HashOpaqueToken(a), HashOpaqueToken(b))
	assert.Len(t, HashOpaqueToken(a), 32)
}
