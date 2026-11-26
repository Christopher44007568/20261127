package holefill

import "math"

const geomEps = 1e-10

func orient2(a, b, c Vec2) float64 {
	return b.Sub(a).Cross(c.Sub(a))
}

func onSegment(a, b, p Vec2) bool {
	return math.Abs(orient2(a, b, p)) < 1e-8 &&
		p.X >= math.Min(a.X, b.X)-1e-8 && p.X <= math.Max(a.X, b.X)+1e-8 &&
		p.Y >= math.Min(a.Y, b.Y)-1e-8 && p.Y <= math.Max(a.Y, b.Y)+1e-8
}

func segmentsIntersect(a, b, c, d Vec2) bool {
	o1 := orient2(a, b, c)
	o2 := orient2(a, b, d)
	o3 := orient2(c, d, a)
	o4 := orient2(c, d, b)
	if (o1 > 1e-9 && o2 < -1e-9 || o1 < -1e-9 && o2 > 1e-9) &&
		(o3 > 1e-9 && o4 < -1e-9 || o3 < -1e-9 && o4 > 1e-9) {
		return true
	}
	if math.Abs(o1) <= 1e-8 && onSegment(a, b, c) {
		return true
	}
	if math.Abs(o2) <= 1e-8 && onSegment(a, b, d) {
		return true
	}
	if math.Abs(o3) <= 1e-8 && onSegment(c, d, a) {
		return true
	}
	if math.Abs(o4) <= 1e-8 && onSegment(c, d, b) {
		return true
	}
	return false
}

func pointSegmentDistance(p, a, b Vec2) float64 {
	ab := b.Sub(a)
	l2 := ab.Dot(ab)
	if l2 < geomEps {
		return p.Distance(a)
	}
	t := p.Sub(a).Dot(ab) / l2
	t = math.Max(0, math.Min(1, t))
	q := a.Add(ab.Scale(t))
	return p.Distance(q)
}

func segmentDistance2D(a, b, c, d Vec2) float64 {
	if segmentsIntersect(a, b, c, d) {
		return 0
	}
	return math.Min(
		math.Min(pointSegmentDistance(a, c, d), pointSegmentDistance(b, c, d)),
		math.Min(pointSegmentDistance(c, a, b), pointSegmentDistance(d, a, b)),
	)
}

func pointInTriangle(p, a, b, c Vec2) bool {
	o1 := orient2(a, b, p)
	o2 := orient2(b, c, p)
	o3 := orient2(c, a, p)
	hasNeg := o1 < -1e-9 || o2 < -1e-9 || o3 < -1e-9
	hasPos := o1 > 1e-9 || o2 > 1e-9 || o3 > 1e-9
	return !(hasNeg && hasPos)
}

func pointInPolygon(p Vec2, poly []Vec2) bool {
	inside := false
	for i := 0; i < len(poly); i++ {
		j := (i + 1) % len(poly)
		a, b := poly[i], poly[j]
		if onSegment(a, b, p) {
			return true
		}
		if (a.Y > p.Y) != (b.Y > p.Y) {
			x := a.X + (p.Y-a.Y)*(b.X-a.X)/(b.Y-a.Y)
			if x > p.X {
				inside = !inside
			}
		}
	}
	return inside
}

func minLoopDistance2D(a, b *BoundaryLoop) float64 {
	best := math.Inf(1)
	for i := 0; i < len(a.Pts); i++ {
		a0 := a.Pts[i]
		a1 := a.Pts[(i+1)%len(a.Pts)]
		for j := 0; j < len(b.Pts); j++ {
			b0 := b.Pts[j]
			b1 := b.Pts[(j+1)%len(b.Pts)]
			gap := segmentDistance2D(a0, a1, b0, b1)
			if gap < best {
				best = gap
			}
		}
	}
	return best
}

func slerpNormal(a, b Vec3, t float64) Vec3 {
	dot := math.Max(-1, math.Min(1, a.Dot(b)))
	theta := math.Acos(dot)
	if theta < 1e-6 {
		return a
	}
	sin := math.Sin(theta)
	wa := math.Sin((1-t)*theta) / sin
	wb := math.Sin(t*theta) / sin
	return a.Scale(wa).Add(b.Scale(wb)).Normalize()
}
