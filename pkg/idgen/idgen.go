package idgen

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
)

// NewPrefixedID generates a random hex ID with an optional prefix, e.g. "evt_a1b2c3d4".
// If byteLen <= 0, 8 bytes (16 hex chars) is used as default.
func NewPrefixedID(prefix string, byteLen ...int) string {
	length := 8
	if len(byteLen) > 0 && byteLen[0] > 0 {
		length = byteLen[0]
	}

	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		panic(fmt.Errorf("idgen: CSPRNG failure: %w", err))
	}

	if prefix == "" {
		return hex.EncodeToString(b)
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(b))
}
