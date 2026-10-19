package idgen

import (
	"strings"
	"unicode"

	"github.com/google/uuid"
)

// NewUUID generates a new RFC 4122 UUID v4 string using github.com/google/uuid.
func NewUUID() string {
	return uuid.NewString()
}

// IsValidUUID checks whether id is a valid canonical UUID using github.com/google/uuid.
func IsValidUUID(id string) bool {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return false
	}
	return uuid.Validate(trimmed) == nil
}

// IsValidUUIDOrSeeded checks whether id is a valid RFC 4122 UUID or a deterministic
// test/seed UUID whose first character is one of allowedPrefixRunes (e.g., 'p', 'c', 'i').
func IsValidUUIDOrSeeded(id string, allowedPrefixRunes ...rune) bool {
	trimmed := strings.TrimSpace(id)
	if trimmed == "" {
		return false
	}
	if uuid.Validate(trimmed) == nil {
		return true
	}
	if len(trimmed) != 36 || len(allowedPrefixRunes) == 0 {
		return false
	}
	first := unicode.ToLower(rune(trimmed[0]))
	for _, allowed := range allowedPrefixRunes {
		if first == unicode.ToLower(allowed) {
			return uuid.Validate("0"+trimmed[1:]) == nil
		}
	}
	return false
}
