package quadtree

import (
	"math"
	"sync"
)

// Point represents a 2D spatial coordinate.
type Point struct {
	X float64 `json:"x"`
	Z float64 `json:"z"`
}

// Rect represents an Axis-Aligned Bounding Box (AABB) in 2D space.
type Rect struct {
	MinX float64 `json:"min_x"`
	MinZ float64 `json:"min_z"`
	MaxX float64 `json:"max_x"`
	MaxZ float64 `json:"max_z"`
}

// NewRect creates a normalized Rect ensuring Min <= Max.
func NewRect(minX, minZ, maxX, maxZ float64) Rect {
	if minX > maxX {
		minX, maxX = maxX, minX
	}
	if minZ > maxZ {
		minZ, maxZ = maxZ, minZ
	}
	return Rect{
		MinX: minX,
		MinZ: minZ,
		MaxX: maxX,
		MaxZ: maxZ,
	}
}

// Contains reports whether the point (x, z) lies inside or on the boundaries of Rect.
func (r Rect) Contains(x, z float64) bool {
	return x >= r.MinX && x <= r.MaxX && z >= r.MinZ && z <= r.MaxZ
}

// Intersects reports whether two Rects overlap.
func (r Rect) Intersects(other Rect) bool {
	return !(r.MaxX < other.MinX || r.MinX > other.MaxX || r.MaxZ < other.MinZ || r.MinZ > other.MaxZ)
}

// Entry wraps a spatial item with its 2D coordinates.
type Entry[T any] struct {
	Point Point `json:"point"`
	Value T     `json:"value"`
}

// Quadtree represents a generic thread-safe 2D Quadtree spatial index.
type Quadtree[T any] struct {
	mu       sync.RWMutex
	bounds   Rect
	capacity int
	maxDepth int
	depth    int
	entries  []Entry[T]
	divided  bool
	nw       *Quadtree[T]
	ne       *Quadtree[T]
	sw       *Quadtree[T]
	se       *Quadtree[T]
}

// New creates a new generic Quadtree root node.
func New[T any](bounds Rect, capacity int, maxDepth int) *Quadtree[T] {
	if capacity <= 0 {
		capacity = 8
	}
	if maxDepth <= 0 {
		maxDepth = 8
	}
	return &Quadtree[T]{
		bounds:   bounds,
		capacity: capacity,
		maxDepth: maxDepth,
		depth:    0,
		entries:  make([]Entry[T], 0, capacity),
	}
}

func newChild[T any](bounds Rect, capacity int, maxDepth int, depth int) *Quadtree[T] {
	return &Quadtree[T]{
		bounds:   bounds,
		capacity: capacity,
		maxDepth: maxDepth,
		depth:    depth,
		entries:  make([]Entry[T], 0, capacity),
	}
}

// Bounds returns the bounding rectangle of the Quadtree.
func (q *Quadtree[T]) Bounds() Rect {
	q.mu.RLock()
	defer q.mu.RUnlock()
	return q.bounds
}

// subdivide splits the current node into 4 quadrant children (NW, NE, SW, SE).
func (q *Quadtree[T]) subdivide() {
	midX := (q.bounds.MinX + q.bounds.MaxX) / 2.0
	midZ := (q.bounds.MinZ + q.bounds.MaxZ) / 2.0
	nextDepth := q.depth + 1

	q.nw = newChild[T](NewRect(q.bounds.MinX, midZ, midX, q.bounds.MaxZ), q.capacity, q.maxDepth, nextDepth)
	q.ne = newChild[T](NewRect(midX, midZ, q.bounds.MaxX, q.bounds.MaxZ), q.capacity, q.maxDepth, nextDepth)
	q.sw = newChild[T](NewRect(q.bounds.MinX, q.bounds.MinZ, midX, midZ), q.capacity, q.maxDepth, nextDepth)
	q.se = newChild[T](NewRect(midX, q.bounds.MinZ, q.bounds.MaxX, midZ), q.capacity, q.maxDepth, nextDepth)
	q.divided = true

	// Re-insert existing entries into children
	oldEntries := q.entries
	q.entries = nil
	for _, entry := range oldEntries {
		q.insertChildLocked(entry)
	}
}

func (q *Quadtree[T]) insertChildLocked(entry Entry[T]) bool {
	if q.nw.insertInternal(entry) {
		return true
	}
	if q.ne.insertInternal(entry) {
		return true
	}
	if q.sw.insertInternal(entry) {
		return true
	}
	if q.se.insertInternal(entry) {
		return true
	}
	return false
}

func (q *Quadtree[T]) insertInternal(entry Entry[T]) bool {
	if !q.bounds.Contains(entry.Point.X, entry.Point.Z) {
		return false
	}

	if q.divided {
		return q.insertChildLocked(entry)
	}

	if len(q.entries) < q.capacity || q.depth >= q.maxDepth {
		q.entries = append(q.entries, entry)
		return true
	}

	q.subdivide()
	return q.insertChildLocked(entry)
}

// Insert inserts a new spatial item into the Quadtree.
// Returns false if the point lies outside the root bounds.
func (q *Quadtree[T]) Insert(x, z float64, value T) bool {
	q.mu.Lock()
	defer q.mu.Unlock()

	entry := Entry[T]{
		Point: Point{X: x, Z: z},
		Value: value,
	}
	return q.insertInternal(entry)
}

// QueryRange finds all entries whose coordinates lie within the specified range bounds.
func (q *Quadtree[T]) QueryRange(rangeBounds Rect) []Entry[T] {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var results []Entry[T]
	q.queryRangeInternal(rangeBounds, &results)
	return results
}

