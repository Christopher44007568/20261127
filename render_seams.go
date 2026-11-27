package main

import (
	"image/color"
	"math"
)

func drawModelSeams(c *Canvas, m *Mesh, atlas *Atlas, cam Camera) {
	tp := BuildTopology(m)
	for eid, isSeam := range atlas.Seam {
		if !isSeam || eid >= len(tp.Edge) {
			continue
		}
		e := tp.Edge[eid]
		x0, y0, z0, ok0 := cam.project(m.Pos[e.A], c.W, c.H)
		x1, y1, z1, ok1 := cam.project(m.Pos[e.B], c.W, c.H)
		if !ok0 || !ok1 {
			continue
		}
		drawDepthLine(c, x0, y0, z0, x1, y1, z1, 5, rgba(15, 23, 42, 220), 0.006)
		drawDepthLine(c, x0, y0, z0, x1, y1, z1, 2, rgba(251, 191, 36), 0.018)
	}
}

func drawDepthLine(c *Canvas, x0, y0, z0, x1, y1, z1 float64, thick int, col color.RGBA, bias float64) {
	steps := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))) + 1
	r := thick / 2
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(lerpFloat(x0, x1, t))
		y := int(lerpFloat(y0, y1, t))
		z := lerpFloat(z0, z1, t) - bias
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if dx*dx+dy*dy > r*r+r {
					continue
				}
				xx, yy := x+dx, y+dy
				if !c.In(xx, yy) {
					continue
				}
				idx := yy*c.W + xx
				if z <= c.Depth[idx]+0.018 {
					c.Set(xx, yy, col)
				}
			}
		}
	}
}
