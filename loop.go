package holefill

import (
	"fmt"
	"math"
	"sort"
)

func ExtractBoundaryLoops(m *Mesh) ([]*BoundaryLoop, error) {
	if err := m.Validate(); err != nil {
		return nil, err
	}
	use := m.EdgeUseCounts()
	outgoing := make(map[int][]BoundaryEdge)
	var boundary []BoundaryEdge
	for _, t := range m.T {
		dirs := [3][2]int{{t[0], t[1]}, {t[1], t[2]}, {t[2], t[0]}}
		for _, d := range dirs {
			e := BoundaryEdge{A: d[0], B: d[1]}
			if use[undirectedEdge(e.A, e.B)] == 1 {
				outgoing[e.A] = append(outgoing[e.A], e)
				boundary = append(boundary, e)
			}
		}
	}

	used := make(map[BoundaryEdge]bool, len(boundary))
	var loops []*BoundaryLoop
	for _, start := range boundary {
		if used[start] {
			continue
		}
		verts, err := walkBoundary(start, outgoing, used, len(boundary))
		if err != nil {
			return nil, err
		}
		if len(verts) < 3 {
			continue
		}
		loop := &BoundaryLoop{ID: len(loops), Verts: verts, Class: ClassUnknown}
		buildLoopFrame(m, loop)
		measureLoop(loop)
		loops = append(loops, loop)
	}
	sort.Slice(loops, func(i, j int) bool { return loops[i].ID < loops[j].ID })
	for i, l := range loops {
		l.ID = i
	}
	return loops, nil
}

func walkBoundary(start BoundaryEdge, outgoing map[int][]BoundaryEdge, used map[BoundaryEdge]bool, maxSteps int) ([]int, error) {
	cur := start
	startKey := cur
	var verts []int
	for step := 0; step <= maxSteps+1; step++ {
		if used[cur] {
			if cur == startKey {
				return verts, nil
			}
			return nil, fmt.Errorf("boundary walk entered a used edge before closing")
		}
		used[cur] = true
		verts = append(verts, cur.A)
		choices := outgoing[cur.B]
		var next BoundaryEdge
		found := false
		for _, cand := range choices {
			if !used[cand] {
				next = cand
				found = true
				break
			}
		}
		if !found {
			for _, cand := range choices {
				if cand == startKey {
					next = cand
					found = true
					break
				}
			}
		}
		if !found {
			return nil, fmt.Errorf("open boundary chain at vertex %d", cur.B)
		}
		if next == startKey {
			return verts, nil
		}
		cur = next
	}
	return nil, fmt.Errorf("boundary walk did not close within %d edges", maxSteps)
}

func buildLoopFrame(m *Mesh, l *BoundaryLoop) {
	var c Vec3
	var n Vec3
	for _, idx := range l.Verts {
		c = c.Add(m.V[idx])
	}
	c = c.Scale(1 / float64(len(l.Verts)))
	for i := 0; i < len(l.Verts); i++ {
		a := m.V[l.Verts[i]]
		b := m.V[l.Verts[(i+1)%len(l.Verts)]]
		n.X += (a.Y - b.Y) * (a.Z + b.Z)
		n.Y += (a.Z - b.Z) * (a.X + b.X)
		n.Z += (a.X - b.X) * (a.Y + b.Y)
	}
	n = n.Normalize()
	if n.Len() < 0.5 {
		n = Vec3{0, 0, 1}
	}
	ref := Vec3{1, 0, 0}
	if math.Abs(n.X) > 0.85 {
		ref = Vec3{0, 1, 0}
	}
	u := n.Cross(ref).Normalize()
	v := n.Cross(u).Normalize()

	l.Centroid = c
	l.Normal = n
	l.U = u
	l.V = v
	l.Pts = make([]Vec2, len(l.Verts))
	for i, idx := range l.Verts {
		d := m.V[idx].Sub(c)
		l.Pts[i] = Vec2{d.Dot(u), d.Dot(v)}
	}
}

func measureLoop(l *BoundaryLoop) {
	pts := l.Pts
	n := len(pts)
	var signed2 float64
	perimeter := 0.0
	for i := 0; i < n; i++ {
		a := pts[i]
		b := pts[(i+1)%n]
		signed2 += a.X*b.Y - b.X*a.Y
		perimeter += a.Distance(b)
	}
	l.Metrics.SignedArea = signed2 * 0.5
	l.Metrics.Area = math.Abs(signed2) * 0.5
	l.Metrics.Perimeter = perimeter
	l.Metrics.HullArea = convexHullArea(pts)
	if l.Metrics.HullArea > 1e-9 {
		l.Metrics.Convexity = l.Metrics.Area / l.Metrics.HullArea
	} else {
		l.Metrics.Convexity = 1
	}

	length, width, aspect := pcaExtents(pts)
	l.Metrics.Length = length
	l.Metrics.Width = width
	l.Metrics.Aspect = aspect
	l.Metrics.MinSelfGap = minimumSelfGap(pts)
	l.Metrics.ReflexCorners, l.Metrics.RightCorners = countCorners(pts, l.Metrics.SignedArea >= 0)
	l.Metrics.MinPairGap = math.Inf(1)
	l.Metrics.MinOuterGap = math.Inf(1)
}

