package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimitConfig defines limits for a particular route group.
type RateLimitConfig struct {
	// Max requests per window per key
	Max    int64
	Window time.Duration
}

// RateLimit returns a sliding-window rate limiter middleware backed by Redis.
// keyFn extracts the rate limit key from the request (e.g. IP or user ID).
func RateLimit(rdb *redis.Client, cfg RateLimitConfig, keyFn func(r *http.Request) string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := "ratelimit:" + keyFn(r)
			now := time.Now().UnixMilli()
			windowMs := cfg.Window.Milliseconds()

			ctx := r.Context()
			count, err := slidingWindowCount(ctx, rdb, key, now, windowMs, cfg.Window)
			if err != nil {
				// Fail open — don't block requests if Redis is down
				next.ServeHTTP(w, r)
				return
			}

			if count > cfg.Max {
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", cfg.Window.Seconds()))
				writeError(w, http.StatusTooManyRequests, "rate limit exceeded", "RATE_LIMITED")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func slidingWindowCount(ctx context.Context, rdb *redis.Client, key string, nowMs, windowMs int64, ttl time.Duration) (int64, error) {
	pipe := rdb.Pipeline()
	pipe.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", nowMs-windowMs))
	pipe.ZAdd(ctx, key, redis.Z{Score: float64(nowMs), Member: nowMs})
	zcard := pipe.ZCard(ctx, key)
	pipe.Expire(ctx, key, ttl)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}
	return zcard.Val(), nil
}

// IPKey extracts the client IP address as rate limit key.
func IPKey(r *http.Request) string {
	if ip := r.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	return r.RemoteAddr
}
