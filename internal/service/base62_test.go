package service

import (
	"testing"
)

func TestEncodeDecodeBase62(t *testing.T) {
	testCases := []struct {
		id       uint64
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{61, "Z"},
		{62, "10"},
		{125, "21"},
		{1000000, "4c92"},
		{18446744073709551615, "lYGhA16ahyf"}, // Max uint64
	}

	for _, tc := range testCases {
		encoded := EncodeBase62(tc.id)
		if encoded != tc.expected {
			t.Errorf("EncodeBase62(%d) = %s; expected %s", tc.id, encoded, tc.expected)
		}

		decoded, err := DecodeBase62(encoded)
		if err != nil {
			t.Errorf("DecodeBase62(%s) unexpected error: %v", encoded, err)
		}
		if decoded != tc.id {
			t.Errorf("DecodeBase62(%s) = %d; expected %d", encoded, decoded, tc.id)
		}
	}
}

func TestDecodeBase62Invalid(t *testing.T) {
	// Empty string
	if _, err := DecodeBase62(""); err != ErrEmptyBase62String {
		t.Errorf("expected ErrEmptyBase62String for empty input, got %v", err)
	}

	// Invalid characters
	invalidCases := []string{"abc-123", "hello@world", "foo!", "test space"}
	for _, s := range invalidCases {
		if _, err := DecodeBase62(s); err != ErrInvalidBase62Char {
			t.Errorf("expected ErrInvalidBase62Char for %q, got %v", s, err)
		}
	}
}

func TestGenerateRandomCode(t *testing.T) {
	code, err := GenerateRandomCode(8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(code) != 8 {
		t.Errorf("expected length 8, got %d (%s)", len(code), code)
	}

	// Generating a second one should produce a distinct code
	code2, err := GenerateRandomCode(8)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if code == code2 {
		t.Errorf("expected random codes to differ, got identical codes: %s", code)
	}
}
