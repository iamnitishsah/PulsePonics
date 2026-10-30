package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"

	"pulse-backend/internal/platform/apperr"
	"pulse-backend/internal/platform/database"
)

type Repository interface {
	CreateUser(ctx context.Context, email, name, passwordHash string) (User, error)
	UserByEmail(ctx context.Context, email string) (*User, error)
	UserByID(ctx context.Context, id int64) (*User, error)
	UpdatePassword(ctx context.Context, userID int64, passwordHash string) error
	DeleteUser(ctx context.Context, userID int64) error

	InsertRefreshToken(ctx context.Context, userID int64, hash string, expiresAt time.Time) error
	RefreshTokenByHash(ctx context.Context, hash string) (*refreshToken, error)
	// RevokeRefreshToken returns false if the token was already revoked, so
	// two concurrent refreshes with the same token cannot both succeed.
	// rotated marks a revocation caused by exchanging the token in Refresh.
	RevokeRefreshToken(ctx context.Context, id int64, at time.Time, rotated bool) (bool, error)
	RevokeAllRefreshTokens(ctx context.Context, userID int64, at time.Time) error
}

type mysqlRepository struct {
	db *sqlx.DB
}

func NewMySQLRepository(db *sqlx.DB) Repository {
	return &mysqlRepository{db: db}
}

const userColumns = `id, email, name, password_hash, created_at, updated_at`

func (r *mysqlRepository) CreateUser(ctx context.Context, email, name, passwordHash string) (User, error) {
	res, err := r.db.ExecContext(ctx,
		`INSERT INTO users (email, name, password_hash) VALUES (?, ?, ?)`, email, name, passwordHash)
	if err != nil {
		if database.IsDuplicateKey(err) {
			return User{}, apperr.Conflict("an account with this email already exists")
		}
		return User{}, fmt.Errorf("auth repo: insert user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return User{}, fmt.Errorf("auth repo: last insert id: %w", err)
	}
	u, err := r.UserByID(ctx, id)
	if err != nil {
		return User{}, err
	}
	return *u, nil
}

func (r *mysqlRepository) getUser(ctx context.Context, query string, arg any) (*User, error) {
	var u User
	err := r.db.GetContext(ctx, &u, query, arg)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("auth repo: get user: %w", err)
	}
	return &u, nil
}

func (r *mysqlRepository) UserByEmail(ctx context.Context, email string) (*User, error) {
	return r.getUser(ctx, `SELECT `+userColumns+` FROM users WHERE email = ?`, email)
}

func (r *mysqlRepository) UserByID(ctx context.Context, id int64) (*User, error) {
	return r.getUser(ctx, `SELECT `+userColumns+` FROM users WHERE id = ?`, id)
}

func (r *mysqlRepository) UpdatePassword(ctx context.Context, userID int64, passwordHash string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET password_hash = ? WHERE id = ?`, passwordHash, userID)
	if err != nil {
		return fmt.Errorf("auth repo: update password: %w", err)
	}
	return nil
}

func (r *mysqlRepository) DeleteUser(ctx context.Context, userID int64) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM users WHERE id = ?`, userID)
	if err != nil {
		return fmt.Errorf("auth repo: delete user: %w", err)
	}
	return nil
}

func (r *mysqlRepository) InsertRefreshToken(ctx context.Context, userID int64, hash string, expiresAt time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES (?, ?, ?)`, userID, hash, expiresAt)
	if err != nil {
		return fmt.Errorf("auth repo: insert refresh token: %w", err)
	}
	return nil
}

func (r *mysqlRepository) RefreshTokenByHash(ctx context.Context, hash string) (*refreshToken, error) {
	var t refreshToken
	err := r.db.GetContext(ctx, &t,
		`SELECT id, user_id, expires_at, revoked_at, rotated FROM refresh_tokens WHERE token_hash = ?`, hash)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("auth repo: get refresh token: %w", err)
	}
	return &t, nil
}

func (r *mysqlRepository) RevokeRefreshToken(ctx context.Context, id int64, at time.Time, rotated bool) (bool, error) {
	res, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = ?, rotated = ? WHERE id = ? AND revoked_at IS NULL`, at, rotated, id)
	if err != nil {
		return false, fmt.Errorf("auth repo: revoke refresh token: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("auth repo: rows affected: %w", err)
	}
	return n == 1, nil
}

func (r *mysqlRepository) RevokeAllRefreshTokens(ctx context.Context, userID int64, at time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE refresh_tokens SET revoked_at = ? WHERE user_id = ? AND revoked_at IS NULL`, at, userID)
	if err != nil {
		return fmt.Errorf("auth repo: revoke all refresh tokens: %w", err)
	}
	return nil
}
