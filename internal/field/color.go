package field

import (
	"math"

	"glyphsdf/internal/geom"
)

type Mask uint8

const (
	ChR Mask = 1 << iota
	ChG
	ChB
	MaskRG Mask = ChR | ChG
	MaskRB Mask = ChR | ChB
	MaskGB Mask = ChG | ChB
)

var secondaryMasks = [3]Mask{MaskRG, MaskRB, MaskGB}

// ColorContour assigns one of the three two-channel masks to every segment.
// Smooth joins keep the same mask; sharp joins switch to a different mask,
// leaving exactly one common channel at the corner.
func ColorContour(points []geom.V, cornerAngle float64) ([]Mask, []bool) {
	n := len(points)
	sharp := make([]bool, n)
	if n < 3 {
		return nil, sharp
	}
	for i := 0; i < n; i++ {
		prev := points[(i-1+n)%n]
		cur := points[i]
		next := points[(i+1)%n]
		in := cur.Sub(prev)
		out := next.Sub(cur)
		if in.Len() < 1e-9 || out.Len() < 1e-9 {
			sharp[i] = false
			continue
		}
		dot := in.Norm().Dot(out.Norm())
		dot = math.Max(-1, math.Min(1, dot))
		angle := math.Acos(dot)
		sharp[i] = angle > cornerAngle
	}

	colors := make([]int, n)
	for i := range colors {
		colors[i] = -1
	}
	var solve func(i int) bool
	solve = func(i int) bool {
		if i == n {
			if sharp[0] {
				return colors[n-1] != colors[0]
			}
			return colors[n-1] == colors[0]
		}
		if i == 0 {
			colors[0] = 0
			if solve(1) {
				return true
			}
			// Fixing the first color is only a symmetry reduction; if a
			// solver ever needs another starting color, try all three.
			for c := 1; c < 3; c++ {
				colors[0] = c
				if solve(1) {
					return true
				}
			}
			return false
		}
		mustDiffer := sharp[i]
		for c := 0; c < 3; c++ {
			if mustDiffer && c == colors[i-1] {
				continue
			}
			if !mustDiffer && c != colors[i-1] {
				continue
			}
			colors[i] = c
			if solve(i + 1) {
				return true
			}
			colors[i] = -1
		}
		return false
	}
	if !solve(0) {
		for i := range colors {
			colors[i] = 0
		}
	}
	masks := make([]Mask, n)
	for i, c := range colors {
		masks[i] = secondaryMasks[c%3]
	}
	return masks, sharp
}

func OrientShape(shape geom.Shape) geom.Shape {
	out := geom.Shape{Contours: make([]geom.Contour, len(shape.Contours))}
	for ci, c := range shape.Contours {
		points := c.Points
		n := len(points)
		if n < 3 {
			out.Contours[ci] = c
			continue
		}
		a := points[0]
		b := points[1]
		d := b.Sub(a)
		l := d.Len()
		if l < 1e-9 {
			out.Contours[ci] = c
			continue
		}
		normal := geom.V{X: -d.Y / l, Y: d.X / l}
		mid := a.Add(b).Mul(0.5)
		plus := geom.Inside(mid.Add(normal.Mul(0.08)), shape.Contours)
		minus := geom.Inside(mid.Sub(normal.Mul(0.08)), shape.Contours)
		if minus && !plus {
			reversed := make([]geom.V, n)
			for i := range points {
				reversed[i] = points[n-1-i]
			}
			points = reversed
		}
		out.Contours[ci] = geom.Contour{Points: points}
	}
	return out
}
