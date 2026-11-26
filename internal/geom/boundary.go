package geom

import (
	"math"
	"sort"
)

type isect struct {
	t, u float64
}

func SegmentIntersection(a, b Segment) (t, u float64, ok bool) {
	p := a.A
	r := a.B.Sub(a.A)
	q := b.A
	s := b.B.Sub(b.A)
	den := r.Cross(s)
	qp := q.Sub(p)
	if math.Abs(den) < 1e-12 {
		return 0, 0, false
	}
	t = qp.Cross(s) / den
	u = qp.Cross(r) / den
	if t < -1e-8 || t > 1+1e-8 || u < -1e-8 || u > 1+1e-8 {
		return 0, 0, false
	}
	return math.Max(0, math.Min(1, t)), math.Max(0, math.Min(1, u)), true
}

// ExtractBoundaryShape splits crossing contours and discards segments whose
// two infinitesimal sides have the same fill state. This removes internal
// seams from overlapping same-winding contours.
func ExtractBoundaryShape(shape Shape, sideEpsilon float64) Shape {
	if sideEpsilon <= 0 {
		sideEpsilon = 0.03
	}
	var segs []Segment
	for _, c := range shape.Contours {
		n := len(c.Points)
		if n < 2 {
			continue
		}
		limit := n
		if n >= 3 {
			limit = n
		}
		for i := 0; i < limit; i++ {
			segs = append(segs, Segment{A: c.Points[i], B: c.Points[(i+1)%n]})
		}
	}

	cuts := make([][]float64, len(segs))
	for i := range segs {
		cuts[i] = []float64{0, 1}
	}
	for i := 0; i < len(segs); i++ {
		for j := i + 1; j < len(segs); j++ {
			t, u, ok := SegmentIntersection(segs[i], segs[j])
			if !ok {
				continue
			}
			cuts[i] = append(cuts[i], t)
			cuts[j] = append(cuts[j], u)
		}
	}

	var boundary []Segment
	for i, seg := range segs {
		ts := uniqueSorted(cuts[i])
		d := seg.B.Sub(seg.A)
		l := d.Len()
		if l < 1e-10 {
			continue
		}
		nrm := V{X: -d.Y / l, Y: d.X / l}
		for k := 0; k+1 < len(ts); k++ {
			t0, t1 := ts[k], ts[k+1]
			if t1-t0 < 1e-7 {
				continue
			}
			a := seg.A.Add(d.Mul(t0))
			b := seg.A.Add(d.Mul(t1))
			mid := a.Add(b).Mul(0.5)
			plus := Inside(mid.Add(nrm.Mul(sideEpsilon)), shape.Contours)
			minus := Inside(mid.Sub(nrm.Mul(sideEpsilon)), shape.Contours)
			if plus == minus {
				continue
			}
			if plus {
				boundary = append(boundary, Segment{A: a, B: b})
			} else {
				boundary = append(boundary, Segment{A: b, B: a})
			}
		}
	}
	return stitchSegments(boundary)
}

func uniqueSorted(in []float64) []float64 {
	sort.Float64s(in)
	out := in[:0]
	for _, v := range in {
		if len(out) == 0 || math.Abs(v-out[len(out)-1]) > 1e-7 {
			out = append(out, v)
		}
	}
	return out
}

type pointKey struct {
	x, y int64
}

func keyOf(p V) pointKey {
	return pointKey{x: int64(math.Round(p.X * 1e5)), y: int64(math.Round(p.Y * 1e5))}
}

func stitchSegments(segs []Segment) Shape {
	byStart := map[pointKey][]int{}
	for i, s := range segs {
		k := keyOf(s.A)
		byStart[k] = append(byStart[k], i)
	}
	used := make([]bool, len(segs))
	shape := Shape{}
	for startIdx := 0; startIdx < len(segs); startIdx++ {
		if used[startIdx] {
			continue
		}
		used[startIdx] = true
		first := segs[startIdx]
		startKey := keyOf(first.A)
		points := []V{first.A, first.B}
		currentKey := keyOf(first.B)
		for steps := 0; steps < len(segs)+2; steps++ {
			if currentKey == startKey {
				if len(points) > 1 && points[len(points)-1].Dist(points[0]) < 1e-4 {
					points = points[:len(points)-1]
				}
				break
			}
			candidates := byStart[currentKey]
			next := -1
			for _, idx := range candidates {
				if !used[idx] {
					next = idx
					break
				}
			}
			if next < 0 {
				break
			}
			used[next] = true
			s := segs[next]
			points = append(points, s.B)
			currentKey = keyOf(s.B)
		}
		if len(points) >= 3 {
			shape.Contours = append(shape.Contours, Contour{Points: points})
		}
	}
	return shape
}
