package main

import (
	"fmt"
	"math"
)

// Grid is a regular elevation grid. Blocked marks holes/no-data cells.
// Cell centers use integer coordinates; CellSize is the physical spacing.
type Grid struct {
	W         int
	H         int
	CellSize  float64
	Elevation []float64
	Blocked   []bool
}

func NewGrid(w, h int) (*Grid, error) {
	if w <= 2 || h <= 2 {
		return nil, fmt.Errorf("grid dimensions must be greater than 2, got %dx%d", w, h)
	}
	n := w * h
	return &Grid{
		W:         w,
		H:         h,
		CellSize:  1,
		Elevation: make([]float64, n),
		Blocked:   make([]bool, n),
	}, nil
}

func (g *Grid) ID(x, y int) int { return y*g.W + x }

func (g *Grid) Inside(x, y int) bool {
	return x >= 0 && x < g.W && y >= 0 && y < g.H
}

func (g *Grid) Open(x, y int) bool {
	return g.Inside(x, y) && !g.Blocked[g.ID(x, y)]
}

func (g *Grid) OpenID(id int) bool {
	return id >= 0 && id < len(g.Blocked) && !g.Blocked[id]
}

func (g *Grid) ClampedElevation(x, y int) float64 {
	if x < 0 {
		x = 0
	} else if x >= g.W {
		x = g.W - 1
	}
	if y < 0 {
		y = 0
	} else if y >= g.H {
		y = g.H - 1
	}
	return g.Elevation[g.ID(x, y)]
}

// CanDiagonal reports whether a diagonal step may be used. Requiring both
// orthogonal side cells prevents paths from squeezing through a corner where
// two blocked cells meet diagonally.
func (g *Grid) CanDiagonal(x, y, dx, dy int) bool {
	if dx == 0 || dy == 0 {
		return true
	}
	return g.Open(x+dx, y) && g.Open(x, y+dy)
}

func (g *Grid) Clone() *Grid {
	elev := append([]float64(nil), g.Elevation...)
	blocked := append([]bool(nil), g.Blocked...)
	return &Grid{W: g.W, H: g.H, CellSize: g.CellSize, Elevation: elev, Blocked: blocked}
}

func (g *Grid) OpenCount() int {
	count := 0
	for _, blocked := range g.Blocked {
		if !blocked {
			count++
		}
	}
	return count
}

func finiteMax(values []float64) float64 {
	max := 0.0
	for _, v := range values {
		if !math.IsInf(v, 1) && !math.IsNaN(v) && v > max {
			max = v
		}
	}
	return max
}
