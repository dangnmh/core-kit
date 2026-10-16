package quadtree_test

import (
	"fmt"
	"math"
	"sync"
	"testing"

	"github.com/dangnmh/core-kit/pkg/quadtree"
)

type ResourceInfo struct {
	Name  string
	Count int
}

func TestQuadtree_Basic(t *testing.T) {
	t.Parallel()

	bounds := quadtree.NewRect(-100, -100, 100, 100)
	qt := quadtree.New[ResourceInfo](bounds, 4, 5)

	if count := qt.Count(); count != 0 {
		t.Fatalf("expected count 0, got %d", count)
	}

	// Insert items
	if !qt.Insert(10, 10, ResourceInfo{Name: "iron_ore", Count: 5}) {
		t.Fatal("expected insert to succeed")
	}
	if !qt.Insert(-20, -30, ResourceInfo{Name: "coal_ore", Count: 10}) {
		t.Fatal("expected insert to succeed")
	}
	if !qt.Insert(50, -50, ResourceInfo{Name: "spruce_log", Count: 20}) {
		t.Fatal("expected insert to succeed")
	}

	// Out of bounds insertion
	if qt.Insert(200, 200, ResourceInfo{Name: "out_of_bounds", Count: 1}) {
		t.Fatal("expected out of bounds insert to fail")
	}

	if count := qt.Count(); count != 3 {
		t.Fatalf("expected count 3, got %d", count)
	}
}

func TestQuadtree_SubdivisionAndQueryRange(t *testing.T) {
	t.Parallel()

	bounds := quadtree.NewRect(0, 0, 100, 100)
	// Low capacity to trigger multiple subdivisions
	qt := quadtree.New[string](bounds, 2, 4)

	// Insert 10 points
	points := []struct {
		x, z float64
		name string
	}{
		{5, 5, "p1"},
		{10, 10, "p2"},
		{15, 15, "p3"},
		{20, 20, "p4"},
		{60, 60, "p5"},
		{70, 70, "p6"},
		{80, 80, "p7"},
		{90, 90, "p8"},
		{10, 80, "p9"},
		{80, 10, "p10"},
	}

	for _, p := range points {
		if !qt.Insert(p.x, p.z, p.name) {
			t.Fatalf("failed to insert point %+v", p)
		}
	}

	if count := qt.Count(); count != 10 {
		t.Fatalf("expected 10 items, got %d", count)
	}

	// Query box covering only [0..30, 0..30] -> should find p1, p2, p3, p4
	rangeBox := quadtree.NewRect(0, 0, 30, 30)
	results := qt.QueryRange(rangeBox)

	if len(results) != 4 {
		t.Fatalf("expected 4 results in [0..30, 0..30], got %d", len(results))
	}

	// Query box covering entire world -> should find all 10
	allResults := qt.QueryRange(bounds)
	if len(allResults) != 10 {
		t.Fatalf("expected 10 results in full world, got %d", len(allResults))
	}

	// Query empty area
	emptyResults := qt.QueryRange(quadtree.NewRect(35, 35, 45, 45))
	if len(emptyResults) != 0 {
		t.Fatalf("expected 0 results, got %d", len(emptyResults))
	}
}

func TestQuadtree_QueryNearest(t *testing.T) {
	t.Parallel()

	bounds := quadtree.NewRect(-1000, -1000, 1000, 1000)
	qt := quadtree.New[ResourceInfo](bounds, 4, 6)

	qt.Insert(100, 100, ResourceInfo{Name: "iron_ore", Count: 10})
	qt.Insert(10, 10, ResourceInfo{Name: "coal_ore", Count: 5})
	qt.Insert(-50, -50, ResourceInfo{Name: "iron_ore", Count: 20})
	qt.Insert(500, 500, ResourceInfo{Name: "diamond_ore", Count: 3})

	// Query nearest to (0, 0) without filter -> should be coal_ore at (10, 10)
	entry, dist, found := qt.QueryNearest(0, 0, nil)
	if !found {
		t.Fatal("expected to find nearest point")
	}
	expectedDist := math.Hypot(10, 10)
	if math.Abs(dist-expectedDist) > 0.001 || entry.Value.Name != "coal_ore" {
		t.Fatalf("expected coal_ore at dist %.2f, got %s at dist %.2f", expectedDist, entry.Value.Name, dist)
	}

	// Query nearest with filter for "iron_ore"
	entryIron, distIron, foundIron := qt.QueryNearest(0, 0, func(v ResourceInfo) bool {
		return v.Name == "iron_ore"
	})
	if !foundIron {
		t.Fatal("expected to find nearest iron_ore")
	}
	// (-50, -50) dist = sqrt(5000) ~= 70.71, (100, 100) dist = sqrt(20000) ~= 141.42
	expectedIronDist := math.Hypot(-50, -50)
	if math.Abs(distIron-expectedIronDist) > 0.001 || entryIron.Point.X != -50 {
		t.Fatalf("expected iron_ore at (-50,-50) dist %.2f, got (%.0f, %.0f) at dist %.2f",
			expectedIronDist, entryIron.Point.X, entryIron.Point.Z, distIron)
	}

	// Query nearest for non-existent item
	_, _, foundEmerald := qt.QueryNearest(0, 0, func(v ResourceInfo) bool {
		return v.Name == "emerald_ore"
	})
	if foundEmerald {
		t.Fatal("expected emerald_ore to not be found")
	}
}

func TestQuadtree_ForEachAndClear(t *testing.T) {
	t.Parallel()

	bounds := quadtree.NewRect(0, 0, 100, 100)
	qt := quadtree.New[int](bounds, 2, 4)

	for i := 0; i < 20; i++ {
		qt.Insert(float64(i*5), float64(i*5), i)
	}

	if count := qt.Count(); count != 20 {
		t.Fatalf("expected count 20, got %d", count)
	}

	visited := make(map[int]bool)
	qt.ForEach(func(e quadtree.Entry[int]) {
		visited[e.Value] = true
	})

	if len(visited) != 20 {
		t.Fatalf("expected 20 visited entries, got %d", len(visited))
	}

	qt.Clear()
	if count := qt.Count(); count != 0 {
		t.Fatalf("expected count 0 after clear, got %d", count)
	}
	if len(qt.QueryRange(bounds)) != 0 {
		t.Fatal("expected empty query range after clear")
	}
}

func TestQuadtree_Concurrent(t *testing.T) {
	t.Parallel()

	bounds := quadtree.NewRect(-500, -500, 500, 500)
	qt := quadtree.New[string](bounds, 4, 6)

	var wg sync.WaitGroup
	numWorkers := 8
	itemsPerWorker := 100

	// Concurrent Inserts
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < itemsPerWorker; i++ {
				x := float64((workerID*50 + i) % 400)
				z := float64((workerID*40 + i*2) % 400)
				qt.Insert(x, z, fmt.Sprintf("w%d_item%d", workerID, i))
			}
		}(w)
	}

	// Concurrent Reads
	for w := 0; w < numWorkers; w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				qt.QueryRange(quadtree.NewRect(-100, -100, 200, 200))
				qt.QueryNearest(0, 0, nil)
				qt.Count()
			}
		}()
	}

	wg.Wait()

	if total := qt.Count(); total != numWorkers*itemsPerWorker {
		t.Fatalf("expected total count %d, got %d", numWorkers*itemsPerWorker, total)
	}
}
