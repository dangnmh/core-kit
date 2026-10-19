package strutil

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

var (
	nonAlphaNumSlugRegex = regexp.MustCompile(`[^a-z0-9]+`)
	vnDiacriticRegex     = regexp.MustCompile(`(?i)[àáạảãâầấậẩẫăằắặẳẵèéẹẻẽêềếệểễìíịỉĩòóọỏõôồốộổỗơờớợởỡùúụủũưừứựửữỳýỵỷỹđ]`)
	englishWordRegex     = regexp.MustCompile(`(?i)\b(?:the|and|for|with|your|you|app|tool|free|online|create|build|generate|fast|smart|modern|manage|track|ai|platform|dashboard|studio|generator|assistant|extension|developer|workflow|code|search|analytics|design)\b`)
)

// TruncateRunes cuts a UTF-8 string to at most maxRunes without splitting multi-byte runes (e.g. Vietnamese accents, emojis).
func TruncateRunes(s string, maxRunes int) string {
	if maxRunes <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= maxRunes {
		return s
	}

	count := 0
	for i := range s {
		if count == maxRunes {
			return s[:i]
		}
		count++
	}
	return s
}

// Slugify converts text (including Vietnamese diacritics and đ/Đ) into a lowercase URL-friendly kebab-case slug.
func Slugify(s string) string {
	lower := strings.ToLower(strings.TrimSpace(s))
	if lower == "" {
		return ""
	}
	lower = strings.ReplaceAll(lower, "đ", "d")
	lower = strings.ReplaceAll(lower, "Đ", "d")

	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	normalized, _, _ := transform.String(t, lower)

	slug := nonAlphaNumSlugRegex.ReplaceAllString(normalized, "-")
	return strings.Trim(slug, "-")
}

// UniqueStrings trims, filters empty strings, and deduplicates a string slice while preserving order.
func UniqueStrings(input []string) []string {
	seen := make(map[string]struct{}, len(input))
	out := make([]string, 0, len(input))
	for _, s := range input {
		trimmed := strings.TrimSpace(s)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; !exists {
			seen[trimmed] = struct{}{}
			out = append(out, trimmed)
		}
	}
	return out
}

// AppendUniqueTrimmed appends strings.TrimSpace(val) to slice only if non-empty and not already present.
func AppendUniqueTrimmed(slice []string, val string) []string {
	trimmed := strings.TrimSpace(val)
	if trimmed == "" {
		return slice
	}
	for _, item := range slice {
		if item == trimmed {
			return slice
		}
	}
	return append(slice, trimmed)
}

// ContainsVietnameseDiacritics returns true if text contains any Vietnamese diacritic characters.
func ContainsVietnameseDiacritics(text string) bool {
	return vnDiacriticRegex.MatchString(text)
}

// IsEnglishText returns true if text has >= 10 chars, contains common English words, and has no Vietnamese diacritics.
func IsEnglishText(text string) bool {
	trimmed := strings.TrimSpace(text)
	if len(trimmed) < 10 {
		return false
	}
	return !vnDiacriticRegex.MatchString(trimmed) && englishWordRegex.MatchString(trimmed)
}

// SHA256Hex computes the lowercase hex-encoded SHA-256 digest of data.
func SHA256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// SHA256HexString computes the lowercase hex-encoded SHA-256 digest of string s.
func SHA256HexString(s string) string {
	return SHA256Hex([]byte(s))
}

// HMACSHA256 computes the HMAC-SHA256 MAC of data using key.
func HMACSHA256(key []byte, data string) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write([]byte(data))
	return mac.Sum(nil)
}
