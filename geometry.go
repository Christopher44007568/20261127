package bezierstroke

import "math"

// Point is a position in the same coordinate space as the curve.
type Point struct {
	X float64
	Y float64
}

func (p Point) Add(q Point) Point   { return Point{p.X + q.X, p.Y + q.Y} }
func (p Point) Sub(q Point) Point   { return Point{p.X - q.X, p.Y - q.Y} }
func (p Point) Mul(s float64) Point { return Point{p.X * s, p.Y * s} }

func (p Point) Dot(q Point) float64 {
	return p.X*q.X + p.Y*q.Y
}

func (p Point) Cross(q Point) float64 {
	return p.X*q.Y - p.Y*q.X
}

func (p Point) Len() float64 {
	return math.Hypot(p.X, p.Y)
}

func (p Point) Distance(q Point) float64 {
	return p.Sub(q).Len()
}

func (p Point) Unit() Point {
	n := p.Len()
	if n <= 1e-15 {
		return Point{}
	}
	return p.Mul(1 / n)
}

func (p Point) Perp() Point {
	return Point{-p.Y, p.X}
}

func lerpPoint(a, b Point, t float64) Point {
	return Point{a.X + (b.X-a.X)*t, a.Y + (b.Y-a.Y)*t}
}

// Cubic is a cubic Bezier segment in control-point form.
type Cubic struct {
	P0 Point
	P1 Point
	P2 Point
	P3 Point
}

// Eval returns B(t) for t in [0, 1].
func (c Cubic) Eval(t float64) Point {
	u := 1 - t
	a := u * u * u
	b := 3 * u * u * t
	d := 3 * u * t * t
	e := t * t * t
	return Point{
		X: a*c.P0.X + b*c.P1.X + d*c.P2.X + e*c.P3.X,
		Y: a*c.P0.Y + b*c.P1.Y + d*c.P2.Y + e*c.P3.Y,
	}
}

// Derivative returns B'(t).
func (c Cubic) Derivative(t float64) Point {
	u := 1 - t
	a := 3 * u * u
	b := 6 * u * t
	d := 3 * t * t
	return Point{
		X: a*(c.P1.X-c.P0.X) + b*(c.P2.X-c.P1.X) + d*(c.P3.X-c.P2.X),
		Y: a*(c.P1.Y-c.P0.Y) + b*(c.P2.Y-c.P1.Y) + d*(c.P3.Y-c.P2.Y),
	}
}

// Segment returns the exact control polygon for the parameter interval
// [t0, t1]. Reconstructing from endpoint derivatives avoids parameter drift
// that can occur when repeatedly applying de Casteljau splits.
func (c Cubic) Segment(t0, t1 float64) Cubic {
	h := t1 - t0
	d0 := c.Derivative(t0).Mul(h / 3)
	d1 := c.Derivative(t1).Mul(h / 3)
	a := c.Eval(t0)
	d := c.Eval(t1)
	return Cubic{P0: a, P1: a.Add(d0), P2: d.Sub(d1), P3: d}
}

// Split divides the curve at t using de Casteljau's algorithm.
func (c Cubic) Split(t float64) (Cubic, Cubic) {
	p01 := lerpPoint(c.P0, c.P1, t)
	p12 := lerpPoint(c.P1, c.P2, t)
	p23 := lerpPoint(c.P2, c.P3, t)
	p012 := lerpPoint(p01, p12, t)
	p123 := lerpPoint(p12, p23, t)
	p0123 := lerpPoint(p012, p123, t)
	return Cubic{c.P0, p01, p012, p0123}, Cubic{p0123, p123, p23, c.P3}
}

// TransformCubic maps every control point through f.
func TransformCubic(c Cubic, f func(Point) Point) Cubic {
	return Cubic{f(c.P0), f(c.P1), f(c.P2), f(c.P3)}
}
