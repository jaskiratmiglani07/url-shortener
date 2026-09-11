package model

import (
	"time"
)

// URL represents a shortened URL entry in the database.
type URL struct {
	ID          int64      `json:"id" db:"id"`
	ShortCode   string     `json:"short_code" db:"short_code"`
	OriginalURL string     `json:"original_url" db:"original_url"`
	CreatedAt   time.Time  `json:"created_at" db:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty" db:"expires_at"`
}

// IsExpired returns true if the URL has an expiration date set and it is before current time.
func (u *URL) IsExpired() bool {
	if u.ExpiresAt == nil {
		return false
	}
	return time.Now().UTC().After(*u.ExpiresAt)
}

// ShortenRequest represents the payload for creating a new short URL.
type ShortenRequest struct {
	URL              string     `json:"url"`
	CustomAlias      string     `json:"custom_alias,omitempty"`
	ExpiresInMinutes *int       `json:"expires_in_minutes,omitempty"`
	ExpiresAt        *time.Time `json:"expires_at,omitempty"`
}

// ShortenResponse represents the API response when a URL is shortened.
type ShortenResponse struct {
	ShortCode   string     `json:"short_code"`
	ShortURL    string     `json:"short_url"`
	OriginalURL string     `json:"original_url"`
	CreatedAt   time.Time  `json:"created_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
}

// AnalyticsResponse represents the click analytics data for a short URL.
type AnalyticsResponse struct {
	ShortCode       string     `json:"short_code"`
	OriginalURL     string     `json:"original_url"`
	CreatedAt       time.Time  `json:"created_at"`
	ExpiresAt       *time.Time `json:"expires_at,omitempty"`
	TotalClicks     int64      `json:"total_clicks"`
	ClicksToday     int64      `json:"clicks_today"`
	ClicksLast7Days int64      `json:"clicks_last_7_days"`
	LastClickedAt   *time.Time `json:"last_clicked_at,omitempty"`
}
