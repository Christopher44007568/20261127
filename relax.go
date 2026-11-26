package main

import "math"

type SegmentOptions struct {
	SeedCount            int
	RelaxIterations      int
	GeoWeight            float64
	SmoothWeight         float64
	FeatureWeight        float64
	FeatureDiscount      float64
	VerticalScale        float64
	StrictSeeds          bool
	DisableTopologyCheck bool
	FourNeighborHeatmap  bool
}

func DefaultSegmentOptions() SegmentOptions {
	return SegmentOptions{
		SeedCount:       8,
		RelaxIterations: 30,
		GeoWeight:       0.22,
		SmoothWeight:    0.46,
		FeatureWeight:   1.05,
		FeatureDiscount: 0.16,
		VerticalScale:   1.25,
	}
}

func (o SegmentOptions) withDefaults() SegmentOptions {
	if o.SeedCount <= 0 {
		o.SeedCount = 8
	}
	if o.RelaxIterations < 0 {
		o.RelaxIterations = 0
	}
	if o.VerticalScale <= 0 {
		o.VerticalScale = 1
	}
	return o
}

// RelaxBoundaries performs deterministic iterated conditional modes. A flip is
// accepted only when it lowers geodesic + perimeter - feature energy and, by
// default, preserves 4-connectivity of both affected labels.
func RelaxBoundaries(g *Grid, labels []int, seeds []Seed, distances [][]float64, curvature *CurvatureField, opts SegmentOptions) (int, float64) {
	n := g.W * g.H
	isSeed := make([]bool, n)
	for _, seed := range seeds {
		isSeed[g.ID(seed.X, seed.Y)] = true
	}
	marks := make([]int, n)
	bfsQueue := make([]int, 0, 256)
	token := 0
	totalFlips := 0

	for iter := 0; iter < opts.RelaxIterations; iter++ {
		iterationFlips := 0
		forward := iter%2 == 0
		for yy := 0; yy < g.H; yy++ {
			y := yy
			if !forward {
				y = g.H - 1 - yy
			}
			for xx := 0; xx < g.W; xx++ {
				x := xx
				if !forward {
					x = g.W - 1 - xx
				}
				p := g.ID(x, y)
				if g.Blocked[p] || isSeed[p] {
					continue
				}
				current := labels[p]
				bestLabel := current
				bestDelta := 0.0
				candidates := make([]int, 0, 4)
				for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
					nx, ny := x+d[0], y+d[1]
					if !g.Open(nx, ny) {
						continue
					}
					b := labels[g.ID(nx, ny)]
					if b == current {
						continue
					}
					seen := false
					for _, c := range candidates {
						if c == b {
							seen = true
							break
						}
					}
					if !seen {
						candidates = append(candidates, b)
					}
				}

				for _, candidate := range candidates {
					if !opts.DisableTopologyCheck {
						token++
						if !flipIsTopologySafe(g, labels, p, candidate, marks, bfsQueue, token) {
							continue
						}
					}
					delta := flipEnergyDelta(g, labels, distances, curvature, opts, p, current, candidate)
					if delta < bestDelta-1e-10 {
						bestDelta = delta
						bestLabel = candidate
					}
				}

				if bestLabel != current {
					labels[p] = bestLabel
					iterationFlips++
					totalFlips++
				}
			}
		}
		if iterationFlips == 0 {
			break
		}
	}
	return totalFlips, LabelEnergy(g, labels, distances, curvature, opts)
}

func flipEnergyDelta(g *Grid, labels []int, distances [][]float64, curvature *CurvatureField, opts SegmentOptions, p, oldLabel, newLabel int) float64 {
	cs := g.CellSize
	if cs <= 0 {
		cs = 1
	}
	delta := opts.GeoWeight * (distances[newLabel][p] - distances[oldLabel][p])
	px, py := p%g.W, p/g.W
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nx, ny := px+d[0], py+d[1]
		if !g.Open(nx, ny) {
			continue
		}
		q := g.ID(nx, ny)
		neighborLabel := labels[q]
		before := 0.0
		after := 0.0
		if neighborLabel != oldLabel {
			before = 1
		}
		if neighborLabel != newLabel {
			after = 1
		}
		change := after - before
		delta += opts.SmoothWeight * cs * change
		score := orientedEdgeFeature(g, curvature, px, py, nx, ny)
		delta -= opts.FeatureWeight * cs * score * change
	}
	return delta
}

func LabelEnergy(g *Grid, labels []int, distances [][]float64, curvature *CurvatureField, opts SegmentOptions) float64 {
	cs := g.CellSize
	if cs <= 0 {
		cs = 1
	}
	energy := 0.0
	for id := 0; id < g.W*g.H; id++ {
		if g.Blocked[id] {
			continue
		}
		label := labels[id]
		if label >= 0 && label < len(distances) {
			energy += opts.GeoWeight * distances[label][id]
		}
	}
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			id := g.ID(x, y)
			if g.Blocked[id] {
				continue
			}
			if x+1 < g.W && !g.Blocked[id+1] && labels[id] != labels[id+1] {
				score := orientedEdgeFeature(g, curvature, x, y, x+1, y)
				energy += cs * (opts.SmoothWeight - opts.FeatureWeight*score)
			}
			if y+1 < g.H && !g.Blocked[id+g.W] && labels[id] != labels[id+g.W] {
				score := orientedEdgeFeature(g, curvature, x, y, x, y+1)
				energy += cs * (opts.SmoothWeight - opts.FeatureWeight*score)
			}
		}
	}
	return math.Round(energy*1e6) / 1e6
}
