package main

import "math"

// Segment is a closed line segment.
type Segment struct {
	A Vec2 `json:"a"`
	B Vec2 `json:"b"`
}

func ContourSegments(c Contour) []Segment {
	if len(c) < 2 {
		return nil
	}
	segments := make([]Segment, 0, len(c))
	for i := range c {
		segments = append(segments, Segment{A: c[i], B: c[(i+1)%len(c)]})
	}
	return segments
}

// PointSegmentDistance returns Euclidean distance and clamped projection t.
func PointSegmentDistance(p, a, b Vec2) (distance, t float64) {
	ab := b.Sub(a)
	l2 := ab.Len2()
	if l2 <= 1e-18 {
		return p.Sub(a).Len(), 0
	}
	t = p.Sub(a).Dot(ab) / l2
	t = math.Max(0, math.Min(1, t))
	q := a.Add(ab.Scale(t))
	return p.Sub(q).Len(), t
}

func SegmentDistance(a, b Segment) float64 {
	if hit, ok := SegmentIntersection(a.A, a.B, b.A, b.B); ok {
		_ = hit
		return 0
	}
	d1, _ := PointSegmentDistance(a.A, b.A, b.B)
	d2, _ := PointSegmentDistance(a.B, b.A, b.B)
	d3, _ := PointSegmentDistance(b.A, a.A, a.B)
	d4, _ := PointSegmentDistance(b.B, a.A, a.B)
	return math.Min(math.Min(d1, d2), math.Min(d3, d4))
}

// DistanceToContour is the minimum distance to any boundary edge.
func DistanceToContour(p Vec2, c Contour) float64 {
	best := math.Inf(1)
	for i := range c {
		d, _ := PointSegmentDistance(p, c[i], c[(i+1)%len(c)])
		best = math.Min(best, d)
	}
	return best
}

// DistanceToOutline is the unsigned boundary distance.
func DistanceToOutline(p Vec2, o Outline) float64 {
	best := math.Inf(1)
	for _, c := range o.Contours() {
		best = math.Min(best, DistanceToContour(p, c))
	}
	return best
}

// SignedDistance returns negative values inside filled material and positive
// values in exterior/counter empty space.
func SignedDistance(p Vec2, o Outline) float64 {
	d := DistanceToOutline(p, o)
	if o.Contains(p) {
		return -d
	}
	return d
}

// MinimumNonAdjacentDistance gives a conservative collision budget for one
// contour. Shared-vertex edges are excluded because the corner routine owns
// that local case.
func MinimumNonAdjacentDistance(c Contour) float64 {
	n := len(c)
	if n < 4 {
		return math.Inf(1)
	}
	best := math.Inf(1)
	for i := 0; i < n; i++ {
		a := Segment{A: c[i], B: c[(i+1)%n]}
		for j := i + 1; j < n; j++ {
			if j == i+1 || (i == 0 && j == n-1) {
				continue
			}
			b := Segment{A: c[j], B: c[(j+1)%n]}
			best = math.Min(best, SegmentDistance(a, b))
		}
	}
	return best
}

// DistanceField is a small segment index used by diagnostics and rendering.
type DistanceField struct {
	Segments []Segment
}

func NewDistanceField(o Outline) DistanceField {
	var segments []Segment
	for _, c := range o.Contours() {
		segments = append(segments, ContourSegments(c)...)
	}
	return DistanceField{Segments: segments}
}

func (f DistanceField) Distance(p Vec2) float64 {
	best := math.Inf(1)
	for _, s := range f.Segments {
		d, _ := PointSegmentDistance(p, s.A, s.B)
		best = math.Min(best, d)
	}
	return best
}
