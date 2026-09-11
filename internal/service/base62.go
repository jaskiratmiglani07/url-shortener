package service

import (
	"crypto/rand"
	"errors"
	"math/big"
	"strings"
)

const (
	Base62Alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	Base62Length   = 62
)

var (
	ErrInvalidBase62Char = errors.New("invalid character in base62 string")
	ErrEmptyBase62String = errors.New("cannot decode empty base62 string")
)

// EncodeBase62 converts an unsigned 64-bit integer into a Base62 string.
func EncodeBase62(num uint64) string {
	if num == 0 {
		return "0"
	}

	var sb strings.Builder
	for num > 0 {
		rem := num % Base62Length
		sb.WriteByte(Base62Alphabet[rem])
		num /= Base62Length
	}

	// Reverse the builder output
	runes := []rune(sb.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}

	return string(runes)
}

// DecodeBase62 decodes a Base62 string back into a uint64 number.
func DecodeBase62(s string) (uint64, error) {
	if len(s) == 0 {
		return 0, ErrEmptyBase62String
	}

	var result uint64
	for _, r := range s {
		idx := strings.IndexRune(Base62Alphabet, r)
		if idx == -1 {
			return 0, ErrInvalidBase62Char
		}
		result = result*Base62Length + uint64(idx)
	}

	return result, nil
}

// GenerateRandomCode generates a cryptographically secure random Base62 string of the given length.
func GenerateRandomCode(length int) (string, error) {
	if length <= 0 {
		length = 7
	}

	alphabetLen := big.NewInt(Base62Length)
	bytes := make([]byte, length)
	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, alphabetLen)
		if err != nil {
			return "", err
		}
		bytes[i] = Base62Alphabet[num.Int64()]
	}

	return string(bytes), nil
}
