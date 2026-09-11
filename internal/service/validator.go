package service

import (
	"errors"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

var (
	ErrEmptyURL         = errors.New("url cannot be empty")
	ErrURLTooLong       = errors.New("url exceeds maximum allowed length of 2048 characters")
	ErrInvalidURL       = errors.New("invalid url format")
	ErrInvalidScheme    = errors.New("url must use http or https protocol")
	ErrMissingHost      = errors.New("url must have a valid host domain")
	ErrInvalidAlias     = errors.New("custom alias must be 3-32 alphanumeric characters (hyphens and underscores allowed)")
	ErrReservedAlias    = errors.New("custom alias is a reserved system path and cannot be used")
	ErrExpiredTimestamp = errors.New("expiration time must be in the future")
)

var aliasRegex = regexp.MustCompile(`^[a-zA-Z0-9_-]{3,32}$`)

var reservedAliases = map[string]struct{}{
	"api":         {},
	"analytics":   {},
	"preview":     {},
	"shorten":     {},
	"health":      {},
	"healthz":     {},
	"metrics":     {},
	"static":      {},
	"assets":      {},
	"favicon.ico": {},
	"robots.txt":  {},
	"docs":        {},
	"swagger":     {},
	"dashboard":   {},
	"admin":       {},
}

// ValidateURL ensures the long URL is properly formatted, has a valid scheme, and a valid host.
func ValidateURL(rawURL string, serviceHost string) error {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return ErrEmptyURL
	}
	if len(trimmed) > 2048 {
		return ErrURLTooLong
	}

	parsed, err := url.ParseRequestURI(trimmed)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidURL, err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return ErrInvalidScheme
	}

	if parsed.Host == "" {
		return ErrMissingHost
	}

	// Prevent self-referencing redirect loops
	if serviceHost != "" {
		cleanHost := strings.TrimPrefix(strings.TrimPrefix(serviceHost, "http://"), "https://")
		cleanHost = strings.TrimRight(cleanHost, "/")
		if strings.EqualFold(parsed.Host, cleanHost) {
			return errors.New("cannot shorten URL pointing to this service domain")
		}
	}

	return nil
}

// ValidateCustomAlias ensures custom alias conforms to allowed length, character set, and is not reserved.
func ValidateCustomAlias(alias string) error {
	trimmed := strings.TrimSpace(alias)
	if trimmed == "" {
		return nil // custom alias is optional
	}

	if !aliasRegex.MatchString(trimmed) {
		return ErrInvalidAlias
	}

	lower := strings.ToLower(trimmed)
	if _, reserved := reservedAliases[lower]; reserved {
		return ErrReservedAlias
	}

	return nil
}
