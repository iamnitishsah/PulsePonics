package auth

import (
	"context"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"pulse-backend/internal/platform/apperr"
)

const secret = "test-secret-that-is-at-least-32-chars!!"

func TestTokenRoundTrip(t *testing.T) {
	issuer := NewTokenIssuer(secret, 15*time.Minute)
	token, exp, err := issuer.Issue(42)
	if err != nil {
		t.Fatal(err)
	}
	if time.Until(exp) <= 0 {
		t.Error("expiry should be in the future")
	}
	id, err := issuer.Parse(token)
	if err != nil || id != 42 {
		t.Fatalf("Parse = %d, %v", id, err)
	}
}

func TestTokenRejectsExpiredAndForeign(t *testing.T) {
	issuer := NewTokenIssuer(secret, time.Minute)
	token, _, _ := issuer.Issue(1)

	issuer.now = func() time.Time { return time.Now().Add(2 * time.Minute) }
	if _, err := issuer.Parse(token); err == nil {
		t.Error("expired token accepted")
	}

	other := NewTokenIssuer("another-secret-that-is-also-32-chars!!", time.Minute)
	foreign, _, _ := other.Issue(1)
	if _, err := NewTokenIssuer(secret, time.Minute).Parse(foreign); err == nil {
		t.Error("token signed with another secret accepted")
	}
	if _, err := issuer.Parse("not-a-jwt"); err == nil {
		t.Error("garbage accepted")
	}
}

// ---- service with an in-memory repository ----

type memRepo struct {
	users  []User
	tokens map[string]*refreshToken
	nextID int64
}

func newMemRepo() *memRepo { return &memRepo{tokens: map[string]*refreshToken{}} }

func (m *memRepo) CreateUser(_ context.Context, email, name, hash string) (User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return User{}, apperr.Conflict("exists")
		}
	}
	m.nextID++
	u := User{ID: m.nextID, Email: email, Name: name, PasswordHash: hash}
	m.users = append(m.users, u)
	return u, nil
}

func (m *memRepo) find(match func(User) bool) *User {
	for i := range m.users {
		if match(m.users[i]) {
			u := m.users[i]
			return &u
		}
	}
	return nil
}

func (m *memRepo) UserByEmail(_ context.Context, email string) (*User, error) {
	return m.find(func(u User) bool { return u.Email == email }), nil
}

func (m *memRepo) UserByID(_ context.Context, id int64) (*User, error) {
	return m.find(func(u User) bool { return u.ID == id }), nil
}

func (m *memRepo) UpdatePassword(_ context.Context, id int64, hash string) error {
	for i := range m.users {
		if m.users[i].ID == id {
			m.users[i].PasswordHash = hash
		}
	}
	return nil
}

func (m *memRepo) DeleteUser(context.Context, int64) error { return nil }

func (m *memRepo) InsertRefreshToken(_ context.Context, userID int64, hash string, exp time.Time) error {
	m.tokens[hash] = &refreshToken{ID: int64(len(m.tokens) + 1), UserID: userID, ExpiresAt: exp}
	return nil
}

func (m *memRepo) RefreshTokenByHash(_ context.Context, hash string) (*refreshToken, error) {
	return m.tokens[hash], nil
}

func (m *memRepo) RevokeRefreshToken(_ context.Context, id int64, at time.Time, rotated bool) (bool, error) {
	for _, t := range m.tokens {
		if t.ID == id && t.RevokedAt == nil {
			t.RevokedAt, t.Rotated = &at, rotated
			return true, nil
		}
	}
	return false, nil
}

func (m *memRepo) RevokeAllRefreshTokens(_ context.Context, userID int64, at time.Time) error {
	for _, t := range m.tokens {
		if t.UserID == userID && t.RevokedAt == nil {
			t.RevokedAt = &at
		}
	}
	return nil
}

func newService() *Service {
	s := NewService(newMemRepo(), NewTokenIssuer(secret, 15*time.Minute), time.Hour)
	s.bcryptCost = bcrypt.MinCost // keep tests fast
	return s
}

