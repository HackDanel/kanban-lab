package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/HackDanel/kanban-lab/backend/internal/auth"
	"github.com/HackDanel/kanban-lab/backend/internal/domain"
	"github.com/HackDanel/kanban-lab/backend/internal/repository"
)

const minPasswordLength = 8

// Session is the result of a successful login or token refresh.
type Session struct {
	AccessToken  string
	ExpiresIn    time.Duration
	RefreshToken string
	RefreshTTL   time.Duration
	CSRFToken    string
	User         repository.User
}

var errBadCredentials = domain.E(domain.KindUnauthorized, "Неверный email или пароль")

// Register creates a developer account.
func (s *Service) Register(ctx context.Context, email, name, password string) (repository.User, error) {
	var v validator
	email = normalizeEmail(email)
	if !validEmail(email) {
		v.add("email", "Укажите корректный email")
	}
	name = v.text("name", name, true, 100, "Имя обязательно")
	switch {
	case len(password) < minPasswordLength:
		v.add("password", fmt.Sprintf("Пароль должен быть не короче %d символов", minPasswordLength))
	case len(password) > auth.MaxPasswordBytes:
		v.add("password", "Пароль слишком длинный")
	}
	if err := v.err(); err != nil {
		return repository.User{}, err
	}
	return s.createUser(ctx, email, name, password, domain.RoleDeveloper)
}

func (s *Service) createUser(ctx context.Context, email, name, password string, role domain.Role) (repository.User, error) {
	hash, err := s.hasher.Hash(password)
	if err != nil {
		return repository.User{}, fmt.Errorf("hash password: %w", err)
	}
	user, err := s.store.CreateUser(ctx, repository.CreateUserParams{
		Email: email, Name: name, PasswordHash: hash, Role: string(role),
	})
	if repository.IsUniqueViolation(err) {
		return user, domain.E(domain.KindConflict, "Пользователь с таким email уже зарегистрирован")
	}
	if err != nil {
		return user, fmt.Errorf("create user: %w", err)
	}
	return user, nil
}

// EnsureAdmin creates the bootstrap administrator unless an account with
// that email already exists.
func (s *Service) EnsureAdmin(ctx context.Context, email, password string) error {
	email = normalizeEmail(email)
	_, err := s.store.GetUserByEmail(ctx, email)
	if err == nil {
		return nil
	}
	if !repository.IsNotFound(err) {
		return fmt.Errorf("find admin: %w", err)
	}
	_, err = s.createUser(ctx, email, "Администратор", password, domain.RoleAdmin)
	if domain.KindOf(err) == domain.KindConflict {
		return nil // created concurrently by another instance
	}
	return err
}

// Login checks the credentials and opens a new session.
func (s *Service) Login(ctx context.Context, email, password string) (Session, error) {
	user, err := s.store.GetUserByEmail(ctx, normalizeEmail(email))
	if repository.IsNotFound(err) {
		s.hasher.BurnTime(password)
		return Session{}, errBadCredentials
	}
	if err != nil {
		return Session{}, fmt.Errorf("find user: %w", err)
	}
	ok, err := s.hasher.Verify(user.PasswordHash, password)
	if err != nil {
		return Session{}, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return Session{}, errBadCredentials
	}
	return s.openSession(ctx, s.store.Queries, user, uuid.New())
}

// Refresh exchanges a refresh token for a new session. Each refresh token is
// single-use: presenting one that was already exchanged means it leaked, and
// the whole family of tokens descending from that login is revoked.
func (s *Service) Refresh(ctx context.Context, refreshToken string) (Session, error) {
	if refreshToken == "" {
		return Session{}, domain.ErrUnauthorized
	}
	var session Session
	revokedFamily := false
	err := s.store.InTx(ctx, func(q *repository.Queries) error {
		token, err := q.GetRefreshTokenForUpdate(ctx, auth.HashOpaqueToken(refreshToken))
		if repository.IsNotFound(err) {
			return domain.ErrUnauthorized
		}
		if err != nil {
			return fmt.Errorf("find refresh token: %w", err)
		}
		if token.RevokedAt != nil {
			if err := q.RevokeRefreshTokenFamily(ctx, token.FamilyID); err != nil {
				return fmt.Errorf("revoke token family: %w", err)
			}
			revokedFamily = true
			return nil
		}
		if !s.now().Before(token.ExpiresAt) {
			return domain.ErrUnauthorized
		}
		if err := q.RevokeRefreshToken(ctx, token.ID); err != nil {
			return fmt.Errorf("revoke refresh token: %w", err)
		}
		user, err := q.GetUser(ctx, token.UserID)
		if err != nil {
			return fmt.Errorf("get user: %w", err)
		}
		session, err = s.openSession(ctx, q, user, token.FamilyID)
		return err
	})
	if err != nil {
		return Session{}, err
	}
	if revokedFamily {
		s.log.Warn("refresh token reuse detected, session family revoked")
		return Session{}, domain.ErrUnauthorized
	}
	return session, nil
}

// Logout revokes the session the refresh token belongs to. Unknown tokens are ignored.
func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	if refreshToken == "" {
		return nil
	}
	return s.store.InTx(ctx, func(q *repository.Queries) error {
		token, err := q.GetRefreshTokenForUpdate(ctx, auth.HashOpaqueToken(refreshToken))
		if repository.IsNotFound(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("find refresh token: %w", err)
		}
		return q.RevokeRefreshTokenFamily(ctx, token.FamilyID)
	})
}

func (s *Service) openSession(ctx context.Context, q *repository.Queries, user repository.User, family uuid.UUID) (Session, error) {
	access, err := s.tokens.Issue(domain.Actor{ID: user.ID, Role: domain.Role(user.Role)})
	if err != nil {
		return Session{}, fmt.Errorf("issue access token: %w", err)
	}
	refresh, err := auth.NewOpaqueToken()
	if err != nil {
		return Session{}, err
	}
	csrf, err := auth.NewOpaqueToken()
	if err != nil {
		return Session{}, err
	}
	err = q.CreateRefreshToken(ctx, repository.CreateRefreshTokenParams{
		UserID:    user.ID,
		FamilyID:  family,
		TokenHash: auth.HashOpaqueToken(refresh),
		ExpiresAt: s.now().Add(s.refreshTTL),
	})
	if err != nil {
		return Session{}, fmt.Errorf("store refresh token: %w", err)
	}
	return Session{
		AccessToken:  access,
		ExpiresIn:    s.tokens.TTL(),
		RefreshToken: refresh,
		RefreshTTL:   s.refreshTTL,
		CSRFToken:    csrf,
		User:         user,
	}, nil
}

// Authenticate verifies an access token.
func (s *Service) Authenticate(token string) (domain.Actor, error) {
	actor, err := s.tokens.Parse(token)
	if err != nil {
		return domain.Actor{}, domain.ErrUnauthorized
	}
	return actor, nil
}

// PurgeExpiredTokens deletes refresh tokens that can no longer be used.
func (s *Service) PurgeExpiredTokens(ctx context.Context) error {
	return s.store.DeleteExpiredRefreshTokens(ctx)
}
