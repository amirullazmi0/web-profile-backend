package redis

import (
	"context"
	"fmt"
	"time"

	"github.com/amirullazmi0/web-profile-backend/config"
	"github.com/amirullazmi0/web-profile-backend/pkg/logger"
	goredis "github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type Client struct {
	Client *goredis.Client
}

// NewClient creates a new Redis client and validates the connection.
func NewClient(cfg *config.RedisConfig) (*Client, error) {
	client := goredis.NewClient(&goredis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to ping redis: %w", err)
	}

	logger.Info("Redis connected successfully", zap.String("addr", cfg.Addr), zap.Int("db", cfg.DB))

	return &Client{Client: client}, nil
}

// Close closes the Redis connection.
func (c *Client) Close() error {
	if c == nil || c.Client == nil {
		return nil
	}
	return c.Client.Close()
}

// Set stores a value in Redis with an expiration.
func (c *Client) Set(ctx context.Context, key string, value any, expiration time.Duration) error {
	return c.Client.Set(ctx, key, value, expiration).Err()
}

// Get fetches a value from Redis.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	return c.Client.Get(ctx, key).Result()
}

// Del removes a key from Redis.
func (c *Client) Del(ctx context.Context, key string) error {
	return c.Client.Del(ctx, key).Err()
}
