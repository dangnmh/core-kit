package idgen_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/dangnmh/core-kit/pkg/idgen"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPrefixedID(t *testing.T) {
	t.Parallel()

	id := idgen.NewPrefixedID("msg", 8)
	if !strings.HasPrefix(id, "msg_") {
		t.Fatalf("expected prefix msg_, got %s", id)
	}

	id2 := idgen.NewPrefixedID("msg", 8)
	if id == id2 {
		t.Fatal("expected random IDs to be unique")
	}

	raw := idgen.NewPrefixedID("", 4)
	if strings.Contains(raw, "_") {
		t.Fatalf("expected no underscore, got %s", raw)
	}
}

func TestNanoID_DefaultAlphabet(t *testing.T) {
	t.Parallel()

	alphabetRegex := regexp.MustCompile(`^[a-zA-Z0-9]{21}$`)

	for range 100 {
		id := idgen.NanoID(21)
		assert.Len(t, id, 21)
		assert.True(t, alphabetRegex.MatchString(id), "NanoID should only contain alphanumeric characters: %s", id)
	}
}

func TestNanoID_Uniqueness(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{}, 1000)
	for range 1000 {
		id := idgen.NanoID(21)
		_, exists := seen[id]
		require.False(t, exists, "collision detected for NanoID: %s", id)
		seen[id] = struct{}{}
	}
}

func TestNanoID_EdgeCases(t *testing.T) {
	t.Parallel()

	assert.Empty(t, idgen.NanoID(0))
	assert.Empty(t, idgen.NanoID(-5))
	assert.Empty(t, idgen.NanoIDWithAlphabet(10, ""))
}

func TestNanoIDWithAlphabet_Custom(t *testing.T) {
	t.Parallel()

	customAlphabet := "ABC"
	customRegex := regexp.MustCompile(`^[ABC]{15}$`)

	for range 50 {
		id := idgen.NanoIDWithAlphabet(15, customAlphabet)
		assert.Len(t, id, 15)
		assert.True(t, customRegex.MatchString(id))
	}
}
