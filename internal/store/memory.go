package store

import (
	"context"
	"sync"
	"time"

	"github.com/jaskiratmiglani07/url-shortener/internal/model"
)

// MemoryStore provides a thread-safe in-memory implementation of URLStore for testing.
type MemoryStore struct {
	mu     sync.RWMutex
	urls   map[string]*model.URL
	clicks map[int64][]time.Time
	nextID int64
}

// NewMemoryStore creates an initialized MemoryStore.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		urls:   make(map[string]*model.URL),
		clicks: make(map[int64][]time.Time),
		nextID: 1,
	}
}

func (m *MemoryStore) Save(ctx context.Context, u *model.URL) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.urls[u.ShortCode]; exists {
		return ErrAlreadyExists
	}

	u.ID = m.nextID
	m.nextID++

	// Deep copy to prevent mutation
	stored := *u
	m.urls[u.ShortCode] = &stored
	return nil
}

func (m *MemoryStore) GetByShortCode(ctx context.Context, shortCode string) (*model.URL, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	u, exists := m.urls[shortCode]
	if !exists {
		return nil, ErrNotFound
	}

	copied := *u
	return &copied, nil
}

// SetExpiration is a test helper method to adjust expiration timestamps.
func (m *MemoryStore) SetExpiration(shortCode string, expiresAt *time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	u, exists := m.urls[shortCode]
	if !exists {
		return ErrNotFound
	}
	u.ExpiresAt = expiresAt
	return nil
}

func (m *MemoryStore) RecordClick(ctx context.Context, urlID int64) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.clicks[urlID] = append(m.clicks[urlID], time.Now().UTC())
	return nil
}

func (m *MemoryStore) GetAnalytics(ctx context.Context, shortCode string) (*model.AnalyticsResponse, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	u, exists := m.urls[shortCode]
	if !exists {
		return nil, ErrNotFound
	}

	now := time.Now().UTC()
	startOfToday := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	sevenDaysAgo := now.Add(-7 * 24 * time.Hour)

	timestamps := m.clicks[u.ID]
	var totalClicks int64 = int64(len(timestamps))
	var clicksToday int64
	var clicks7Days int64
	var lastClickedAt *time.Time

	for _, ts := range timestamps {
		if !ts.Before(startOfToday) {
			clicksToday++
		}
		if !ts.Before(sevenDaysAgo) {
			clicks7Days++
		}
		if lastClickedAt == nil || ts.After(*lastClickedAt) {
			tsCopy := ts
			lastClickedAt = &tsCopy
		}
	}

	return &model.AnalyticsResponse{
		ShortCode:       u.ShortCode,
		OriginalURL:     u.OriginalURL,
		CreatedAt:       u.CreatedAt,
		ExpiresAt:       u.ExpiresAt,
		IsExpired:       u.IsExpired(),
		TotalClicks:     totalClicks,
		ClicksToday:     clicksToday,
		ClicksLast7Days: clicks7Days,
		LastClickedAt:   lastClickedAt,
	}, nil
}

func (m *MemoryStore) Ping(ctx context.Context) error {
	return nil
}

func (m *MemoryStore) Close() error {
	return nil
}
