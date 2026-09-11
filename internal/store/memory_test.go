package store

import (
	"context"
	"testing"
	"time"

	"github.com/jaskiratmiglani07/url-shortener/internal/model"
)

func TestMemoryStore(t *testing.T) {
	ctx := context.Background()
	m := NewMemoryStore()

	now := time.Now().UTC()
	u := &model.URL{
		ShortCode:   "test1",
		OriginalURL: "https://golang.org",
		CreatedAt:   now,
	}

	// Save
	if err := m.Save(ctx, u); err != nil {
		t.Fatalf("unexpected save error: %v", err)
	}
	if u.ID != 1 {
		t.Errorf("expected ID 1, got %d", u.ID)
	}

	// Duplicate save
	err := m.Save(ctx, &model.URL{ShortCode: "test1", OriginalURL: "https://google.com"})
	if err != ErrAlreadyExists {
		t.Errorf("expected ErrAlreadyExists, got %v", err)
	}

	// GetByShortCode
	fetched, err := m.GetByShortCode(ctx, "test1")
	if err != nil {
		t.Fatalf("unexpected get error: %v", err)
	}
	if fetched.OriginalURL != "https://golang.org" {
		t.Errorf("expected URL https://golang.org, got %s", fetched.OriginalURL)
	}

	// Record clicks and verify analytics
	_ = m.RecordClick(ctx, u.ID)
	_ = m.RecordClick(ctx, u.ID)

	analytics, err := m.GetAnalytics(ctx, "test1")
	if err != nil {
		t.Fatalf("unexpected analytics error: %v", err)
	}
	if analytics.TotalClicks != 2 {
		t.Errorf("expected 2 total clicks, got %d", analytics.TotalClicks)
	}
	if analytics.ClicksToday != 2 {
		t.Errorf("expected 2 clicks today, got %d", analytics.ClicksToday)
	}
	if analytics.LastClickedAt == nil {
		t.Errorf("expected last clicked timestamp, got nil")
	}

	// Not found
	_, err = m.GetByShortCode(ctx, "nonexistent")
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}
