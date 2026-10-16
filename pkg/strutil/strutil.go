package strutil

import "unicode/utf8"

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
