package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jaskiratmiglani07/url-shortener/internal/model"
	"github.com/jaskiratmiglani07/url-shortener/internal/store"
)

var (
	ErrCodeGenerationFailed = errors.New("failed to generate a unique short code")
)

// ShortenerService handles business logic for URL shortening, redirection, and analytics.
type ShortenerService struct {
	store   store.URLStore
	baseURL string
}

// NewShortenerService instantiates a new ShortenerService.
func NewShortenerService(store store.URLStore, baseURL string) *ShortenerService {
	return &ShortenerService{
		store:   store,
		baseURL: strings.TrimRight(baseURL, "/"),
	}
}

// Shorten creates a new shortened URL according to the request parameters.
func (s *ShortenerService) Shorten(ctx context.Context, req *model.ShortenRequest) (*model.ShortenResponse, error) {
	if err := ValidateURL(req.URL, s.baseURL); err != nil {
		return nil, err
	}

	alias := strings.TrimSpace(req.CustomAlias)
	if alias != "" {
		if err := ValidateCustomAlias(alias); err != nil {
			return nil, err
		}
	}

	var expiresAt *time.Time
	now := time.Now().UTC()

	if req.ExpiresInMinutes != nil {
		if *req.ExpiresInMinutes <= 0 {
			return nil, errors.New("expires_in_minutes must be greater than 0")
		}
		exp := now.Add(time.Duration(*req.ExpiresInMinutes) * time.Minute)
		expiresAt = &exp
	} else if req.ExpiresAt != nil {
		if !req.ExpiresAt.After(now) {
			return nil, ErrExpiredTimestamp
		}
		exp := req.ExpiresAt.UTC()
		expiresAt = &exp
	}

	// Case 1: User requested a custom alias
	if alias != "" {
		u := &model.URL{
			ShortCode:   alias,
			OriginalURL: req.URL,
			CreatedAt:   now,
			ExpiresAt:   expiresAt,
		}
		if err := s.store.Save(ctx, u); err != nil {
			return nil, err
		}
		return s.buildResponse(u), nil
	}

	// Case 2: Auto-generate unique short code with collision retries
	const maxRetries = 5
	for attempt := 0; attempt < maxRetries; attempt++ {
		code, err := GenerateRandomCode(7)
		if err != nil {
			return nil, fmt.Errorf("failed to generate random code: %w", err)
		}

		u := &model.URL{
			ShortCode:   code,
			OriginalURL: req.URL,
			CreatedAt:   now,
			ExpiresAt:   expiresAt,
		}

		err = s.store.Save(ctx, u)
		if err == nil {
			return s.buildResponse(u), nil
		}
		if !errors.Is(err, store.ErrAlreadyExists) {
			return nil, err
		}
	}

	return nil, ErrCodeGenerationFailed
}

// Resolve retrieves the URL for redirection and checks for expiration.
func (s *ShortenerService) Resolve(ctx context.Context, shortCode string) (*model.URL, error) {
	u, err := s.store.GetByShortCode(ctx, shortCode)
	if err != nil {
		return nil, err
	}

	if u.IsExpired() {
		return nil, store.ErrURLExpired
	}

	return u, nil
}

// RecordClick asynchronously or synchronously records a click on a URL.
func (s *ShortenerService) RecordClick(ctx context.Context, urlID int64) error {
	return s.store.RecordClick(ctx, urlID)
}

// GetAnalytics retrieves the click analytics summary for a given short code.
func (s *ShortenerService) GetAnalytics(ctx context.Context, shortCode string) (*model.AnalyticsResponse, error) {
	return s.store.GetAnalytics(ctx, shortCode)
}

func (s *ShortenerService) buildResponse(u *model.URL) *model.ShortenResponse {
	return &model.ShortenResponse{
		ShortCode:   u.ShortCode,
		ShortURL:    fmt.Sprintf("%s/%s", s.baseURL, u.ShortCode),
		OriginalURL: u.OriginalURL,
		CreatedAt:   u.CreatedAt,
		ExpiresAt:   u.ExpiresAt,
		IsExpired:   u.IsExpired(),
	}
}
