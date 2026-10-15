// Package auth implements password hashing and token handling.
package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// MaxPasswordBytes is the longest password bcrypt accepts.
const MaxPasswordBytes = 72

// Hasher hashes and verifies passwords with bcrypt.
type Hasher struct {
	cost  int
	dummy []byte
}

// NewHasher returns a Hasher using the given bcrypt cost.
func NewHasher(cost int) (*Hasher, error) {
	dummy, err := bcrypt.GenerateFromPassword([]byte("dummy-password"), cost)
	if err != nil {
		return nil, err
	}
	return &Hasher{cost: cost, dummy: dummy}, nil
}

// Hash returns the bcrypt hash of password.
func (h *Hasher) Hash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// Verify reports whether password matches hash.
func (h *Hasher) Verify(hash, password string) (bool, error) {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
		return false, nil
	}
	return err == nil, err
}

// BurnTime spends the time of one verification. It is called when the account
// does not exist so that response time does not reveal which emails are registered.
func (h *Hasher) BurnTime(password string) {
	_ = bcrypt.CompareHashAndPassword(h.dummy, []byte(password))
}
