package service

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name        string
		rawURL      string
		serviceHost string
		wantErr     error
	}{
		{
			name:        "valid https url",
			rawURL:      "https://example.com/some/path?query=1#hash",
			serviceHost: "http://localhost:8080",
			wantErr:     nil,
		},
		{
			name:        "valid http url",
			rawURL:      "http://subdomain.test.org/resource",
			serviceHost: "http://localhost:8080",
			wantErr:     nil,
		},
		{
			name:        "empty url",
			rawURL:      "   ",
			serviceHost: "http://localhost:8080",
			wantErr:     ErrEmptyURL,
		},
		{
			name:        "unsupported scheme ftp",
			rawURL:      "ftp://ftp.example.com/file.txt",
			serviceHost: "http://localhost:8080",
			wantErr:     ErrInvalidScheme,
		},
		{
			name:        "unsupported scheme javascript",
			rawURL:      "javascript:alert(1)",
			serviceHost: "http://localhost:8080",
			wantErr:     ErrInvalidScheme,
		},
		{
			name:        "missing scheme",
			rawURL:      "example.com/no-scheme",
			serviceHost: "http://localhost:8080",
			wantErr:     ErrInvalidURL,
		},
		{
			name:        "url too long",
			rawURL:      "https://example.com/" + strings.Repeat("a", 2050),
			serviceHost: "http://localhost:8080",
			wantErr:     ErrURLTooLong,
		},
		{
			name:        "self referencing domain",
			rawURL:      "http://localhost:8080/xyz",
			serviceHost: "http://localhost:8080",
			wantErr:     errors.New("cannot shorten URL pointing to this service domain"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.rawURL, tt.serviceHost)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateURL() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateURL() expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) && err.Error() != tt.wantErr.Error() {
					t.Errorf("ValidateURL() expected error %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestValidateCustomAlias(t *testing.T) {
	tests := []struct {
		name    string
		alias   string
		wantErr error
	}{
		{
			name:    "empty alias is valid (optional)",
			alias:   "",
			wantErr: nil,
		},
		{
			name:    "valid alphanumeric alias",
			alias:   "my-link-123",
			wantErr: nil,
		},
		{
			name:    "valid with underscore",
			alias:   "my_custom_code",
			wantErr: nil,
		},
		{
			name:    "too short alias (2 chars)",
			alias:   "ab",
			wantErr: ErrInvalidAlias,
		},
		{
			name:    "too long alias (33 chars)",
			alias:   strings.Repeat("a", 33),
			wantErr: ErrInvalidAlias,
		},
		{
			name:    "invalid characters (spaces)",
			alias:   "my link",
			wantErr: ErrInvalidAlias,
		},
		{
			name:    "invalid characters (symbols)",
			alias:   "link$123",
			wantErr: ErrInvalidAlias,
		},
		{
			name:    "reserved path analytics",
			alias:   "analytics",
			wantErr: ErrReservedAlias,
		},
		{
			name:    "reserved path api case-insensitive",
			alias:   "API",
			wantErr: ErrReservedAlias,
		},
		{
			name:    "reserved path health",
			alias:   "health",
			wantErr: ErrReservedAlias,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateCustomAlias(tt.alias)
			if tt.wantErr == nil {
				if err != nil {
					t.Errorf("ValidateCustomAlias() unexpected error: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("ValidateCustomAlias() expected error %v, got nil", tt.wantErr)
				} else if !errors.Is(err, tt.wantErr) {
					t.Errorf("ValidateCustomAlias() expected %v, got %v", tt.wantErr, err)
				}
			}
		})
	}
}
