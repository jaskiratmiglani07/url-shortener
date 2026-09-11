package database

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/jaskiratmiglani07/url-shortener/internal/config"
	"github.com/redis/go-redis/v9"
)

// NewRedisClient creates and validates a connection to Redis.
// If Redis is unreachable, it returns the client with an error, allowing callers to operate in degraded mode if desired.
func NewRedisClient(cfg *config.Config, logger *slog.Logger) (*redis.Client, error) {
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("redis url is empty")
	}

	opt, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse redis url: %w", err)
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		if logger != nil {
			logger.Warn("redis is currently unavailable; running in degraded cache mode", "error", err)
		}
		return client, fmt.Errorf("failed to ping redis: %w", err)
	}

	if logger != nil {
		logger.Info("connected to redis cache successfully", "addr", opt.Addr)
	}

	return client, nil
}
