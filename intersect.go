package main

import "math"

const geometryEps = 1e-9

// SegmentHit describes an intersection between two line segments.
type SegmentHit struct {
	Point     Vec2
	T         float64
	U         float64
	Collinear bool
}

// Intersection is a located non-adjacent edge crossing or touch.
type Intersection struct {
	Point     Vec2   `json:"point"`
	ContourA  int    `json:"contour_a"`
	SegmentA  int    `json:"segment_a"`
	ContourB  int    `json:"contour_b"`
	SegmentB  int    `json:"segment_b"`
	Collinear bool   `json:"collinear,omitempty"`
	Kind      string `json:"kind"`
}

func approxZero(x float64) bool { return math.Abs(x) <= geometryEps }

// SegmentIntersection finds proper crossings, endpoint touches, and
// overlapping collinear segments.
func SegmentIntersection(a, b, c, d Vec2) (SegmentHit, bool) {
	r := b.Sub(a)
	s := d.Sub(c)
	rxs := r.Cross(s)
	qp := c.Sub(a)
	qpxr := qp.Cross(r)

	if approxZero(rxs) && approxZero(qpxr) {
		r2 := r.Len2()
		if r2 <= 1e-18 {
			dist, _ := PointSegmentDistance(a, c, d)
			if dist <= geometryEps*100 {
				return SegmentHit{Point: a}, true
			}
			return SegmentHit{}, false
		}
		t0 := qp.Dot(r) / r2
		t1 := d.Sub(a).Dot(r) / r2
		lo := math.Max(0, math.Min(t0, t1))
		hi := math.Min(1, math.Max(t0, t1))
		if lo > hi+1e-8 {
			return SegmentHit{}, false
		}
		t := math.Max(0, math.Min(1, (lo+hi)*0.5))
		return SegmentHit{Point: a.Add(r.Scale(t)), T: t, U: 0, Collinear: true}, true
	}

	if approxZero(rxs) {
		return SegmentHit{}, false
	}

	t := qp.Cross(s) / rxs
	u := qpxr / rxs
	eps := 1e-8
	if t < -eps || t > 1+eps || u < -eps || u > 1+eps {
		return SegmentHit{}, false
	}
	t = math.Max(0, math.Min(1, t))
	u = math.Max(0, math.Min(1, u))
	return SegmentHit{Point: a.Add(r.Scale(t)), T: t, U: u}, true
}

type indexedSegment struct {
	contour int
	index   int
	seg     Segment
}

func outlineIndexedSegments(o Outline) []indexedSegment {
	var result []indexedSegment
	for ci, c := range o.Contours() {
		for si, s := range ContourSegments(c) {
			result = append(result, indexedSegment{contour: ci, index: si, seg: s})
		}
	}
	return result
}

// FindOutlineIntersections returns all forbidden non-adjacent intersections.
func FindOutlineIntersections(o Outline) []Intersection {
	segments := outlineIndexedSegments(o)
	contours := o.Contours()
	result := make([]Intersection, 0)

	for i := 0; i < len(segments); i++ {
		for j := i + 1; j < len(segments); j++ {
			a := segments[i]
			b := segments[j]
			if a.contour == b.contour {
				n := len(contours[a.contour])
				diff := a.index - b.index
				if diff < 0 {
					diff = -diff
				}
				if diff == 1 || diff == n-1 {
					continue
				}
			}
			hit, ok := SegmentIntersection(a.seg.A, a.seg.B, b.seg.A, b.seg.B)
			if !ok {
				continue
			}
			kind := "cross"
			if hit.Collinear {
				kind = "collinear"
			} else if hit.T <= 1e-7 || hit.T >= 1-1e-7 || hit.U <= 1e-7 || hit.U >= 1-1e-7 {
				kind = "touch"
			}
			result = append(result, Intersection{
				Point:     hit.Point,
				ContourA:  a.contour,
				SegmentA:  a.index,
				ContourB:  b.contour,
				SegmentB:  b.index,
				Collinear: hit.Collinear,
				Kind:      kind,
			})
		}
	}
	return result
}

// FindContourIntersections is a convenience wrapper for one contour.
func FindContourIntersections(c Contour) []Intersection {
	return FindOutlineIntersections(Outline{Outer: c})
}
