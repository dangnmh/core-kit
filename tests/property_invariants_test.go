package tests_test

import (
	"math"
	"math/rand"
	"reflect"
	"testing"
	"testing/quick"

	"github.com/dangnmh/core-kit/pkg/decmath"
	"github.com/dangnmh/core-kit/pkg/idgen"
	"github.com/dangnmh/core-kit/pkg/ringbuffer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProperty_DecmathTickSnapping verifies mathematical properties of decmath tick snapping.
func TestProperty_DecmathTickSnapping(t *testing.T) {
	t.Parallel()

	// 1. Idempotence of SnapToTickFloor: f(f(x)) == f(x)
	idempotenceFloor := func(price float64, tick float64) bool {
		price = math.Abs(price)
		tick = math.Abs(tick)
		if tick <= 0.0001 || tick > 1000 || price > 1e6 || math.IsNaN(price) || math.IsInf(price, 0) {
			return true // Skip invalid domain
		}
		// Scale tick to sensible currency decimals
		tick = decmath.RoundToScale(tick, 4)
		if tick == 0 {
			return true
		}
		snappedOnce := decmath.SnapToTickFloor(price, tick)
		snappedTwice := decmath.SnapToTickFloor(snappedOnce, tick)
		return math.Abs(snappedOnce-snappedTwice) < 1e-8
	}

	cfg := &quick.Config{MaxCount: 1000}
	require.NoError(t, quick.Check(idempotenceFloor, cfg), "Idempotence failed for SnapToTickFloor")

	// 2. Idempotence of SnapToTickCeil: f(f(x)) == f(x)
	idempotenceCeil := func(price float64, tick float64) bool {
		price = math.Abs(price)
		tick = math.Abs(tick)
		if tick <= 0.0001 || tick > 1000 || price > 1e6 || math.IsNaN(price) || math.IsInf(price, 0) {
			return true
		}
		tick = decmath.RoundToScale(tick, 4)
		if tick == 0 {
			return true
		}
		snappedOnce := decmath.SnapToTickCeil(price, tick)
		snappedTwice := decmath.SnapToTickCeil(snappedOnce, tick)
		return math.Abs(snappedOnce-snappedTwice) < 1e-8
	}
	require.NoError(t, quick.Check(idempotenceCeil, cfg), "Idempotence failed for SnapToTickCeil")

	// 3. Invariant: Floor(p) <= p <= Ceil(p)
	boundingInvariant := func(price float64, tick float64) bool {
		price = math.Abs(price)
		tick = math.Abs(tick)
		if tick <= 0.0001 || tick > 1000 || price > 1e6 || math.IsNaN(price) || math.IsInf(price, 0) {
			return true
		}
		tick = decmath.RoundToScale(tick, 4)
		if tick == 0 {
			return true
		}
		floor := decmath.SnapToTickFloor(price, tick)
		ceil := decmath.SnapToTickCeil(price, tick)
		return (floor <= price+1e-8) && (price-1e-8 <= ceil) && (floor <= ceil)
	}
	require.NoError(t, quick.Check(boundingInvariant, cfg), "Floor/Ceil bounding invariant failed")
}

// TestProperty_IdGenInvariants verifies algebraic invariants of ID generation.
func TestProperty_IdGenInvariants(t *testing.T) {
	t.Parallel()

	// Length invariant: len(NanoID(n)) == n for n in 1..128
	lengthInvariant := func(size uint8) bool {
		n := int(size%128) + 1
		id := idgen.NanoID(n)
		return len(id) == n
	}
	cfg := &quick.Config{MaxCount: 500}
	require.NoError(t, quick.Check(lengthInvariant, cfg), "NanoID length invariant failed")

	// Alphabet containment invariant
	alphabetInvariant := func(size uint8) bool {
		n := int(size%64) + 1
		customAlphabet := "0123456789ABCDEF"
		id := idgen.NanoIDWithAlphabet(n, customAlphabet)
		if len(id) != n {
			return false
		}
		for _, ch := range id {
			found := false
			for _, allowed := range customAlphabet {
				if ch == allowed {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
		return true
	}
	require.NoError(t, quick.Check(alphabetInvariant, cfg), "NanoID alphabet containment invariant failed")
}

// TestProperty_RingBufferInvariants verifies RingBuffer capacity and sequence invariants.
func TestProperty_RingBufferInvariants(t *testing.T) {
	t.Parallel()

	f := func(capUint uint8, ops []uint16) bool {
		capacity := int(capUint%50) + 1
		rb := ringbuffer.New[int](capacity)

		if rb.Cap() != capacity {
			return false
		}

		lastSeq := int64(0)
		for _, val := range ops {
			seq := rb.Push(int(val))
			if seq != lastSeq+1 {
				return false // Monotonic sequence invariant
			}
			lastSeq = seq

			// Invariant: Len <= Cap
			if rb.Len() > rb.Cap() {
				return false
			}

			// Invariant: len(GetAll()) == Len()
			all := rb.GetAll()
			if len(all) != rb.Len() {
				return false
			}

			// Invariant: Latest() matches last element
			latest, ok := rb.Latest()
			if !ok || latest != int(val) {
				return false
			}
		}
		return true
	}

	cfg := &quick.Config{
		MaxCount: 500,
		Values: func(args []reflect.Value, r *rand.Rand) {
			args[0] = reflect.ValueOf(uint8(r.Intn(50) + 1))
			opCount := r.Intn(100)
			ops := make([]uint16, opCount)
			for i := range ops {
				ops[i] = uint16(r.Intn(65535))
			}
			args[1] = reflect.ValueOf(ops)
		},
	}
	require.NoError(t, quick.Check(f, cfg), "RingBuffer invariants failed under property-based fuzzing")
}

// TestProperty_FailClosedSecurityBoundaries tests fail-closed panic behaviors.
func TestProperty_FailClosedSecurityBoundaries(t *testing.T) {
	t.Parallel()

	// NanoID non-positive lengths must return empty string, never panic or buffer underrun
	assert.Empty(t, idgen.NanoID(0))
	assert.Empty(t, idgen.NanoID(-1))
	assert.Empty(t, idgen.NanoID(-1000))

	// NanoID with empty alphabet must return empty string
	assert.Empty(t, idgen.NanoIDWithAlphabet(10, ""))

	// SnapToTick with non-positive tick must return original price safely without panic
	assert.Equal(t, 123.45, decmath.SnapToTickFloor(123.45, 0))
	assert.Equal(t, 123.45, decmath.SnapToTickFloor(123.45, -0.05))
	assert.Equal(t, 123.45, decmath.SnapToTickCeil(123.45, 0))
	assert.Equal(t, 123.45, decmath.SnapToTickCeil(123.45, -0.05))
}
