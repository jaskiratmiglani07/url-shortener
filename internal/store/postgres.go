package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/jaskiratmiglani07/url-shortener/internal/config"
	"github.com/jaskiratmiglani07/url-shortener/internal/model"
	"github.com/lib/pq"
)

// PostgresStore implements URLStore backed by a PostgreSQL database.
type PostgresStore struct {
	db *sql.DB
}

// NewPostgresStore creates and verifies a connection to PostgreSQL.
func NewPostgresStore(cfg *config.Config) (*PostgresStore, error) {
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open postgres database: %w", err)
	}

	db.SetMaxOpenConns(cfg.DBMaxOpenConns)
	db.SetMaxIdleConns(cfg.DBMaxIdleConns)
	db.SetConnMaxLifetime(cfg.DBConnMaxLifetime)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	s := &PostgresStore{db: db}
	if err := s.AutoMigrate(ctx); err != nil {
		return nil, fmt.Errorf("failed to apply migrations: %w", err)
	}

	return s, nil
}

// AutoMigrate creates the required tables and indexes if they do not exist.
func (s *PostgresStore) AutoMigrate(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS urls (
			id BIGSERIAL PRIMARY KEY,
			short_code VARCHAR(64) NOT NULL,
			original_url TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			expires_at TIMESTAMPTZ NULL,
			CONSTRAINT uq_urls_short_code UNIQUE (short_code)
		);`,
		`CREATE INDEX IF NOT EXISTS idx_urls_short_code ON urls (short_code);`,
		`CREATE INDEX IF NOT EXISTS idx_urls_expires_at ON urls (expires_at) WHERE expires_at IS NOT NULL;`,
		`CREATE TABLE IF NOT EXISTS url_clicks (
			id BIGSERIAL PRIMARY KEY,
			url_id BIGINT NOT NULL,
			clicked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			CONSTRAINT fk_url_clicks_url FOREIGN KEY (url_id) REFERENCES urls(id) ON DELETE CASCADE
		);`,
		`CREATE INDEX IF NOT EXISTS idx_url_clicks_url_id_clicked_at ON url_clicks (url_id, clicked_at DESC);`,
	}

	for _, query := range queries {
		if _, err := s.db.ExecContext(ctx, query); err != nil {
			return err
		}
	}

	return nil
}

// Save inserts a new URL entry.
func (s *PostgresStore) Save(ctx context.Context, u *model.URL) error {
	query := `
		INSERT INTO urls (short_code, original_url, created_at, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING id;
	`

	err := s.db.QueryRowContext(ctx, query, u.ShortCode, u.OriginalURL, u.CreatedAt, u.ExpiresAt).Scan(&u.ID)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == "23505" {
			return ErrAlreadyExists
		}
		return fmt.Errorf("failed to save url: %w", err)
	}

	return nil
}

// GetByShortCode retrieves a URL by its short code.
func (s *PostgresStore) GetByShortCode(ctx context.Context, shortCode string) (*model.URL, error) {
	query := `
		SELECT id, short_code, original_url, created_at, expires_at
		FROM urls
		WHERE short_code = $1;
	`

	var u model.URL
	var expiresAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, shortCode).Scan(
		&u.ID,
		&u.ShortCode,
		&u.OriginalURL,
		&u.CreatedAt,
		&expiresAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get url by short code: %w", err)
	}

	if expiresAt.Valid {
		u.ExpiresAt = &expiresAt.Time
	}

	return &u, nil
}

// RecordClick records a timestamped click event.
func (s *PostgresStore) RecordClick(ctx context.Context, urlID int64) error {
	query := `
		INSERT INTO url_clicks (url_id, clicked_at)
		VALUES ($1, NOW());
	`
	_, err := s.db.ExecContext(ctx, query, urlID)
	if err != nil {
		return fmt.Errorf("failed to record click: %w", err)
	}
	return nil
}

// GetAnalytics calculates click statistics for a short URL.
func (s *PostgresStore) GetAnalytics(ctx context.Context, shortCode string) (*model.AnalyticsResponse, error) {
	query := `
		SELECT 
			u.short_code,
			u.original_url,
			u.created_at,
			u.expires_at,
			COUNT(c.id) AS total_clicks,
			COUNT(c.id) FILTER (WHERE c.clicked_at >= date_trunc('day', NOW())) AS clicks_today,
			COUNT(c.id) FILTER (WHERE c.clicked_at >= NOW() - INTERVAL '7 days') AS clicks_last_7_days,
			MAX(c.clicked_at) AS last_clicked_at
		FROM urls u
		LEFT JOIN url_clicks c ON u.id = c.url_id
		WHERE u.short_code = $1
		GROUP BY u.id, u.short_code, u.original_url, u.created_at, u.expires_at;
	`

	var resp model.AnalyticsResponse
	var expiresAt sql.NullTime
	var lastClickedAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, shortCode).Scan(
		&resp.ShortCode,
		&resp.OriginalURL,
		&resp.CreatedAt,
		&expiresAt,
		&resp.TotalClicks,
		&resp.ClicksToday,
		&resp.ClicksLast7Days,
		&lastClickedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get analytics: %w", err)
	}

	if expiresAt.Valid {
		resp.ExpiresAt = &expiresAt.Time
		resp.IsExpired = time.Now().UTC().After(expiresAt.Time)
	}
	if lastClickedAt.Valid {
		resp.LastClickedAt = &lastClickedAt.Time
	}

	return &resp, nil
}

// Ping checks PostgreSQL connectivity.
func (s *PostgresStore) Ping(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// Close closes the underlying DB connection.
func (s *PostgresStore) Close() error {
	return s.db.Close()
}
