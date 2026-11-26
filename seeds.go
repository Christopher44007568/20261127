package main

import (
	"fmt"
	"math"
)

func ValidateSeeds(g *Grid, seeds []Seed) ([]Seed, error) {
	if len(seeds) == 0 {
		return nil, fmt.Errorf("at least one seed is required")
	}
	seen := make(map[int]bool, len(seeds))
	out := make([]Seed, 0, len(seeds))
	for i, seed := range seeds {
		if !g.Inside(seed.X, seed.Y) {
			return nil, fmt.Errorf("seed %d (%d,%d) outside grid", i, seed.X, seed.Y)
		}
		id := g.ID(seed.X, seed.Y)
		if g.Blocked[id] {
			return nil, fmt.Errorf("seed %d (%d,%d) lies in blocked hole", i, seed.X, seed.Y)
		}
		if seen[id] {
			return nil, fmt.Errorf("duplicate seed at (%d,%d)", seed.X, seed.Y)
		}
		seen[id] = true
		out = append(out, seed)
	}
	return out, nil
}

// FarthestSeeds chooses deterministic geodesic farthest-point samples. It
// naturally walks around holes and assigns seeds to disconnected open parts.
func FarthestSeeds(g *Grid, count int) ([]Seed, error) {
	return FarthestSeedsStartingAt(g, count, centerOpenCell(g))
}

func FarthestSeedsStartingAt(g *Grid, count int, first Seed) ([]Seed, error) {
	openCount := g.OpenCount()
	if openCount == 0 {
		return nil, fmt.Errorf("grid has no traversable cells")
	}
	if count <= 0 {
		count = 8
	}
	if count > openCount {
		count = openCount
	}
	if !g.Open(first.X, first.Y) {
		first = centerOpenCell(g)
	}
	seeds := []Seed{first}
	for len(seeds) < count {
		field, err := MultiSourceDistance(g, seeds, nil, DistanceOptions{
			EightNeighbors: true, VerticalScale: 1,
		})
		if err != nil {
			return nil, err
		}
		bestID := -1
		bestDistance := -1.0
		bestInfinite := false
		for id, distance := range field.Dist {
			if g.Blocked[id] {
				continue
			}
			infinite := math.IsInf(distance, 1)
			if infinite && !bestInfinite {
				bestInfinite = true
				bestID = id
				continue
			}
			if bestInfinite && !infinite {
				continue
			}
			if distance > bestDistance || (infinite && bestDistance == -1) {
				bestDistance = distance
				bestID = id
			}
		}
		if bestID < 0 {
			break
		}
		seeds = append(seeds, Seed{X: bestID % g.W, Y: bestID / g.W})
	}
	return seeds, nil
}

func centerOpenCell(g *Grid) Seed {
	cx := g.W / 2
	cy := g.H / 2
	best := 0
	bestDistance := math.MaxInt
	found := false
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			if !g.Open(x, y) {
				continue
			}
			d := absInt(x-cx) + absInt(y-cy)
			if !found || d < bestDistance || (d == bestDistance && y*g.W+x < best) {
				found = true
				bestDistance = d
				best = y*g.W + x
			}
		}
	}
	return Seed{X: best % g.W, Y: best / g.W}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

// DomainComponents labels all traversable connected components using
// 4-connectivity.
func DomainComponents(g *Grid) ([]int, int) {
	comp := make([]int, g.W*g.H)
	for i := range comp {
		comp[i] = -1
	}
	count := 0
	queue := make([]int, 0, 128)
	for start := 0; start < g.W*g.H; start++ {
		if !g.OpenID(start) || comp[start] != -1 {
			continue
		}
		comp[start] = count
		queue = append(queue[:0], start)
		for len(queue) > 0 {
			p := queue[0]
			queue = queue[1:]
			px, py := p%g.W, p/g.W
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := px+d[0], py+d[1]
				if !g.Open(nx, ny) {
					continue
				}
				q := g.ID(nx, ny)
				if comp[q] == -1 {
					comp[q] = count
					queue = append(queue, q)
				}
			}
		}
		count++
	}
	return comp, count
}

// EnsureSeedCoverage appends a centroid-near seed to each disconnected open
// component that has no user seed.
func EnsureSeedCoverage(g *Grid, seeds []Seed) ([]Seed, int, error) {
	seeds, err := ValidateSeeds(g, seeds)
	if err != nil {
		return nil, 0, err
	}
	domain, componentCount := DomainComponents(g)
	hasSeed := make([]bool, componentCount)
	for _, seed := range seeds {
		hasSeed[domain[g.ID(seed.X, seed.Y)]] = true
	}

	added := 0
	for c := 0; c < componentCount; c++ {
		if hasSeed[c] {
			continue
		}
		sumX, sumY, cells := 0, 0, 0
		for id, dc := range domain {
			if dc != c {
				continue
			}
			sumX += id % g.W
			sumY += id / g.W
			cells++
		}
		if cells == 0 {
			continue
		}
		cx := float64(sumX) / float64(cells)
		cy := float64(sumY) / float64(cells)
		bestID := -1
		bestD := math.Inf(1)
		for id, dc := range domain {
			if dc != c {
				continue
			}
			x := float64(id % g.W)
			y := float64(id / g.W)
			d := (x-cx)*(x-cx) + (y-cy)*(y-cy)
			if d < bestD {
				bestD = d
				bestID = id
			}
		}
		seeds = append(seeds, Seed{X: bestID % g.W, Y: bestID / g.W})
		added++
	}
	return seeds, added, nil
}