func TestSignupAndLogin(t *testing.T) {
	s, ctx := newService(), context.Background()

	session, err := s.Signup(ctx, SignupInput{Email: "  Grower@Example.com ", Password: "hydro-pass-1"})
	if err != nil {
		t.Fatal(err)
	}
	if session.User.Email != "grower@example.com" || session.User.Name != "grower" {
		t.Errorf("email/name not normalised: %+v", session.User)
	}
	if session.AccessToken == "" || session.RefreshToken == "" {
		t.Error("tokens missing")
	}

	if _, err := s.Signup(ctx, SignupInput{Email: "grower@example.com", Password: "hydro-pass-1"}); !apperr.IsKind(err, apperr.KindConflict) {
		t.Errorf("duplicate signup: want conflict, got %v", err)
	}
	if _, err := s.Login(ctx, LoginInput{Email: "GROWER@example.com", Password: "hydro-pass-1"}); err != nil {
		t.Errorf("login failed: %v", err)
	}
	if _, err := s.Login(ctx, LoginInput{Email: "grower@example.com", Password: "wrong-pass"}); !apperr.IsKind(err, apperr.KindUnauthorized) {
		t.Errorf("wrong password: want unauthorized, got %v", err)
	}
	if _, err := s.Login(ctx, LoginInput{Email: "nobody@example.com", Password: "whatever1"}); !apperr.IsKind(err, apperr.KindUnauthorized) {
		t.Errorf("unknown email: want unauthorized, got %v", err)
	}
}

func TestSignupValidation(t *testing.T) {
	s, ctx := newService(), context.Background()
	bad := []SignupInput{
		{Email: "not-an-email", Password: "long-enough"},
		{Email: "Name <a@b.co>", Password: "long-enough"},
		{Email: "a@b.co", Password: "short"},
	}
	for _, in := range bad {
		if _, err := s.Signup(ctx, in); !apperr.IsKind(err, apperr.KindInvalid) {
			t.Errorf("%+v: want invalid, got %v", in, err)
		}
	}
}

func TestRefreshRotationAndReuseDetection(t *testing.T) {
	s, ctx := newService(), context.Background()
	first, _ := s.Signup(ctx, SignupInput{Email: "a@b.co", Password: "hydro-pass-1"})

	second, err := s.Refresh(ctx, RefreshInput{RefreshToken: first.RefreshToken})
	if err != nil {
		t.Fatal(err)
	}
	if second.RefreshToken == first.RefreshToken {
		t.Error("refresh token was not rotated")
	}

	// Replaying the old token revokes everything, including the new token.
	if _, err := s.Refresh(ctx, RefreshInput{RefreshToken: first.RefreshToken}); !apperr.IsKind(err, apperr.KindUnauthorized) {
		t.Errorf("reused token: want unauthorized, got %v", err)
	}
	if _, err := s.Refresh(ctx, RefreshInput{RefreshToken: second.RefreshToken}); !apperr.IsKind(err, apperr.KindUnauthorized) {
		t.Error("token issued before reuse should have been revoked")
	}
}

func TestChangePasswordRevokesOtherSessions(t *testing.T) {
	s, ctx := newService(), context.Background()
	old, _ := s.Signup(ctx, SignupInput{Email: "a@b.co", Password: "hydro-pass-1"})

	if _, err := s.ChangePassword(ctx, old.User.ID, ChangePasswordInput{CurrentPassword: "wrong", NewPassword: "hydro-pass-2"}); !apperr.IsKind(err, apperr.KindInvalid) {
		t.Errorf("wrong current password: want invalid, got %v", err)
	}
	fresh, err := s.ChangePassword(ctx, old.User.ID, ChangePasswordInput{CurrentPassword: "hydro-pass-1", NewPassword: "hydro-pass-2"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Refresh(ctx, RefreshInput{RefreshToken: old.RefreshToken}); err == nil {
		t.Error("old session should be revoked")
	}
	if _, err := s.Refresh(ctx, RefreshInput{RefreshToken: fresh.RefreshToken}); err != nil {
		t.Errorf("new session should work: %v", err)
	}
	if _, err := s.Login(ctx, LoginInput{Email: "a@b.co", Password: "hydro-pass-2"}); err != nil {
		t.Errorf("login with new password: %v", err)
	}
}
