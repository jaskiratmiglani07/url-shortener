package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds all configuration for the URL shortener application.
type Config struct {
	ServerPort        string
	BaseURL           string
	LogLevel          string
	DatabaseURL       string
	DBMaxOpenConns    int
	DBMaxIdleConns    int
	DBConnMaxLifetime time.Duration
	RedisURL          string
	CacheTTL          time.Duration
	RateLimitEnabled  bool
	RateLimitRPS      int
	RateLimitBurst    int
}

// Load reads configuration from environment variables, optionally loading a .env file if it exists.
func Load() (*Config, error) {
	loadDotEnv(".env")

	cfg := &Config{
		ServerPort:        getEnv("SERVER_PORT", "8080"),
		BaseURL:           strings.TrimRight(getEnv("BASE_URL", "http://localhost:8080"), "/"),
		LogLevel:          getEnv("LOG_LEVEL", "info"),
		DatabaseURL:       getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/urlshortener?sslmode=disable"),
		DBMaxOpenConns:    getEnvAsInt("DB_MAX_OPEN_CONNS", 25),
		DBMaxIdleConns:    getEnvAsInt("DB_MAX_IDLE_CONNS", 10),
		DBConnMaxLifetime: time.Duration(getEnvAsInt("DB_CONN_MAX_LIFETIME_MINUTES", 30)) * time.Minute,
		RedisURL:          getEnv("REDIS_URL", "redis://localhost:6379/0"),
		CacheTTL:          time.Duration(getEnvAsInt("CACHE_TTL_MINUTES", 60)) * time.Minute,
		RateLimitEnabled:  getEnvAsBool("RATE_LIMIT_ENABLED", true),
		RateLimitRPS:      getEnvAsInt("RATE_LIMIT_RPS", 10),
		RateLimitBurst:    getEnvAsInt("RATE_LIMIT_BURST", 20),
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if val, ok := os.LookupEnv(key); ok && strings.TrimSpace(val) != "" {
		return strings.TrimSpace(val)
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	valStr := getEnv(key, "")
	if val, err := strconv.Atoi(valStr); err == nil {
		return val
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	valStr := strings.ToLower(getEnv(key, ""))
	if valStr == "true" || valStr == "1" || valStr == "yes" {
		return true
	}
	if valStr == "false" || valStr == "0" || valStr == "no" {
		return false
	}
	return fallback
}

// loadDotEnv parses key=value pairs from a file into environment variables without overwriting already set env vars.
func loadDotEnv(filepath string) {
	file, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		val := strings.TrimSpace(parts[1])
		// Remove surrounding quotes if present
		if len(val) >= 2 && ((val[0] == '"' && val[len(val)-1] == '"') || (val[0] == '\'' && val[len(val)-1] == '\'')) {
			val = val[1 : len(val)-1]
		}
		if _, exists := os.LookupEnv(key); !exists {
			os.Setenv(key, val)
		}
	}
}
