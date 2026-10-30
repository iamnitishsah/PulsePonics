package auth

import (
	"context"
	"fmt"
	"log"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"golang.org/x/crypto/bcrypt"

	"pulse-backend/internal/platform/apperr"
)

const (
	minPasswordLen = 8
	maxPasswordLen = 72 // bcrypt ignores bytes past 72
	maxNameLen     = 100
	maxEmailLen    = 254
)

var errInvalidCredentials = apperr.Unauthorized("invalid email or password")

type Service struct {
	repo       Repository
	tokens     *TokenIssuer
	refreshTTL time.Duration
	bcryptCost int
	now        func() time.Time

	// Compared against when an email is unknown, so login takes the same time
	// whether or not the account exists.
	dummyHash []byte
}

func NewService(repo Repository, tokens *TokenIssuer, refreshTTL time.Duration) *Service {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("pulseponics-timing-equaliser"), bcrypt.DefaultCost)
	return &Service{
		repo:       repo,
		tokens:     tokens,
		refreshTTL: refreshTTL,
		bcryptCost: bcrypt.DefaultCost,
		now:        time.Now,
		dummyHash:  dummy,
	}
}

func (s *Service) Signup(ctx context.Context, in SignupInput) (Session, error) {
	email, err := normalizeEmail(in.Email)
	if err != nil {
		return Session{}, err
	}
	if err := validatePassword("password", in.Password); err != nil {
		return Session{}, err
	}
	name := strings.TrimSpace(in.Name)
	if name == "" {
		name = strings.SplitN(email, "@", 2)[0]
	}
	if utf8.RuneCountInString(name) > maxNameLen {
		return Session{}, apperr.Invalid("name", fmt.Sprintf("name must be at most %d characters", maxNameLen))
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), s.bcryptCost)
	if err != nil {
		return Session{}, fmt.Errorf("auth: hash password: %w", err)
	}
	user, err := s.repo.CreateUser(ctx, email, name, string(hash))
	if err != nil {
		return Session{}, err
	}
	return s.issueSession(ctx, user)
}

func (s *Service) Login(ctx context.Context, in LoginInput) (Session, error) {
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if email == "" || in.Password == "" {
		return Session{}, apperr.Invalid("", "email and password are required")
	}
	user, err := s.repo.UserByEmail(ctx, email)
	if err != nil {
		return Session{}, err
	}
	if user == nil {
		_ = bcrypt.CompareHashAndPassword(s.dummyHash, []byte(in.Password))
		return Session{}, errInvalidCredentials
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		return Session{}, errInvalidCredentials
	}
	return s.issueSession(ctx, *user)
}

// Refresh rotates the refresh token: the presented token is revoked and a new
// pair is issued. Presenting a token that was already rotated means it was
// copied (stolen or replayed), so every session of that user is revoked.
func (s *Service) Refresh(ctx context.Context, in RefreshInput) (Session, error) {
	invalid := apperr.Unauthorized("invalid or expired refresh token")
	if in.RefreshToken == "" {
		return Session{}, invalid
	}
	stored, err := s.repo.RefreshTokenByHash(ctx, HashToken(in.RefreshToken))
	if err != nil {
		return Session{}, err
	}
	if stored == nil {
		return Session{}, invalid
	}
	now := s.now()
	if stored.RevokedAt != nil {
		if stored.Rotated {
			log.Printf("auth: reuse of rotated refresh token for user %d, revoking all sessions", stored.UserID)
			if err := s.repo.RevokeAllRefreshTokens(ctx, stored.UserID, now); err != nil {
				return Session{}, err
			}
		}
		return Session{}, invalid // logged out or signed out by a password change
	}
	if now.After(stored.ExpiresAt) {
		return Session{}, invalid
	}
	revoked, err := s.repo.RevokeRefreshToken(ctx, stored.ID, now, true)
	if err != nil {
		return Session{}, err
	}
	if !revoked {
		return Session{}, invalid // lost a race with a concurrent refresh
	}
	user, err := s.repo.UserByID(ctx, stored.UserID)
	if err != nil {
		return Session{}, err
	}
	if user == nil {
		return Session{}, invalid
	}
	return s.issueSession(ctx, *user)
}

// Logout revokes the given refresh token. Unknown tokens are ignored, so
// logout always succeeds from the client's point of view.
func (s *Service) Logout(ctx context.Context, in RefreshInput) error {
	if in.RefreshToken == "" {
		return nil
	}
	stored, err := s.repo.RefreshTokenByHash(ctx, HashToken(in.RefreshToken))
	if err != nil || stored == nil {
		return err
	}
	_, err = s.repo.RevokeRefreshToken(ctx, stored.ID, s.now(), false)
	return err
}

func (s *Service) Me(ctx context.Context, userID int64) (User, error) {
	user, err := s.repo.UserByID(ctx, userID)
	if err != nil {
		return User{}, err
	}
	if user == nil {
		return User{}, apperr.Unauthorized("account no longer exists")
	}
	return *user, nil
}

// ChangePassword signs out every other session and returns a fresh one for
// the caller.
func (s *Service) ChangePassword(ctx context.Context, userID int64, in ChangePasswordInput) (Session, error) {
	user, err := s.Me(ctx, userID)
	if err != nil {
		return Session{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.CurrentPassword)) != nil {
		return Session{}, apperr.Invalid("current_password", "current password is incorrect")
	}
	if err := validatePassword("new_password", in.NewPassword); err != nil {
		return Session{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.NewPassword), s.bcryptCost)
	if err != nil {
		return Session{}, fmt.Errorf("auth: hash password: %w", err)
	}
	if err := s.repo.UpdatePassword(ctx, userID, string(hash)); err != nil {
		return Session{}, err
	}
	if err := s.repo.RevokeAllRefreshTokens(ctx, userID, s.now()); err != nil {
		return Session{}, err
	}
	return s.issueSession(ctx, user)
}

// DeleteAccount removes the user; devices, readings, thresholds and alerts
// go with it through ON DELETE CASCADE.
func (s *Service) DeleteAccount(ctx context.Context, userID int64, in DeleteAccountInput) error {
	user, err := s.Me(ctx, userID)
	if err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)) != nil {
		return apperr.Invalid("password", "password is incorrect")
	}
	return s.repo.DeleteUser(ctx, userID)
}

func (s *Service) issueSession(ctx context.Context, user User) (Session, error) {
	access, accessExp, err := s.tokens.Issue(user.ID)
	if err != nil {
		return Session{}, err
	}
	refresh, hash, err := newOpaqueToken("")
	if err != nil {
		return Session{}, err
	}
	refreshExp := s.now().Add(s.refreshTTL)
	if err := s.repo.InsertRefreshToken(ctx, user.ID, hash, refreshExp); err != nil {
		return Session{}, err
	}
	return Session{
		AccessToken:           access,
		AccessTokenExpiresAt:  accessExp.UTC(),
		RefreshToken:          refresh,
		RefreshTokenExpiresAt: refreshExp.UTC(),
		User:                  user,
	}, nil
}

func normalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", apperr.Invalid("email", "email is required")
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || len(email) > maxEmailLen {
		return "", apperr.Invalid("email", "email address is not valid")
	}
	return email, nil
}

func validatePassword(field, password string) error {
	switch {
	case len(password) < minPasswordLen:
		return apperr.Invalid(field, fmt.Sprintf("password must be at least %d characters", minPasswordLen))
	case len(password) > maxPasswordLen:
		return apperr.Invalid(field, fmt.Sprintf("password must be at most %d bytes", maxPasswordLen))
	}
	return nil
}
