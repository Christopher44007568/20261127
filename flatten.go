package bezierstroke

import (
	"math"
)

const defaultMaxDepth = 48

// FlatVertex is a point on an adaptively flattened centerline. T is its
// original Bezier parameter and Depth records how many splits produced it.
type FlatVertex struct {
	Point
	T     float64
	Depth int
}

// FlatPolyline contains both the renderable points and their parameters.
type FlatPolyline struct {
	Points []Point
	Params []float64
}

// ChordHeight returns the maximum perpendicular distance from the cubic to
// its endpoint chord. The extrema of that distance are found analytically by
// solving d/dt cross(B(t)-P0, P3-P0) = 0, so the result is not a sampled
// estimate and does not depend on a fixed subdivision count.
func (c Cubic) ChordHeight() float64 {
	chord := c.P3.Sub(c.P0)
	chordLen := chord.Len()
	if chordLen <= 1e-12 {
		// A zero chord is degenerate for the line-distance formula. The
		// convex hull gives a conservative fallback.
		return math.Max(c.P1.Distance(c.P0),
			math.Max(c.P2.Distance(c.P0), c.P3.Distance(c.P0)))
	}

	v1 := c.P1.Sub(c.P0)
	v2 := c.P2.Sub(c.P1.Mul(2)).Add(c.P0)
	v3 := c.P3.Sub(c.P2.Mul(3)).Add(c.P1.Mul(3)).Sub(c.P0)

	a1 := 3 * v1.Cross(chord)
	a2 := 3 * v2.Cross(chord)
	a3 := v3.Cross(chord)

	roots := quadraticRoots(3*a3, 2*a2, a1)
	maxAbs := 0.0
	for _, t := range roots {
		if t > 0 && t < 1 {
			q := c.Eval(t).Sub(c.P0).Cross(chord)
			maxAbs = math.Max(maxAbs, math.Abs(q))
		}
	}
	return maxAbs / chordLen
}

// chordMonotonic reports whether the control polygon is ordered along its
// endpoint chord. A curve can have zero chord height and still fold back on
// itself when its control points lie on the same line, so flatness and
// monotonicity are checked together.
func (c Cubic) chordMonotonic() bool {
	chord := c.P3.Sub(c.P0)
	l2 := chord.Dot(chord)
	if l2 <= 1e-24 {
		return false
	}
	s1 := c.P1.Sub(c.P0).Dot(chord) / l2
	s2 := c.P2.Sub(c.P0).Dot(chord) / l2
	const eps = 1e-12
	return s1 >= -eps && s2 >= s1-eps && s2 <= 1+eps
}

func (c Cubic) flatEnough(tolerance float64) bool {
	if c.ChordHeight() > tolerance {
		return false
	}
	// A collapsed subcurve has no chord direction to test for monotonicity.
	// ChordHeight uses the convex hull in that case, so accepting it cannot
	// hide a shape larger than tolerance.
	if c.P3.Distance(c.P0) <= 1e-12 {
		return true
	}
	return c.chordMonotonic()
}

func quadraticRoots(a, b, c float64) []float64 {
	const eps = 1e-14
	if math.Abs(a) <= eps {
		if math.Abs(b) <= eps {
			return nil
		}
		return []float64{-c / b}
	}
	disc := b*b - 4*a*c
	if disc < 0 {
		return nil
	}
	if disc == 0 {
		return []float64{-b / (2 * a)}
	}
	rootDisc := math.Sqrt(disc)
	q := -0.5 * (b + math.Copysign(rootDisc, b))
	if q == 0 {
		return []float64{-b / (2 * a)}
	}
	return []float64{q / a, c / q}
}

// AdaptiveFlattenDetailed subdivides each interval until its analytic chord
// height is at most tolerance. Tolerance is in the same units as the curve.
// The returned parameters are ordered and span [0, 1].
func (c Cubic) AdaptiveFlattenDetailed(tolerance float64) []FlatVertex {
	if math.IsNaN(tolerance) || tolerance <= 0 {
		tolerance = 1e-9
	}

	type task struct {
		curve Cubic
		t0    float64
		t1    float64
		depth int
	}
	stack := []task{{curve: c, t0: 0, t1: 1}}
	out := make([]FlatVertex, 0, 8)

	for len(stack) > 0 {
		n := len(stack) - 1
		item := stack[n]
		stack = stack[:n]

		flat := item.curve.flatEnough(tolerance)
		if !flat && item.depth < defaultMaxDepth {
			tm := (item.t0 + item.t1) * 0.5
			left, right := item.curve.Split(0.5)
			// Push right first so the left interval is emitted first.
			stack = append(stack,
				task{right, tm, item.t1, item.depth + 1},
				task{left, item.t0, tm, item.depth + 1})
			continue
		}

		if len(out) == 0 || item.curve.P0.Distance(out[len(out)-1].Point) > 1e-12 {
			out = append(out, FlatVertex{item.curve.P0, item.t0, item.depth})
		}
	}

	end := FlatVertex{c.P3, 1, 0}
	if len(out) == 0 || out[len(out)-1].Point.Distance(end.Point) > 1e-12 {
		out = append(out, end)
	} else {
		out[len(out)-1].T = 1
	}
	return out
}

// AdaptiveFlatten returns only the centerline points.
func (c Cubic) AdaptiveFlatten(tolerance float64) []Point {
	details := c.AdaptiveFlattenDetailed(tolerance)
	points := make([]Point, len(details))
	for i := range details {
		points[i] = details[i].Point
	}
	return points
}

// AdaptiveFlattenPolyline returns centerline points and their parameters.
func (c Cubic) AdaptiveFlattenPolyline(tolerance float64) FlatPolyline {
	details := c.AdaptiveFlattenDetailed(tolerance)
	points := make([]Point, len(details))
	params := make([]float64, len(details))
	for i := range details {
		points[i] = details[i].Point
		params[i] = details[i].T
	}
	return FlatPolyline{Points: points, Params: params}
}
