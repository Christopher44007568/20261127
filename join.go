package stroke

import "math"

type sideSet struct {
	leftIn   Point
	rightIn  Point
	leftOut  Point
	rightOut Point
}

// joinSides returns the shared cross-sections at a vertex and, for bevel or
// round outer corners, a fan polygon [inner, outerStart, arc..., outerEnd].
// Miter joins that stay inside the limit need no extra fill because both
// incident quads use the same inner/outer intersection.
func joinSides(p, din, dout Point, radius float64, style Style) (sideSet, []Point) {
	nin := leftNormal(din)
	nout := leftNormal(dout)
	leftIn := add(p, scale(nin, radius))
	rightIn := add(p, scale(nin, -radius))
	leftOut := add(p, scale(nout, radius))
	rightOut := add(p, scale(nout, -radius))

	c := cross(din, dout)
	dotv := math.Max(-1, math.Min(1, dot(din, dout)))
	if math.Abs(c) <= 1e-12 && dotv > 0 {
		n := normalize(add(nin, nout))
		left := add(p, scale(n, radius))
		right := add(p, scale(n, -radius))
		return sideSet{
			leftIn: left, rightIn: right,
			leftOut: left, rightOut: right,
		}, nil
	}

	phi := math.Acos(dotv)
	if c > 0 {
		inner := intersectLines(
			add(p, scale(nin, radius)), din,
			add(p, scale(nout, radius)), dout,
		)
		if !validPoint(inner) || distance2(inner, p) > (16*radius)*(16*radius) {
			inner = add(p, scale(normalize(add(nin, nout)), radius))
		}
		if style.Join == JoinMiter && phi*4 < math.Pi*3 {
			ratio := 1 / math.Cos(phi/2)
			if ratio <= style.MiterLimit+1e-9 {
				outer := intersectLines(
					add(p, scale(nin, -radius)), din,
					add(p, scale(nout, -radius)), dout,
				)
				if validPoint(outer) &&
					distance2(outer, p) <= (style.MiterLimit*radius+1e-9)*(style.MiterLimit*radius+1e-9) {
					return sideSet{
						leftIn: inner, rightIn: outer,
						leftOut: inner, rightOut: outer,
					}, nil
				}
			}
		}
		ss := sideSet{
			leftIn: inner, rightIn: rightIn,
			leftOut: inner, rightOut: rightOut,
		}
		fill := joinFill(p, inner, rightIn, rightOut, radius, phi, style)
		return ss, fill
	}

	inner := intersectLines(
		add(p, scale(nin, -radius)), din,
		add(p, scale(nout, -radius)), dout,
	)
	if !validPoint(inner) || distance2(inner, p) > (16*radius)*(16*radius) {
		inner = add(p, scale(normalize(add(nin, nout)), -radius))
	}
	if style.Join == JoinMiter && phi*4 < math.Pi*3 {
		ratio := 1 / math.Cos(phi/2)
		if ratio <= style.MiterLimit+1e-9 {
			outer := intersectLines(
				add(p, scale(nin, radius)), din,
				add(p, scale(nout, radius)), dout,
			)
			if validPoint(outer) &&
				distance2(outer, p) <= (style.MiterLimit*radius+1e-9)*(style.MiterLimit*radius+1e-9) {
				return sideSet{
					leftIn: outer, rightIn: inner,
					leftOut: outer, rightOut: inner,
				}, nil
			}
		}
	}
	ss := sideSet{
		leftIn: leftIn, rightIn: inner,
		leftOut: leftOut, rightOut: inner,
	}
	fill := joinFill(p, inner, leftIn, leftOut, -radius, phi, style)
	return ss, fill
}

func joinFill(p, inner, outerA, outerB Point, signedRadius, phi float64, style Style) []Point {
	if style.Join != JoinRound {
		return []Point{inner, outerA, outerB}
	}
	sweep := phi
	if signedRadius < 0 {
		sweep = -phi
	}
	arc := arcPolygon(p, outerA, outerB, math.Abs(signedRadius), sweep, style.ArcTolerance)
	return append([]Point{inner}, arc...)
}

func arcPolygon(center, start, end Point, radius, sweep, tolerance float64) []Point {
	n := arcSegments(radius, math.Abs(sweep), tolerance)
	poly := make([]Point, 0, n+1)
	poly = append(poly, start)
	v := sub(start, center)
	for i := 1; i < n; i++ {
		a := sweep * float64(i) / float64(n)
		r := rotate(v, a)
		poly = append(poly, add(center, r))
	}
	poly = append(poly, end)
	return poly
}

func arcSegments(radius, angle, tolerance float64) int {
	if radius <= 0 || angle <= 0 {
		return 1
	}
	if tolerance <= 0 {
		tolerance = 0.05
	}
	if tolerance >= radius {
		return 4
	}
	maxStep := 2 * math.Acos(1-tolerance/radius)
	if maxStep <= 1e-6 || !isFinite(maxStep) {
		maxStep = math.Pi / 8
	}
	n := ceilInt(angle / maxStep)
	if n < 4 {
		n = 4
	} else {
		n = (n + 3) / 4 * 4
	}
	if n > 64 {
		n = 64
	}
	return n
}

func intersectLines(p, d, q, e Point) Point {
	den := cross(d, e)
	if math.Abs(den) <= 1e-12 {
		return Point{X: math.NaN(), Y: math.NaN()}
	}
	t := cross(sub(q, p), e) / den
	return add(p, scale(d, t))
}

func rotate(p Point, angle float64) Point {
	c, s := math.Cos(angle), math.Sin(angle)
	return Point{X: p.X*c - p.Y*s, Y: p.X*s + p.Y*c}
}

func normalize(p Point) Point {
	l := math.Hypot(p.X, p.Y)
	if l == 0 {
		return Point{}
	}
	return Point{X: p.X / l, Y: p.Y / l}
}

func validPoint(p Point) bool {
	return isFinite(p.X) && isFinite(p.Y)
}

func distance2(a, b Point) float64 {
	dx, dy := a.X-b.X, a.Y-b.Y
	return dx*dx + dy*dy
}

func pointDistance(a, b Point) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}
