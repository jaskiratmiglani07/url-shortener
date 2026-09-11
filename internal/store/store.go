package store

import (
	"context"
	"errors"

	"github.com/jaskiratmiglani07/url-shortener/internal/model"
)

var (
	ErrNotFound      = errors.New("short url not found")
	ErrAlreadyExists = errors.New("short code already in use")
	ErrURLExpired    = errors.New("short url has expired")
)

// URLStore defines the persistence interface for URLs and click analytics.
type URLStore interface {
	// Save persists a new URL. Returns ErrAlreadyExists if short_code is taken.
	Save(ctx context.Context, u *model.URL) error

	// GetByShortCode finds a URL by its short code. Returns ErrNotFound if not present.
	GetByShortCode(ctx context.Context, shortCode string) (*model.URL, error)

	// RecordClick records a click event for the given URL ID.
	RecordClick(ctx context.Context, urlID int64) error

	// GetAnalytics returns aggregated click statistics for the given short code.
	GetAnalytics(ctx context.Context, shortCode string) (*model.AnalyticsResponse, error)

	// Ping checks if the underlying storage connection is alive.
	Ping(ctx context.Context) error

	// Close terminates the storage connection pool.
	Close() error
}
