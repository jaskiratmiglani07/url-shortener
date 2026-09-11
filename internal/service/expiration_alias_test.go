package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/jaskiratmiglani07/url-shortener/internal/model"
	"github.com/jaskiratmiglani07/url-shortener/internal/store"
)

func TestCustomAliasThorough(t *testing.T) {
	ctx := context.Background()
	memStore := store.NewMemoryStore()
	svc := NewShortenerService(memStore, "http://localhost:8080")

	// 1. Test reserved system aliases
	reservedWords := []string{"api", "analytics", "preview", "shorten", "health", "dashboard", "admin"}
	for _, word := range reservedWords {
		req := &model.ShortenRequest{
			URL:         "https://example.com",
			CustomAlias: word,
		}
		_, err := svc.Shorten(ctx, req)
		if !errors.Is(err, ErrReservedAlias) {
			t.Errorf("expected ErrReservedAlias for %q, got %v", word, err)
		}
	}

	// 2. Test boundary lengths (3 valid, 32 valid, 2 invalid, 33 invalid)
	valid3 := &model.ShortenRequest{URL: "https://example.com", CustomAlias: "abc"}
	resp, err := svc.Shorten(ctx, valid3)
	if err != nil || resp.ShortCode != "abc" {
		t.Fatalf("expected 3-char alias to succeed, got %v", err)
	}

	valid32 := &model.ShortenRequest{URL: "https://example.com", CustomAlias: strings.Repeat("x", 32)}
	resp, err = svc.Shorten(ctx, valid32)
	if err != nil || resp.ShortCode != strings.Repeat("x", 32) {
		t.Fatalf("expected 32-char alias to succeed, got %v", err)
	}

	invalid2 := &model.ShortenRequest{URL: "https://example.com", CustomAlias: "ab"}
	_, err = svc.Shorten(ctx, invalid2)
	if !errors.Is(err, ErrInvalidAlias) {
		t.Errorf("expected ErrInvalidAlias for 2 chars, got %v", err)
	}

	invalid33 := &model.ShortenRequest{URL: "https://example.com", CustomAlias: strings.Repeat("x", 33)}
	_, err = svc.Shorten(ctx, invalid33)
	if !errors.Is(err, ErrInvalidAlias) {
		t.Errorf("expected ErrInvalidAlias for 33 chars, got %v", err)
	}
}

func TestURLExpirationThorough(t *testing.T) {
	ctx := context.Background()
	memStore := store.NewMemoryStore()
	svc := NewShortenerService(memStore, "http://localhost:8080")

	// 1. Negative or zero minutes should error
	zero := 0
	_, err := svc.Shorten(ctx, &model.ShortenRequest{
		URL:              "https://example.com",
		ExpiresInMinutes: &zero,
	})
	if err == nil {
		t.Errorf("expected error for ExpiresInMinutes <= 0, got nil")
	}

	// 2. Setting future ExpiresInMinutes
	tenMins := 10
	resp, err := svc.Shorten(ctx, &model.ShortenRequest{
		URL:              "https://example.com",
		CustomAlias:      "ten-min-link",
		ExpiresInMinutes: &tenMins,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.ExpiresAt == nil {
		t.Fatalf("expected non-nil ExpiresAt")
	}
	if resp.IsExpired {
		t.Errorf("link should not be expired yet")
	}

	// 3. Resolve should succeed when active
	resolved, err := svc.Resolve(ctx, "ten-min-link")
	if err != nil {
		t.Fatalf("failed to resolve active link: %v", err)
	}
	if resolved.OriginalURL != "https://example.com" {
		t.Errorf("expected https://example.com, got %s", resolved.OriginalURL)
	}

	// 4. Manually expire link and verify resolution fails with ErrURLExpired
	past := time.Now().UTC().Add(-1 * time.Minute)
	_ = memStore.SetExpiration("ten-min-link", &past)

	_, err = svc.Resolve(ctx, "ten-min-link")
	if !errors.Is(err, store.ErrURLExpired) {
		t.Errorf("expected ErrURLExpired, got %v", err)
	}

	// 5. Analytics should report IsExpired = true
	analytics, err := svc.GetAnalytics(ctx, "ten-min-link")
	if err != nil {
		t.Fatalf("unexpected analytics error: %v", err)
	}
	if !analytics.IsExpired {
		t.Errorf("expected analytics to report IsExpired true")
	}
}
