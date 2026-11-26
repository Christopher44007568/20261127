package main

import (
	"image"
	"image/color"
	"math"
	"sort"
)

// forEachSpan rasterizes flattened contours with even-odd fill rule.
// Contour coordinates are already in target pixel space. For every covered
// row it calls emit(y, xStart, xEndInclusive).
func forEachSpan(w, h int, contours []Contour, emit func(y, x0, x1 int)) {
	type edge struct {
		y0, y1 float64
		xAt    func(y float64) float64
	}
	var edges []edge
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, c := range contours {
		pts := c.Pts
		if len(pts) < 2 {
			continue
		}
		n := len(pts)
		last := n
		if c.Closed {
			// Ensure closure edge exists.
			if pts[0] == pts[n-1] {
				last = n - 1
			}
		}
		for i := 0; i < last; i++ {
			a := pts[i]
			b := pts[(i+1)%n]
			if a.Y == b.Y {
				continue
			}
			y0, y1 := a.Y, b.Y
			if y0 > y1 {
				y0, y1 = y1, y0
			}
			dy := b.Y - a.Y
			dx := b.X - a.X
			aa, bb := a, b
			e := edge{y0: minF(aa.Y, bb.Y), y1: maxF(aa.Y, bb.Y)}
			e.xAt = func(y float64) float64 {
				return aa.X + (y-aa.Y)*dx/dy
			}
			edges = append(edges, e)
			if e.y0 < minY {
				minY = e.y0
			}
			if e.y1 > maxY {
				maxY = e.y1
			}
		}
	}
	if len(edges) == 0 {
		return
	}
	yStart := maxI(0, int(math.Floor(minY)))
	yEnd := minI(h-1, int(math.Ceil(maxY)))
	xs := make([]float64, 0, len(edges))
	for y := yStart; y <= yEnd; y++ {
		yc := float64(y) + 0.5
		xs = xs[:0]
		for _, e := range edges {
			// Top-left fill convention: lower endpoint excluded to avoid
			// double counting at shared vertices.
			if e.y0 <= yc && yc < e.y1 {
				xs = append(xs, e.xAt(yc))
			}
		}
		if len(xs) < 2 {
			continue
		}
		sort.Float64s(xs)
		for k := 0; k+1 < len(xs); k += 2 {
			xa, xb := xs[k], xs[k+1]
			x0 := int(math.Ceil(xa - 0.5 - 1e-7))
			x1 := int(math.Floor(xb - 0.5 + 1e-7))
			if x0 < 0 {
				x0 = 0
			}
			if x1 >= w {
				x1 = w - 1
			}
			if x0 <= x1 {
				emit(y, x0, x1)
			}
		}
	}
}

func minF(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
func maxF(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
func minI(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxI(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// FillGray renders contours as white coverage on an 8-bit mask.
func FillGray(m *image.Gray, contours []Contour, v uint8) {
	b := m.Bounds()
	forEachSpan(b.Dx(), b.Dy(), contours, func(y, x0, x1 int) {
		row := m.Pix[y*m.Stride : y*m.Stride+b.Dx()]
		for x := x0; x <= x1; x++ {
			row[x] = v
		}
	})
}

// StampPathGray paints a round-capped, round-joined thick polyline.
func StampPathGray(m *image.Gray, pts []Pt, radius float64, v uint8) {
	if len(pts) == 0 || radius <= 0 {
		return
	}
	for i := 0; i < len(pts)-1; i++ {
		stampSegmentGray(m, pts[i], pts[i+1], radius, v)
	}
	for _, p := range pts {
		stampDiscGray(m, p, radius, v)
	}
}

func stampSegmentGray(m *image.Gray, a, b Pt, r float64, v uint8) {
	d := a.dist(b)
	steps := int(math.Ceil(d / maxF(r*0.5, 0.5)))
	if steps < 1 {
		steps = 1
	}
	for k := 0; k <= steps; k++ {
		t := float64(k) / float64(steps)
		stampDiscGray(m, Pt{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t}, r, v)
	}
}

func stampDiscGray(m *image.Gray, c Pt, r float64, v uint8) {
	b := m.Bounds()
	x0 := maxI(0, int(math.Floor(c.X-r)))
	x1 := minI(b.Dx()-1, int(math.Ceil(c.X+r)))
	y0 := maxI(0, int(math.Floor(c.Y-r)))
	y1 := minI(b.Dy()-1, int(math.Ceil(c.Y+r)))
	r2 := r * r
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			dx := float64(x) + 0.5 - c.X
			dy := float64(y) + 0.5 - c.Y
			if dx*dx+dy*dy <= r2 {
				idx := y*m.Stride + x
				if v > m.Pix[idx] {
					m.Pix[idx] = v
				}
			}
		}
	}
}

var _ = color.RGBA{}
var _ image.Image
