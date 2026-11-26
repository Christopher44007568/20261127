package geom

import "math"

// FlattenQuad appends a tolerance-flattened quadratic Bézier. The endpoint b
// is always appended; a is assumed to already be in out when appropriate.
func FlattenQuad(a, c, b V, tolerance float64, out *[]V) {
	if pointLineDistance(c, a, b) <= tolerance || tolerance <= 0 {
		*out = append(*out, b)
		return
	}
	ab := a.Add(c.Sub(a).Mul(0.5))
	cb := c.Add(b.Sub(c).Mul(0.5))
	mid := ab.Add(cb.Sub(ab).Mul(0.5))
	FlattenQuad(a, ab, mid, tolerance, out)
	FlattenQuad(mid, cb, b, tolerance, out)
}

// FlattenCubic appends a tolerance-flattened cubic Bézier.
func FlattenCubic(a, c1, c2, b V, tolerance float64, out *[]V) {
	// A conservative control-net flatness test is sufficient for this
	// offline generator and keeps subdivision nicely balanced.
	d1 := pointLineDistance(c1, a, b)
	d2 := pointLineDistance(c2, a, b)
	if math.Max(d1, d2) <= tolerance {
		*out = append(*out, b)
		return
	}
	p01 := a.Add(c1.Sub(a).Mul(0.5))
	p12 := c1.Add(c2.Sub(c1).Mul(0.5))
	p23 := c2.Add(b.Sub(c2).Mul(0.5))
	p012 := p01.Add(p12.Sub(p01).Mul(0.5))
	p123 := p12.Add(p23.Sub(p12).Mul(0.5))
	mid := p012.Add(p123.Sub(p012).Mul(0.5))
	FlattenCubic(a, p01, p012, mid, tolerance, out)
	FlattenCubic(mid, p123, p23, b, tolerance, out)
}

func pointLineDistance(p, a, b V) float64 {
	d := b.Sub(a)
	l := d.Len()
	if l < 1e-12 {
		return p.Dist(a)
	}
	return math.Abs(d.Cross(p.Sub(a))) / l
}

func TransformShape(s Shape, scale float64, offset V) Shape {
	out := Shape{Contours: make([]Contour, 0, len(s.Contours))}
	for _, c := range s.Contours {
		nc := Contour{Points: make([]V, len(c.Points))}
		for i, p := range c.Points {
			nc.Points[i] = V{X: p.X*scale + offset.X, Y: p.Y*scale + offset.Y}
		}
		out.Contours = append(out.Contours, nc)
	}
	return out
}

func MapShape(s Shape, sx, sy float64, offset V) Shape {
	out := Shape{Contours: make([]Contour, 0, len(s.Contours))}
	for _, c := range s.Contours {
		nc := Contour{Points: make([]V, len(c.Points))}
		for i, p := range c.Points {
			nc.Points[i] = V{X: p.X*sx + offset.X, Y: p.Y*sy + offset.Y}
		}
		out.Contours = append(out.Contours, nc)
	}
	return out
}

func (s Shape) Transformed(scale float64, offset V) Shape {
	return TransformShape(s, scale, offset)
}