func (q *Quadtree[T]) queryRangeInternal(rangeBounds Rect, results *[]Entry[T]) {
	if !q.bounds.Intersects(rangeBounds) {
		return
	}

	if !q.divided {
		for _, e := range q.entries {
			if rangeBounds.Contains(e.Point.X, e.Point.Z) {
				*results = append(*results, e)
			}
		}
		return
	}

	q.nw.queryRangeInternal(rangeBounds, results)
	q.ne.queryRangeInternal(rangeBounds, results)
	q.sw.queryRangeInternal(rangeBounds, results)
	q.se.queryRangeInternal(rangeBounds, results)
}

// UpdateRange finds all entries within rangeBounds matching an optional filter, and modifies them via updateFn.
// If updateFn returns keep=false, the entry is removed from the Quadtree. Returns the number of modified entries.
func (q *Quadtree[T]) UpdateRange(rangeBounds Rect, filter func(T) bool, updateFn func(oldVal T) (newVal T, keep bool)) int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return q.updateRangeInternal(rangeBounds, filter, updateFn)
}

func (q *Quadtree[T]) updateRangeInternal(rangeBounds Rect, filter func(T) bool, updateFn func(oldVal T) (newVal T, keep bool)) int {
	if !q.bounds.Intersects(rangeBounds) {
		return 0
	}

	if !q.divided {
		count := 0
		remaining := make([]Entry[T], 0, len(q.entries))
		for _, e := range q.entries {
			if rangeBounds.Contains(e.Point.X, e.Point.Z) && (filter == nil || filter(e.Value)) {
				newVal, keep := updateFn(e.Value)
				count++
				if keep {
					e.Value = newVal
					remaining = append(remaining, e)
				}
			} else {
				remaining = append(remaining, e)
			}
		}
		q.entries = remaining
		return count
	}

	return q.nw.updateRangeInternal(rangeBounds, filter, updateFn) +
		q.ne.updateRangeInternal(rangeBounds, filter, updateFn) +
		q.sw.updateRangeInternal(rangeBounds, filter, updateFn) +
		q.se.updateRangeInternal(rangeBounds, filter, updateFn)
}

// QueryNearest finds the closest entry to (x, z) matching an optional filter predicate.
// Returns the entry, distance, and true if found, or false if tree is empty / no match.
func (q *Quadtree[T]) QueryNearest(x, z float64, filter func(T) bool) (Entry[T], float64, bool) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	var bestEntry Entry[T]
	bestDistSq := math.MaxFloat64
	found := false

	q.queryNearestInternal(x, z, filter, &bestEntry, &bestDistSq, &found)
	if !found {
		return bestEntry, 0, false
	}
	return bestEntry, math.Sqrt(bestDistSq), true
}

func (q *Quadtree[T]) queryNearestInternal(
	targetX, targetZ float64,
	filter func(T) bool,
	bestEntry *Entry[T],
	bestDistSq *float64,
	found *bool,
) {
	// Prune if bounding box is farther than current best distance
	if *found && rectMinDistSq(q.bounds, targetX, targetZ) >= *bestDistSq {
		return
	}

	if !q.divided {
		for _, e := range q.entries {
			if filter != nil && !filter(e.Value) {
				continue
			}
			dx := e.Point.X - targetX
			dz := e.Point.Z - targetZ
			distSq := dx*dx + dz*dz
			if distSq < *bestDistSq {
				*bestDistSq = distSq
				*bestEntry = e
				*found = true
			}
		}
		return
	}

	// Order children by distance to target point to search closest branches first
	children := []*Quadtree[T]{q.nw, q.ne, q.sw, q.se}
	type childDist struct {
		child *Quadtree[T]
		dist  float64
	}
	cd := make([]childDist, 4)
	for i, c := range children {
		cd[i] = childDist{child: c, dist: rectMinDistSq(c.bounds, targetX, targetZ)}
	}

	// Insertion sort for 4 elements
	for i := 1; i < 4; i++ {
		j := i
		for j > 0 && cd[j].dist < cd[j-1].dist {
			cd[j], cd[j-1] = cd[j-1], cd[j]
			j--
		}
	}

	for _, c := range cd {
		c.child.queryNearestInternal(targetX, targetZ, filter, bestEntry, bestDistSq, found)
	}
}

func rectMinDistSq(r Rect, x, z float64) float64 {
	var dx float64
	if x < r.MinX {
		dx = r.MinX - x
	} else if x > r.MaxX {
		dx = x - r.MaxX
	}

	var dz float64
	if z < r.MinZ {
		dz = r.MinZ - z
	} else if z > r.MaxZ {
		dz = z - r.MaxZ
	}

	return dx*dx + dz*dz
}

// Count returns the total number of entries in the Quadtree.
func (q *Quadtree[T]) Count() int {
	q.mu.RLock()
	defer q.mu.RUnlock()

	return q.countInternal()
}

func (q *Quadtree[T]) countInternal() int {
	if !q.divided {
		return len(q.entries)
	}
	return q.nw.countInternal() + q.ne.countInternal() + q.sw.countInternal() + q.se.countInternal()
}

// Clear removes all entries and resets the Quadtree.
func (q *Quadtree[T]) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.entries = make([]Entry[T], 0, q.capacity)
	q.divided = false
	q.nw = nil
	q.ne = nil
	q.sw = nil
	q.se = nil
}

// ForEach visits all entries in the Quadtree.
func (q *Quadtree[T]) ForEach(fn func(entry Entry[T])) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	q.forEachInternal(fn)
}

func (q *Quadtree[T]) forEachInternal(fn func(entry Entry[T])) {
	if !q.divided {
		for _, e := range q.entries {
			fn(e)
		}
		return
	}
	q.nw.forEachInternal(fn)
	q.ne.forEachInternal(fn)
	q.sw.forEachInternal(fn)
	q.se.forEachInternal(fn)
}
