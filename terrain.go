package main

import "math"

// NewSyntheticGrid creates a deterministic height field with winding
// depressions, a high-curvature rim, and several interior holes.
func NewSyntheticGrid(w, h int) *Grid {
	if w <= 0 {
		w = 192
	}
	if h <= 0 {
		h = 144
	}
	g, err := NewGrid(w, h)
	if err != nil {
		panic(err)
	}
	g.CellSize = 1

	for y := 0; y < h; y++ {
		ny := (float64(y) - float64(h-1)/2) / math.Min(float64(w), float64(h))
		for x := 0; x < w; x++ {
			nx := (float64(x) - float64(w-1)/2) / math.Min(float64(w), float64(h))
			z := syntheticElevation(nx, ny)
			id := g.ID(x, y)
			g.Elevation[id] = z
			g.Blocked[id] = syntheticHole(nx, ny)
		}
	}
	return g
}

func syntheticElevation(x, y float64) float64 {
	z := 0.55*math.Sin(3.0*x+0.5) + 0.42*math.Cos(3.6*y-0.4)

	z += 2.1 * gaussian2(x, y, -0.27, -0.18, 0.20)
	z += 1.7 * gaussian2(x, y, 0.29, 0.22, 0.17)
	z -= 1.8 * gaussian2(x, y, 0.08, -0.28, 0.15)

	valleyY := -0.03 + 0.17*math.Sin(5.0*x+0.7)
	valleyD := y - valleyY
	z -= 1.25 * math.Exp(-(valleyD*valleyD)/(2*0.032*0.032))

	branchX := 0.10 + 0.10*math.Sin(5.5*y-0.8)
	branchD := x - branchX
	z -= 0.72 * math.Exp(-(branchD*branchD)/(2*0.026*0.026))

	r := math.Hypot(x+0.31, y-0.27)
	z += 1.05 * math.Exp(-((r-0.155)*(r-0.155))/(2*0.022*0.022))
	z += 0.10 * math.Sin(17*x+1.3) * math.Cos(15*y-0.6)
	return z
}

func gaussian2(x, y, cx, cy, sigma float64) float64 {
	dx := x - cx
	dy := y - cy
	return math.Exp(-(dx*dx + dy*dy) / (2 * sigma * sigma))
}

func syntheticHole(x, y float64) bool {
	if ellipseHole(x, y, 0.23, -0.03, 0.070, 0.070, 0) {
		return true
	}
	if ellipseHole(x, y, -0.27, 0.23, 0.105, 0.052, -0.45) {
		return true
	}
	if ellipseHole(x, y, 0.03, 0.34, 0.043, 0.043, 0) {
		return true
	}
	// A narrow closed slot makes the detour behavior visually obvious.
	return ellipseHole(x, y, -0.05, -0.35, 0.12, 0.026, 0.12)
}

func ellipseHole(x, y, cx, cy, rx, ry, angle float64) bool {
	dx := x - cx
	dy := y - cy
	c := math.Cos(angle)
	s := math.Sin(angle)
	ux := c*dx + s*dy
	uy := -s*dx + c*dy
	return (ux*ux)/(rx*rx)+(uy*uy)/(ry*ry) <= 1
}

// AddRectangularHole is a small helper for tests and custom terrains.
func (g *Grid) AddRectangularHole(x0, y0, x1, y1 int) {
	if x0 > x1 {
		x0, x1 = x1, x0
	}
	if y0 > y1 {
		y0, y1 = y1, y0
	}
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			if g.Inside(x, y) {
				g.Blocked[g.ID(x, y)] = true
			}
		}
	}
}
