package store

import (
	"context"
	"testing"
	"time"

	"github.com/jaskiratmiglani07/url-shortener/internal/model"
	"github.com/redis/go-redis/v9"
)

func TestCachedStore_GracefulDegradationWithoutRedis(t *testing.T) {
	ctx := context.Background()
	memStore := NewMemoryStore()

	// CachedStore with nil redis client (simulating Redis not being enabled / down)
	cached := NewCachedStore(memStore, nil, 10*time.Minute, nil)

	u := &model.URL{
		ShortCode:   "no-redis-link",
		OriginalURL: "https://example.com/graceful",
		CreatedAt:   time.Now().UTC(),
	}

	// Save should succeed directly on primary store
	if err := cached.Save(ctx, u); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}

	// Lookup should succeed
	fetched, err := cached.GetByShortCode(ctx, "no-redis-link")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if fetched.OriginalURL != u.OriginalURL {
		t.Errorf("expected %s, got %s", u.OriginalURL, fetched.OriginalURL)
	}

	// Close should succeed cleanly
	if err := cached.Close(); err != nil {
		t.Errorf("unexpected close error: %v", err)
	}
}

func TestCachedStore_ExpirationInCache(t *testing.T) {
	ctx := context.Background()
	memStore := NewMemoryStore()

	cached := NewCachedStore(memStore, nil, 10*time.Minute, nil)

	past := time.Now().UTC().Add(-5 * time.Minute)
	expiredURL := &model.URL{
		ShortCode:   "expired-link",
		OriginalURL: "https://example.com/expired",
		CreatedAt:   time.Now().UTC().Add(-10 * time.Minute),
		ExpiresAt:   &past,
	}

	_ = memStore.Save(ctx, expiredURL)

	fetched, err := cached.GetByShortCode(ctx, "expired-link")
	if err != nil {
		t.Fatalf("unexpected error fetching from store: %v", err)
	}
	if !fetched.IsExpired() {
		t.Errorf("expected URL to report as expired")
	}
}

func TestCachedStore_WithLocalRedisIfAvailable(t *testing.T) {
	// Attempt to connect to local redis if one is running
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		t.Skip("local redis server is not running; skipping integration test")
		return
	}
	defer client.Close()

	memStore := NewMemoryStore()
	cached := NewCachedStore(memStore, client, 5*time.Minute, nil)

	shortCode := "redis-test-code"
	u := &model.URL{
		ShortCode:   shortCode,
		OriginalURL: "https://redis-test.org",
		CreatedAt:   time.Now().UTC(),
	}

	// Save writes through to Redis
	if err := cached.Save(ctx, u); err != nil {
		t.Fatalf("save error: %v", err)
	}

	// Verify key exists in Redis
	val, err := client.Get(ctx, "url:"+shortCode).Result()
	if err != nil {
		t.Fatalf("redis key was not set: %v", err)
	}
	if len(val) == 0 {
		t.Errorf("expected non-empty cached value")
	}

	// Delete from underlying memStore to prove subsequent Get comes from Redis cache!
	memStore.mu.Lock()
	delete(memStore.urls, shortCode)
	memStore.mu.Unlock()

	cachedHit, err := cached.GetByShortCode(ctx, shortCode)
	if err != nil {
		t.Fatalf("expected cache hit from redis: %v", err)
	}
	if cachedHit.OriginalURL != u.OriginalURL {
		t.Errorf("expected %s from cache, got %s", u.OriginalURL, cachedHit.OriginalURL)
	}

	// Cleanup
	_ = client.Del(ctx, "url:"+shortCode)
}
