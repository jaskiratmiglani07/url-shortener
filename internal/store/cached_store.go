package store

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jaskiratmiglani07/url-shortener/internal/model"
	"github.com/redis/go-redis/v9"
)

const (
	cacheKeyPrefix = "url:"
)

// CachedStore wraps any URLStore with a Redis cache-aside layer.
// It fails gracefully back to the underlying store if Redis is unavailable.
type CachedStore struct {
	underlying URLStore
	rdb        *redis.Client
	defaultTTL time.Duration
	logger     *slog.Logger
}

// NewCachedStore creates a new CachedStore wrapper.
func NewCachedStore(underlying URLStore, rdb *redis.Client, defaultTTL time.Duration, logger *slog.Logger) *CachedStore {
	if logger == nil {
		logger = slog.Default()
	}
	return &CachedStore{
		underlying: underlying,
		rdb:        rdb,
		defaultTTL: defaultTTL,
		logger:     logger,
	}
}

// Save persists to the primary store and writes to Redis cache.
func (c *CachedStore) Save(ctx context.Context, u *model.URL) error {
	if err := c.underlying.Save(ctx, u); err != nil {
		return err
	}

	// Cache-aside write
	c.setCache(ctx, u)
	return nil
}

// GetByShortCode checks Redis first (cache hit), falling back to underlying store (cache miss).
func (c *CachedStore) GetByShortCode(ctx context.Context, shortCode string) (*model.URL, error) {
	key := cacheKeyPrefix + shortCode

	// Step 1: Check Redis cache
	if c.rdb != nil {
		cachedData, err := c.rdb.Get(ctx, key).Result()
		if err == nil {
			var u model.URL
			if jsonErr := json.Unmarshal([]byte(cachedData), &u); jsonErr == nil {
				if u.IsExpired() {
					// Invalidate expired cache entry
					_ = c.rdb.Del(ctx, key)
					return nil, ErrURLExpired
				}
				return &u, nil
			}
		} else if !errors.Is(err, redis.Nil) {
			// Redis connection/network error -> Log and gracefully degrade
			c.logger.Warn("redis cache read error; falling back to primary store", "key", key, "error", err)
		}
	}

	// Step 2: Cache miss or Redis unavailable -> Query primary database
	u, err := c.underlying.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	// Step 3: Populate Redis cache for subsequent reads
	if !u.IsExpired() {
		c.setCache(ctx, u)
	}

	return u, nil
}

// RecordClick forwards to primary store.
func (c *CachedStore) RecordClick(ctx context.Context, urlID int64) error {
	return c.underlying.RecordClick(ctx, urlID)
}

// GetAnalytics forwards to primary store.
func (c *CachedStore) GetAnalytics(ctx context.Context, shortCode string) (*model.AnalyticsResponse, error) {
	return c.underlying.GetAnalytics(ctx, shortCode)
}

// Ping checks underlying store and Redis if configured.
func (c *CachedStore) Ping(ctx context.Context) error {
	if err := c.underlying.Ping(ctx); err != nil {
		return err
	}
	if c.rdb != nil {
		return c.rdb.Ping(ctx).Err()
	}
	return nil
}

// Close closes both the underlying store and Redis client.
func (c *CachedStore) Close() error {
	var errs []error
	if c.rdb != nil {
		if err := c.rdb.Close(); err != nil {
			errs = append(errs, fmt.Errorf("redis close error: %w", err))
		}
	}
	if err := c.underlying.Close(); err != nil {
		errs = append(errs, fmt.Errorf("primary store close error: %w", err))
	}
	if len(errs) > 0 {
		return errors.Join(errs...)
	}
	return nil
}

// setCache writes the URL to Redis with appropriate TTL.
func (c *CachedStore) setCache(ctx context.Context, u *model.URL) {
	if c.rdb == nil {
		return
	}

	ttl := c.defaultTTL
	if u.ExpiresAt != nil {
		remaining := time.Until(*u.ExpiresAt)
		if remaining <= 0 {
			return // already expired, do not cache
		}
		if remaining < ttl {
			ttl = remaining
		}
	}

	bytes, err := json.Marshal(u)
	if err != nil {
		c.logger.Warn("failed to serialize url for redis cache", "short_code", u.ShortCode, "error", err)
		return
	}

	key := cacheKeyPrefix + u.ShortCode
	if err := c.rdb.Set(ctx, key, bytes, ttl).Err(); err != nil {
		// Non-fatal: primary store already succeeded
		c.logger.Warn("failed to write to redis cache", "key", key, "error", err)
	}
}
