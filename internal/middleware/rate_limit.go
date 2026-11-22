package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/amirullazmi0/web-profile-backend/pkg/logger"
	"github.com/amirullazmi0/web-profile-backend/pkg/redis"
	"github.com/amirullazmi0/web-profile-backend/pkg/response"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type RateLimitConfig struct {
	KeyPrefix string
	Limit     int
	Window    time.Duration
}

// RateLimit creates a fixed-window Redis-backed rate limiter.
// KeyPrefix separates counters per endpoint group, while Limit and Window define
// how many requests are allowed in a time window for a single client IP.
func RateLimit(redisClient *redis.Client, cfg RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// If Redis is unavailable, fail closed so auth traffic does not bypass limits.
		if redisClient == nil || redisClient.Client == nil {
			response.Error(c, http.StatusServiceUnavailable, "Rate limit service unavailable", nil)
			c.Abort()
			return
		}

		// A disabled limiter just passes through without touching Redis.
		if cfg.Limit <= 0 || cfg.Window <= 0 {
			c.Next()
			return
		}

		// Fixed-window key, grouped by route prefix and client IP.
		key := fmt.Sprintf("ratelimit:%s:%s", cfg.KeyPrefix, c.ClientIP())
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Increment the request counter for this key.
		count, err := redisClient.Client.Incr(ctx, key).Result()
		if err != nil {
			logger.Error("Failed to increment rate limit counter",
				zap.Error(err),
				zap.String("key", key),
				zap.String("path", c.Request.URL.Path),
			)
			response.Error(c, http.StatusServiceUnavailable, "Rate limit service unavailable", nil)
			c.Abort()
			return
		}

		// First hit sets the expiration window for the counter.
		if count == 1 {
			if err := redisClient.Client.Expire(ctx, key, cfg.Window).Err(); err != nil {
				logger.Error("Failed to set rate limit TTL",
					zap.Error(err),
					zap.String("key", key),
				)
			}
		}

		// Expose remaining quota in response headers for easier debugging.
		ttl, err := redisClient.Client.TTL(ctx, key).Result()
		if err == nil && ttl > 0 {
			c.Header("X-RateLimit-Limit", strconv.Itoa(cfg.Limit))
			remaining := cfg.Limit - int(count)
			if remaining < 0 {
				remaining = 0
			}
			c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
			c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(ttl).Unix(), 10))
		}

		// Reject requests that exceed the configured limit.
		if int(count) > cfg.Limit {
			retryAfter := int(ttl.Seconds())
			if retryAfter < 1 {
				retryAfter = int(cfg.Window.Seconds())
			}
			c.Header("Retry-After", strconv.Itoa(retryAfter))
			response.Error(c, http.StatusTooManyRequests, "Too many requests, please try again later", nil)
			c.Abort()
			return
		}

		c.Next()
	}
}
