package main

import (
	"container/heap"
	"fmt"
	"math"
)

type Seed struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type DistanceOptions struct {
	EightNeighbors  bool
	VerticalScale   float64
	FeatureDiscount float64
}

func DefaultDistanceOptions() DistanceOptions {
	return DistanceOptions{EightNeighbors: true, VerticalScale: 1.0, FeatureDiscount: 0}
}

type DistanceField struct {
	Dist   []float64
	Label  []int
	Parent []int
}

type heapItem struct {
	id   int
	dist float64
}

type distanceHeap []heapItem

func (h distanceHeap) Len() int            { return len(h) }
func (h distanceHeap) Less(i, j int) bool  { return h[i].dist < h[j].dist }
func (h distanceHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *distanceHeap) Push(x interface{}) { *h = append(*h, x.(heapItem)) }
func (h *distanceHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

// MultiSourceDistance runs seeded Dijkstra. With FourNeighbors, every assigned
// cell has a same-label predecessor and therefore forms a 4-connected tree.
func MultiSourceDistance(g *Grid, seeds []Seed, feature []float64, opts DistanceOptions) (*DistanceField, error) {
	if len(seeds) == 0 {
		return nil, fmt.Errorf("at least one seed is required")
	}
	seedIDs := make(map[int]int, len(seeds))
	for i, s := range seeds {
		if !g.Inside(s.X, s.Y) {
			return nil, fmt.Errorf("seed %d (%d,%d) is outside grid", i, s.X, s.Y)
		}
		id := g.ID(s.X, s.Y)
		if g.Blocked[id] {
			return nil, fmt.Errorf("seed %d (%d,%d) is in a hole", i, s.X, s.Y)
		}
		if prev, exists := seedIDs[id]; exists {
			return nil, fmt.Errorf("seed %d duplicates seed %d at (%d,%d)", i, prev, s.X, s.Y)
		}
		seedIDs[id] = i
	}
	if opts.VerticalScale <= 0 {
		opts.VerticalScale = 1
	}

	n := g.W * g.H
	out := &DistanceField{
		Dist:   make([]float64, n),
		Label:  make([]int, n),
		Parent: make([]int, n),
	}
	for i := range out.Dist {
		out.Dist[i] = math.Inf(1)
		out.Label[i] = -1
		out.Parent[i] = -1
	}

	h := &distanceHeap{}
	heap.Init(h)
	for i, s := range seeds {
		id := g.ID(s.X, s.Y)
		out.Dist[id] = 0
		out.Label[id] = i
		out.Parent[id] = id
		heap.Push(h, heapItem{id: id, dist: 0})
	}

	for h.Len() > 0 {
		item := heap.Pop(h).(heapItem)
		p := item.id
		if item.dist != out.Dist[p] {
			continue
		}
		px, py := p%g.W, p/g.W
		forEachOffset(opts.EightNeighbors, func(dx, dy int) bool {
			nx, ny := px+dx, py+dy
			if !g.Open(nx, ny) || (dx != 0 && dy != 0 && !g.CanDiagonal(px, py, dx, dy)) {
				return true
			}
			q := g.ID(nx, ny)
			cost := g.edgeCost(px, py, nx, ny, dx, dy, feature, opts)
			nd := out.Dist[p] + cost
			if nd < out.Dist[q] {
				out.Dist[q] = nd
				out.Label[q] = out.Label[p]
				out.Parent[q] = p
				heap.Push(h, heapItem{id: q, dist: nd})
			}
			return true
		})
	}
	return out, nil
}

func SingleSourceDistance(g *Grid, seed Seed, feature []float64, opts DistanceOptions) ([]float64, error) {
	f, err := MultiSourceDistance(g, []Seed{seed}, feature, opts)
	if err != nil {
		return nil, err
	}
	return f.Dist, nil
}

func (g *Grid) edgeCost(x, y, nx, ny, dx, dy int, feature []float64, opts DistanceOptions) float64 {
	cs := g.CellSize
	if cs <= 0 {
		cs = 1
	}
	horizontal := cs * math.Sqrt(float64(dx*dx+dy*dy))
	dz := (g.Elevation[g.ID(nx, ny)] - g.Elevation[g.ID(x, y)]) * opts.VerticalScale
	cost := math.Hypot(horizontal, dz)
	if cost < 1e-12 {
		cost = horizontal
	}
	if opts.FeatureDiscount > 0 && feature != nil {
		score := edgeFeature(g, feature, x, y, nx, ny)
		discount := opts.FeatureDiscount * score
		if discount > 0.75 {
			discount = 0.75
		}
		cost *= 1 - discount
	}
	return cost
}

func forEachOffset(eight bool, fn func(dx, dy int) bool) {
	if eight {
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				if dx == 0 && dy == 0 {
					continue
				}
				if !fn(dx, dy) {
					return
				}
			}
		}
		return
	}
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		if !fn(d[0], d[1]) {
			return
		}
	}
}
