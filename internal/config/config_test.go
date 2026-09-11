package config

import (
	"os"
	"testing"
	"time"
)

func TestConfigDefaults(t *testing.T) {
	// Clear any relevant env vars
	keys := []string{
		"SERVER_PORT", "BASE_URL", "LOG_LEVEL", "DATABASE_URL",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME_MINUTES",
		"REDIS_URL", "CACHE_TTL_MINUTES", "RATE_LIMIT_ENABLED", "RATE_LIMIT_RPS", "RATE_LIMIT_BURST",
	}
	for _, k := range keys {
		os.Unsetenv(k)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.ServerPort != "8080" {
		t.Errorf("expected ServerPort 8080, got %s", cfg.ServerPort)
	}
	if cfg.BaseURL != "http://localhost:8080" {
		t.Errorf("expected BaseURL http://localhost:8080, got %s", cfg.BaseURL)
	}
	if cfg.LogLevel != "info" {
		t.Errorf("expected LogLevel info, got %s", cfg.LogLevel)
	}
	if cfg.CacheTTL != 60*time.Minute {
		t.Errorf("expected CacheTTL 60m, got %v", cfg.CacheTTL)
	}
	if !cfg.RateLimitEnabled {
		t.Errorf("expected RateLimitEnabled true, got false")
	}
	if cfg.RateLimitRPS != 10 {
		t.Errorf("expected RateLimitRPS 10, got %d", cfg.RateLimitRPS)
	}
	if cfg.RateLimitBurst != 20 {
		t.Errorf("expected RateLimitBurst 20, got %d", cfg.RateLimitBurst)
	}
}

func TestConfigEnvOverride(t *testing.T) {
	os.Setenv("SERVER_PORT", "9090")
	os.Setenv("BASE_URL", "https://short.io/")
	os.Setenv("RATE_LIMIT_ENABLED", "false")
	os.Setenv("RATE_LIMIT_RPS", "50")
	defer func() {
		os.Unsetenv("SERVER_PORT")
		os.Unsetenv("BASE_URL")
		os.Unsetenv("RATE_LIMIT_ENABLED")
		os.Unsetenv("RATE_LIMIT_RPS")
	}()

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error loading config: %v", err)
	}

	if cfg.ServerPort != "9090" {
		t.Errorf("expected ServerPort 9090, got %s", cfg.ServerPort)
	}
	if cfg.BaseURL != "https://short.io" {
		t.Errorf("expected BaseURL https://short.io, got %s", cfg.BaseURL)
	}
	if cfg.RateLimitEnabled {
		t.Errorf("expected RateLimitEnabled false, got true")
	}
	if cfg.RateLimitRPS != 50 {
		t.Errorf("expected RateLimitRPS 50, got %d", cfg.RateLimitRPS)
	}
}
