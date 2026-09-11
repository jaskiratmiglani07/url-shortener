package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jaskiratmiglani07/url-shortener/internal/model"
	"github.com/jaskiratmiglani07/url-shortener/internal/store"
)

func TestShortenerService(t *testing.T) {
	ctx := context.Background()
	memStore := store.NewMemoryStore()
	svc := NewShortenerService(memStore, "http://localhost:8080")

	// 1. Auto-generated short URL
	req := &model.ShortenRequest{
		URL: "https://example.com/long/path",
	}
	resp, err := svc.Shorten(ctx, req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.ShortCode) != 7 {
		t.Errorf("expected 7-char short code, got %s", resp.ShortCode)
	}
	if resp.ShortURL != "http://localhost:8080/"+resp.ShortCode {
		t.Errorf("expected ShortURL http://localhost:8080/%s, got %s", resp.ShortCode, resp.ShortURL)
	}

	// 2. Resolve the created URL
	resolved, err := svc.Resolve(ctx, resp.ShortCode)
	if err != nil {
		t.Fatalf("unexpected resolve error: %v", err)
	}
	if resolved.OriginalURL != req.URL {
		t.Errorf("expected %s, got %s", req.URL, resolved.OriginalURL)
	}

	// 3. Custom alias
	aliasReq := &model.ShortenRequest{
		URL:         "https://github.com",
		CustomAlias: "my-gh-link",
	}
	aliasResp, err := svc.Shorten(ctx, aliasReq)
	if err != nil {
		t.Fatalf("unexpected custom alias error: %v", err)
	}
	if aliasResp.ShortCode != "my-gh-link" {
		t.Errorf("expected my-gh-link, got %s", aliasResp.ShortCode)
	}

	// 4. Duplicate custom alias
	_, err = svc.Shorten(ctx, aliasReq)
	if !errors.Is(err, store.ErrAlreadyExists) {
		t.Errorf("expected ErrAlreadyExists on duplicate alias, got %v", err)
	}

	// 5. Expiration (already expired)
	past := time.Now().UTC().Add(-10 * time.Minute)
	expiredReq := &model.ShortenRequest{
		URL:         "https://expired.org",
		CustomAlias: "expired-link",
		ExpiresAt:   &past,
	}
	_, err = svc.Shorten(ctx, expiredReq)
	if !errors.Is(err, ErrExpiredTimestamp) {
		t.Errorf("expected ErrExpiredTimestamp, got %v", err)
	}

	// 6. Valid future expiration that expires later
	minutes := 60
	futureReq := &model.ShortenRequest{
		URL:              "https://future.org",
		CustomAlias:      "future-link",
		ExpiresInMinutes: &minutes,
	}
	futureResp, err := svc.Shorten(ctx, futureReq)
	if err != nil {
		t.Fatalf("unexpected future expiration error: %v", err)
	}
	if futureResp.ExpiresAt == nil {
		t.Fatalf("expected ExpiresAt to be set")
	}

	// 7. Resolve expired URL simulation
	expiredModel := &model.URL{
		ShortCode:   "was-active",
		OriginalURL: "https://wasactive.com",
		CreatedAt:   time.Now().UTC().Add(-2 * time.Hour),
		ExpiresAt:   &past,
	}
	_ = memStore.Save(ctx, expiredModel)
	_, err = svc.Resolve(ctx, "was-active")
	if !errors.Is(err, store.ErrURLExpired) {
		t.Errorf("expected ErrURLExpired, got %v", err)
	}

	// 8. Click and Analytics
	_ = svc.RecordClick(ctx, aliasModelID(memStore, "my-gh-link"))
	analytics, err := svc.GetAnalytics(ctx, "my-gh-link")
	if err != nil {
		t.Fatalf("unexpected analytics error: %v", err)
	}
	if analytics.TotalClicks != 1 {
		t.Errorf("expected 1 click, got %d", analytics.TotalClicks)
	}
}

func aliasModelID(memStore *store.MemoryStore, shortCode string) int64 {
	u, _ := memStore.GetByShortCode(context.Background(), shortCode)
	if u != nil {
		return u.ID
	}
	return 0
}
