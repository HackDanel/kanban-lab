package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"

	"github.com/HackDanel/kanban-lab/backend/internal/domain"
)

const issuer = "kanban-lab"

// MinSecretBytes is the shortest accepted HMAC key.
const MinSecretBytes = 32

// ErrInvalidToken is returned for any token that fails verification.
var ErrInvalidToken = errors.New("invalid token")

type claims struct {
	Role domain.Role `json:"role"`
	jwt.RegisteredClaims
}

// Tokens issues and verifies access JWTs.
type Tokens struct {
	secret []byte
	ttl    time.Duration
	now    func() time.Time
}

// NewTokens returns a Tokens that signs with secret and issues tokens valid for ttl.
func NewTokens(secret string, ttl time.Duration, now func() time.Time) (*Tokens, error) {
	if len(secret) < MinSecretBytes {
		return nil, fmt.Errorf("JWT secret must be at least %d bytes", MinSecretBytes)
	}
	if ttl <= 0 {
		return nil, errors.New("access token TTL must be positive")
	}
	return &Tokens{secret: []byte(secret), ttl: ttl, now: now}, nil
}

// TTL is the lifetime of issued access tokens.
func (t *Tokens) TTL() time.Duration { return t.ttl }

// Issue signs an access token for the actor.
func (t *Tokens) Issue(actor domain.Actor) (string, error) {
	now := t.now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims{
		Role: actor.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    issuer,
			Subject:   actor.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(t.ttl)),
		},
	})
	return token.SignedString(t.secret)
}

// Parse verifies an access token and returns the actor it was issued for.
func (t *Tokens) Parse(raw string) (domain.Actor, error) {
	var c claims
	_, err := jwt.ParseWithClaims(raw, &c,
		func(*jwt.Token) (any, error) { return t.secret, nil },
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(issuer),
		jwt.WithExpirationRequired(),
		jwt.WithTimeFunc(t.now),
	)
	if err != nil {
		return domain.Actor{}, ErrInvalidToken
	}
	id, err := uuid.Parse(c.Subject)
	if err != nil || !c.Role.Valid() {
		return domain.Actor{}, ErrInvalidToken
	}
	return domain.Actor{ID: id, Role: c.Role}, nil
}

// NewOpaqueToken returns a random URL-safe token with 256 bits of entropy.
// It is used for refresh and CSRF tokens.
func NewOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// HashOpaqueToken returns the digest under which an opaque token is stored.
// The tokens are high-entropy, so a plain SHA-256 is sufficient.
func HashOpaqueToken(token string) []byte {
	sum := sha256.Sum256([]byte(token))
	return sum[:]
}