func (a Vec2) Distance(b Vec2) float64 {
	return math.Hypot(a.X-b.X, a.Y-b.Y)
}

func pcaExtents(pts []Vec2) (float64, float64, float64) {
	var sxx, syy, sxy float64
	n := float64(len(pts))
	var c Vec2
	for _, p := range pts {
		c = c.Add(p)
	}
	c = c.Scale(1 / n)
	for _, p := range pts {
		d := p.Sub(c)
		sxx += d.X * d.X
		syy += d.Y * d.Y
		sxy += d.X * d.Y
	}
	theta := 0.5 * math.Atan2(2*sxy, sxx-syy)
	axis := Vec2{math.Cos(theta), math.Sin(theta)}
	perp := Vec2{-axis.Y, axis.X}
	minA, maxA := math.Inf(1), math.Inf(-1)
	minB, maxB := math.Inf(1), math.Inf(-1)
	for _, p := range pts {
		d := p.Sub(c)
		x, y := d.Dot(axis), d.Dot(perp)
		minA, maxA = math.Min(minA, x), math.Max(maxA, x)
		minB, maxB = math.Min(minB, y), math.Max(maxB, y)
	}
	a := maxA - minA
	b := maxB - minB
	if a < b {
		a, b = b, a
	}
	if b < 1e-9 {
		b = 1e-9
	}
	return a, b, a / b
}

func countCorners(pts []Vec2, ccw bool) (reflex int, right int) {
	n := len(pts)
	for i := 0; i < n; i++ {
		prev := pts[(i-1+n)%n]
		cur := pts[i]
		next := pts[(i+1)%n]
		in := cur.Sub(prev)
		out := next.Sub(cur)
		if in.Len() < 1e-9 || out.Len() < 1e-9 {
			continue
		}
		dot := in.Dot(out) / (in.Len() * out.Len())
		dot = math.Max(-1, math.Min(1, dot))
		unsigned := math.Acos(dot)
		cross := in.Cross(out)
		if !ccw {
			cross = -cross
		}
		signedTurn := math.Atan2(cross, dot)
		if math.Abs(unsigned-math.Pi/2) < 0.42 {
			right++
		}
		if signedTurn < -0.55 && unsigned > 0.7 {
			reflex++
		}
	}
	return reflex, right
}

func minimumSelfGap(pts []Vec2) float64 {
	n := len(pts)
	best := math.Inf(1)
	for i := 0; i < n; i++ {
		a := pts[i]
		b := pts[(i+1)%n]
		for j := i + 1; j < n; j++ {
			if circularEdgeDistance(i, j, n) <= 3 {
				continue
			}
			c := pts[j]
			d := pts[(j+1)%n]
			gap := segmentDistance2D(a, b, c, d)
			if gap < best {
				best = gap
			}
		}
	}
	if math.IsInf(best, 1) {
		return 0
	}
	return best
}

func circularEdgeDistance(i, j, n int) int {
	d := j - i
	if d < 0 {
		d = -d
	}
	if n-d < d {
		d = n - d
	}
	return d
}

func convexHullArea(pts []Vec2) float64 {
	if len(pts) < 3 {
		return 0
	}
	p := append([]Vec2(nil), pts...)
	sort.Slice(p, func(i, j int) bool {
		if p[i].X == p[j].X {
			return p[i].Y < p[j].Y
		}
		return p[i].X < p[j].X
	})
	var lower, upper []Vec2
	for _, q := range p {
		for len(lower) >= 2 && cross2(lower[len(lower)-2], lower[len(lower)-1], q) <= 0 {
			lower = lower[:len(lower)-1]
		}
		lower = append(lower, q)
	}
	for i := len(p) - 1; i >= 0; i-- {
		q := p[i]
		for len(upper) >= 2 && cross2(upper[len(upper)-2], upper[len(upper)-1], q) <= 0 {
			upper = upper[:len(upper)-1]
		}
		upper = append(upper, q)
	}
	hull := append(lower[:len(lower)-1], upper[:len(upper)-1]...)
	return math.Abs(polygonArea(hull))
}

func cross2(a, b, c Vec2) float64 {
	return b.Sub(a).Cross(c.Sub(a))
}

func polygonArea(p []Vec2) float64 {
	var s float64
	for i := range p {
		a := p[i]
		b := p[(i+1)%len(p)]
		s += a.X*b.Y - b.X*a.Y
	}
	return s * 0.5
}
