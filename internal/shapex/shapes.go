package shapex

import (
	"math"

	"glyphsdf/internal/geom"
)

func Fit(shape geom.Shape, w, h int, margin float64, flipY bool) geom.Shape {
	box := shape.Box()
	if box.Empty() {
		return shape
	}
	bw := math.Max(1e-6, box.Max.X-box.Min.X)
	bh := math.Max(1e-6, box.Max.Y-box.Min.Y)
	scale := math.Min((float64(w)-2*margin)/bw, (float64(h)-2*margin)/bh)
	cx := (box.Min.X + box.Max.X) * 0.5
	cy := (box.Min.Y + box.Max.Y) * 0.5
	out := geom.Shape{Contours: make([]geom.Contour, len(shape.Contours))}
	for i, c := range shape.Contours {
		nc := geom.Contour{Points: make([]geom.V, len(c.Points))}
		for j, p := range c.Points {
			x := float64(w)*0.5 + (p.X-cx)*scale
			y := float64(h)*0.5 + (p.Y-cy)*scale
			if flipY {
				y = float64(h)*0.5 - (p.Y-cy)*scale
			}
			nc.Points[j] = geom.V{X: x, Y: y}
		}
		out.Contours[i] = nc
	}
	return out
}

func polygon(points ...float64) geom.Contour {
	c := geom.Contour{}
	for i := 0; i < len(points); i += 2 {
		c.Points = append(c.Points, geom.V{X: points[i], Y: points[i+1]})
	}
	return c
}

// OverlapStar returns two equally wound overlapping triangles. Non-zero fill
// keeps their shared overlap solid; the crossing edges are internal seams and
// must not dominate the distance field.
func OverlapStar() geom.Shape {
	up := polygon(
		0, 100,
		-86.6, -50,
		86.6, -50,
	)
	down := polygon(
		0, -100,
		86.6, 50,
		-86.6, 50,
	)
	return geom.Shape{Contours: []geom.Contour{up, down}}
}

// CornerComb returns an extremely sharp concave/concave comb used to make
// corner-break detection unambiguous.
func CornerComb() geom.Shape {
	outer := polygon(
		-100, -80,
		100, -80,
		100, 10,
		82, 80,
		62, 10,
		42, 80,
		22, 10,
		2, 80,
		-18, 10,
		-38, 80,
		-58, 10,
		-78, 80,
		-100, 10,
	)
	return geom.Shape{Contours: []geom.Contour{outer}}
}

func BuiltinA() geom.Shape {
	outer := polygon(
		-60, -80,
		-25, -80,
		-17, -55,
		17, -55,
		25, -80,
		60, -80,
		15, 80,
		-15, 80,
	)
	counter := polygon(
		-13, -20,
		13, -20,
		0, 38,
	)
	return geom.Shape{Contours: []geom.Contour{outer, counter}}
}
