package httplog

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// Strategy defines the masking algorithm applied to sensitive fields.
type Strategy string

const (
	StrategyRedact      Strategy = "redact"
	StrategyPartialMask Strategy = "partial_mask"
	StrategyEmailMask   Strategy = "email_mask"
	StrategyRemove      Strategy = "remove"
	StrategyHashSHA256  Strategy = "hash_sha256"
)

// ApplyStrategy applies the specified masking strategy to a string value.
// It returns the masked value (or nil if removed) and a boolean indicating whether the field should be removed.
func ApplyStrategy(val string, strat Strategy, defaultMask string) (any, bool) {
	if defaultMask == "" {
		defaultMask = "[REDACTED]"
	}

	switch strat {
	case StrategyRemove:
		return nil, true

	case StrategyHashSHA256:
		h := sha256.Sum256([]byte(val))
		return "sha256:" + hex.EncodeToString(h[:]), false

	case StrategyPartialMask:
		runes := []rune(val)
		n := len(runes)
		if n == 0 {
			return "", false
		}
		if n <= 4 {
			return strings.Repeat("*", n), false
		}
		if n <= 6 {
			// Keep first 1 and last 1, mask the middle (e.g. 5 chars: 1***5, 6 chars: 1****6)
			return string(runes[:1]) + strings.Repeat("*", n-2) + string(runes[n-1:]), false
		}
		if n <= 10 {
			// Keep first 3 and last 3, mask the middle (e.g. phone: 098****321)
			return string(runes[:3]) + strings.Repeat("*", n-6) + string(runes[n-3:]), false
		}
		// For credit card / long strings (> 10): keep first 4 and last 4, mask middle
		return string(runes[:4]) + strings.Repeat("*", n-8) + string(runes[n-4:]), false

	case StrategyEmailMask:
		parts := strings.SplitN(val, "@", 2)
		if len(parts) != 2 || len(parts[0]) == 0 || len(parts[1]) == 0 {
			return defaultMask, false
		}
		user := []rune(parts[0])
		domain := parts[1]
		if len(user) <= 2 {
			return string(user[0]) + "***@" + domain, false
		}
		return string(user[0]) + "***" + string(user[len(user)-1]) + "@" + domain, false

	case StrategyRedact:
		fallthrough
	default:
		return defaultMask, false
	}
}
