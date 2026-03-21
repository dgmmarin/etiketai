package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type User struct {
	ID            string
	Email         string
	PasswordHash  *string
	OAuthProvider *string
	OAuthSub      *string
	IsVerified    bool
	IsSuperAdmin  bool
	CreatedAt     time.Time
	LastLoginAt   *time.Time
}

var ErrNotFound = errors.New("not found")
var ErrDuplicate = errors.New("already exists")

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, email, passwordHash string) (*User, error) {
	var u User
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, is_verified, created_at)
		VALUES ($1, $2, false, NOW())
		RETURNING id, email, password_hash, is_verified, is_superadmin, created_at
	`, email, passwordHash).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.IsVerified, &u.IsSuperAdmin, &u.CreatedAt,
	)
	if err != nil {
		if isUniqueViolation(err) {
			return nil, ErrDuplicate
		}
		return nil, err
	}
	return &u, nil
}

func (r *UserRepo) CreateOAuth(ctx context.Context, email, provider, sub string) (*User, error) {
	var u User
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (email, oauth_provider, oauth_sub, is_verified, created_at)
		VALUES ($1, $2, $3, true, NOW())
		ON CONFLICT (email) DO UPDATE
		  SET oauth_provider = EXCLUDED.oauth_provider,
		      oauth_sub = EXCLUDED.oauth_sub,
		      last_login_at = NOW()
		RETURNING id, email, oauth_provider, oauth_sub, is_verified, is_superadmin, created_at
	`, email, provider, sub).Scan(
		&u.ID, &u.Email, &u.OAuthProvider, &u.OAuthSub, &u.IsVerified, &u.IsSuperAdmin, &u.CreatedAt,
	)
	return &u, err
}

func (r *UserRepo) GetByEmail(ctx context.Context, email string) (*User, error) {
	var u User
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, oauth_provider, oauth_sub, is_verified, is_superadmin, created_at, last_login_at
		FROM users WHERE email = $1
	`, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.OAuthProvider, &u.OAuthSub,
		&u.IsVerified, &u.IsSuperAdmin, &u.CreatedAt, &u.LastLoginAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*User, error) {
	var u User
	err := r.db.QueryRow(ctx, `
		SELECT id, email, password_hash, oauth_provider, oauth_sub, is_verified, is_superadmin, created_at, last_login_at
		FROM users WHERE id = $1
	`, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.OAuthProvider, &u.OAuthSub,
		&u.IsVerified, &u.IsSuperAdmin, &u.CreatedAt, &u.LastLoginAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	return &u, err
}

func (r *UserRepo) MarkVerified(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET is_verified = true WHERE id = $1
	`, userID)
	return err
}

func (r *UserRepo) UpdateLastLogin(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE users SET last_login_at = NOW() WHERE id = $1
	`, userID)
	return err
}

func (r *UserRepo) SaveEmailVerificationToken(ctx context.Context, userID, tokenHash string, expiresAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO email_verifications (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, tokenHash, expiresAt)
	return err
}

func (r *UserRepo) ConsumeEmailVerificationToken(ctx context.Context, tokenHash string) (string, error) {
	var userID string
	err := r.db.QueryRow(ctx, `
		UPDATE email_verifications
		SET used_at = NOW()
		WHERE token_hash = $1
		  AND expires_at > NOW()
		  AND used_at IS NULL
		RETURNING user_id
	`, tokenHash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return userID, err
}

func isUniqueViolation(err error) bool {
	// pgx wraps pgconn errors; check for SQLSTATE 23505
	return err != nil && (err.Error() == "ERROR: duplicate key value violates unique constraint" ||
		containsCode(err, "23505"))
}

func containsCode(err error, code string) bool {
	type coder interface{ Code() string }
	var c coder
	if errors.As(err, &c) {
		return c.Code() == code
	}
	return false
}
