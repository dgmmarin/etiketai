package repo

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	blacklistPrefix = "jwt:blacklist:"
	ratelimitPrefix = "ratelimit:login:"
)

type TokenRepo struct {
	db    *pgxpool.Pool
	redis *redis.Client
}

func NewTokenRepo(db *pgxpool.Pool, redis *redis.Client) *TokenRepo {
	return &TokenRepo{db: db, redis: redis}
}

// ─── Refresh tokens ───────────────────────────────────────────────────────────

func (r *TokenRepo) SaveRefreshToken(ctx context.Context, userID, rawToken string, expiresAt time.Time) error {
	hash := hashToken(rawToken)
	_, err := r.db.Exec(ctx, `
		INSERT INTO refresh_tokens (user_id, token_hash, expires_at)
		VALUES ($1, $2, $3)
	`, userID, hash, expiresAt)
	return err
}

func (r *TokenRepo) ValidateAndRotateRefreshToken(ctx context.Context, rawToken string) (userID string, err error) {
	hash := hashToken(rawToken)
	err = r.db.QueryRow(ctx, `
		UPDATE refresh_tokens
		SET revoked_at = NOW()
		WHERE token_hash = $1
		  AND expires_at > NOW()
		  AND revoked_at IS NULL
		RETURNING user_id
	`, hash).Scan(&userID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	return userID, err
}

func (r *TokenRepo) RevokeRefreshToken(ctx context.Context, rawToken string) error {
	hash := hashToken(rawToken)
	_, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = NOW()
		WHERE token_hash = $1 AND revoked_at IS NULL
	`, hash)
	return err
}

func (r *TokenRepo) RevokeAllUserTokens(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE refresh_tokens SET revoked_at = NOW()
		WHERE user_id = $1 AND revoked_at IS NULL
	`, userID)
	return err
}

// ─── JWT blacklist (Redis) ────────────────────────────────────────────────────

// BlacklistAccessToken prevents a JWT from being used before it expires.
func (r *TokenRepo) BlacklistAccessToken(ctx context.Context, jti string, ttl time.Duration) error {
	return r.redis.Set(ctx, blacklistPrefix+jti, "1", ttl).Err()
}

func (r *TokenRepo) IsAccessTokenBlacklisted(ctx context.Context, jti string) (bool, error) {
	err := r.redis.Get(ctx, blacklistPrefix+jti).Err()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// ─── Rate limiting (sliding window, Redis) ────────────────────────────────────

// RecordLoginAttempt records an attempt and returns current count.
// Window: 15 minutes. Max: 5 attempts.
func (r *TokenRepo) RecordLoginAttempt(ctx context.Context, email string) (int64, error) {
	key := ratelimitPrefix + email
	now := time.Now().UnixMilli()
	window := 15 * time.Minute

	pipe := r.redis.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", now-window.Milliseconds()))
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(now), Member: now})
	pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, window)

	results, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	count := results[2].(*redis.IntCmd).Val()
	return count, nil
}

func (r *TokenRepo) ClearLoginAttempts(ctx context.Context, email string) error {
	return r.redis.Del(ctx, ratelimitPrefix+email).Err()
}

// ─── helpers ──────────────────────────────────────────────────────────────────

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", h)
}
