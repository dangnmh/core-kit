// Package idutil provides cryptographic identifier generation utilities.
// Deprecated: idutil is preserved for backward compatibility; use pkg/idgen directly.
package idutil

import "github.com/dangnmh/core-kit/pkg/idgen"

// Alphabet defines English letters and digits (a-z, A-Z, 0-9).
const Alphabet = idgen.Alphabet

// NanoID generates a secure random NanoID.
// Deprecated: Use pkg/idgen.NanoID directly.
func NanoID(size int) string {
	return idgen.NanoID(size)
}

// NanoIDWithAlphabet generates a secure random NanoID with a custom alphabet.
// Deprecated: Use pkg/idgen.NanoIDWithAlphabet directly.
func NanoIDWithAlphabet(size int, alphabet string) string {
	return idgen.NanoIDWithAlphabet(size, alphabet)
}
