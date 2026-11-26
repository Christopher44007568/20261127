package demo

import "math"

// Vec2 is a point in device pixel coordinates. Pixels occupy unit squares
// [x, x+1) x [y, y+1); pixel (ix, iy) has its center at (ix+0.5, iy+0.5).
type Vec2 struct {
	X, Y float64
}

// Winding selects the signed orientation of a subpath. In screen coordinates
// (Y grows downwards) WindingClockwise has positive shoelace area.
type Winding int

const (
	WindingAny          Winding = 0
	WindingClockwise    Winding = 1
	WindingCounterClock Winding = -1
)

type segmentKind int

const (
	segLine segmentKind = iota
	segQuad
	segCubic
	segClose
)

type segment struct {
	kind segmentKind
	// pts holds the endpoint plus control points, depending on kind:
	// line: [end], quad: [control, end], cubic: [c1, c2, end].
	pts []Vec2
}

// Subpath is one MoveTo-started contour.
type Subpath struct {
	start    Vec2
	segments []segment
	closed   bool
}

// Path is a collection of nested/overlapping subpaths filled together,
// exactly like a canvas/SVG path. All subpaths participate in a single
// fill-rule evaluation so outer contours can never be painted "over" holes.
type Path struct {
	subs []*Subpath
	cur  *Subpath
	pos  Vec2
}

// MoveTo starts a new subpath.
func (p *Path) MoveTo(x, y float64) *Path {
	s := &Subpath{start: Vec2{x, y}}
	p.subs = append(p.subs, s)
	p.cur = s
	p.pos = s.start
	return p
}

// LineTo appends a straight segment.
func (p *Path) LineTo(x, y float64) *Path {
	p.ensure(x, y)
	p.cur.segments = append(p.cur.segments, segment{kind: segLine, pts: []Vec2{{x, y}}})
	p.pos = Vec2{x, y}
	return p
}

// QuadTo appends a quadratic Bézier segment.
func (p *Path) QuadTo(cx, cy, x, y float64) *Path {
	p.ensure(x, y)
	p.cur.segments = append(p.cur.segments, segment{kind: segQuad, pts: []Vec2{{cx, cy}, {x, y}}})
	p.pos = Vec2{x, y}
	return p
}

// CubicTo appends a cubic Bézier segment.
func (p *Path) CubicTo(c1x, c1y, c2x, c2y, x, y float64) *Path {
	p.ensure(x, y)
	p.cur.segments = append(p.cur.segments, segment{
		kind: segCubic,
		pts:  []Vec2{{c1x, c1y}, {c2x, c2y}, {x, y}},
	})
	p.pos = Vec2{x, y}
	return p
}

// Close closes the current subpath with a straight segment back to its start.
func (p *Path) Close() *Path {
	if p.cur != nil {
		p.cur.closed = true
		p.pos = p.cur.start
	}
	return p
}

func (p *Path) ensure(x, y float64) {
	if p.cur == nil {
		p.MoveTo(x, y)
	}
}

// Subpaths returns the raw subpaths.
func (p *Path) Subpaths() []*Subpath { return p.subs }

// Flatten converts every subpath to a closed or open polyline. Curves are
// subdivided until their chord error is below tol pixels.
func (p *Path) Flatten(tol float64) [][]Vec2 {
	if tol <= 0 {
		tol = 0.35
	}
	var loops [][]Vec2
	for _, s := range p.subs {
		loop := []Vec2{s.start}
		cur := s.start
		for _, seg := range s.segments {
			switch seg.kind {
			case segLine:
				loop = append(loop, seg.pts[0])
				cur = seg.pts[0]
			case segQuad:
				quadFlatten(cur, seg.pts[0], seg.pts[1], tol, &loop)
				cur = seg.pts[1]
			case segCubic:
				cubicFlatten(cur, seg.pts[0], seg.pts[1], seg.pts[2], tol, &loop)
				cur = seg.pts[2]
			case segClose:
				// Close is represented by closed flag; nothing to append.
			}
		}
		if s.closed {
			if last := loop[len(loop)-1]; last != s.start {
				loop = append(loop, s.start)
			}
		}
		loops = append(loops, loop)
	}
	return loops
}

func quadFlatten(p0, c, p1 Vec2, tol float64, out *[]Vec2) {
	// De Casteljau adaptive subdivision.
	var rec func(a, ca, b Vec2, depth int)
	rec = func(a, ca, b Vec2, depth int) {
		if depth > 18 || quadFlatEnough(a, ca, b, tol) {
			*out = append(*out, b)
			return
		}
		c1 := mid(a, ca)
		c2 := mid(ca, b)
		mid := mid(c1, c2)
		rec(a, c1, mid, depth+1)
		rec(mid, c2, b, depth+1)
	}
	rec(p0, c, p1, 0)
}

func quadFlatEnough(p0, c, p1 Vec2, tol float64) bool {
	d := pointSegmentDist(c, p0, p1)
	return d <= tol
}

func cubicFlatten(p0, c1, c2, p1 Vec2, tol float64, out *[]Vec2) {
	var rec func(a, ca, cb, b Vec2, depth int)
	rec = func(a, ca, cb, b Vec2, depth int) {
		if depth > 18 || cubicFlatEnough(a, ca, cb, b, tol) {
			*out = append(*out, b)
			return
		}
		m1 := mid(a, ca)
		m2 := mid(ca, cb)
		m3 := mid(cb, b)
		m12 := mid(m1, m2)
		m23 := mid(m2, m3)
		mid := mid(m12, m23)
		rec(a, m1, m12, mid, depth+1)
		rec(mid, m23, m3, b, depth+1)
	}
	rec(p0, c1, c2, p1, 0)
}

func cubicFlatEnough(p0, c1, c2, p1 Vec2, tol float64) bool {
	return pointSegmentDist(c1, p0, p1) <= tol && pointSegmentDist(c2, p0, p1) <= tol
}

func mid(a, b Vec2) Vec2 { return Vec2{(a.X + b.X) / 2, (a.Y + b.Y) / 2} }

// pointSegmentDist returns the perpendicular distance of p to the line a-b.
// Curve flatness only needs the cheap line distance.
func pointSegmentDist(p, a, b Vec2) float64 {
	dx, dy := b.X-a.X, b.Y-a.Y
	l2 := dx*dx + dy*dy
	if l2 == 0 {
		return math.Hypot(p.X-a.X, p.Y-a.Y)
	}
	t := ((p.X-a.X)*dx + (p.Y-a.Y)*dy) / l2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	qx, qy := a.X+t*dx, a.Y+t*dy
	return math.Hypot(p.X-qx, p.Y-qy)
}

// SignedArea2 returns twice the signed shoelace area of a loop.
// Positive means clockwise on a Y-down device context.
func SignedArea2(loop []Vec2) float64 {
	var a float64
	for i := range loop {
		j := (i + 1) % len(loop)
		a += loop[i].X*loop[j].Y - loop[j].X*loop[i].Y
	}
	return a
}

// OrientLoop returns a copy of loop reversed if needed to match w.
func OrientLoop(loop []Vec2, w Winding) []Vec2 {
	if w == WindingAny {
		return loop
	}
	wantPositive := w == WindingClockwise
	positive := SignedArea2(loop) > 0
	if positive != wantPositive {
		out := make([]Vec2, len(loop))
		for i := range loop {
			out[i] = loop[len(loop)-1-i]
		}
		return out
	}
	return loop
}

// AddRect appends an axis-aligned rectangle subpath with requested orientation.
func AddRect(p *Path, x0, y0, x1, y1 float64, w Winding) *Path {
	pts := []Vec2{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
	pts = OrientLoop(pts, w)
	p.MoveTo(pts[0].X, pts[0].Y)
	for _, q := range pts[1:] {
		p.LineTo(q.X, q.Y)
	}
	return p.Close()
}

// AddDiamond appends a diamond centered at cx,cy with horizontal/vertical radii.
func AddDiamond(p *Path, cx, cy, rx, ry float64, w Winding) *Path {
	pts := []Vec2{{cx, cy - ry}, {cx + rx, cy}, {cx, cy + ry}, {cx - rx, cy}}
	pts = OrientLoop(pts, w)
	p.MoveTo(pts[0].X, pts[0].Y)
	for _, q := range pts[1:] {
		p.LineTo(q.X, q.Y)
	}
	return p.Close()
}

// AddRoundedRect appends a rounded rectangle. r is the corner radius.
func AddRoundedRect(p *Path, x0, y0, x1, y1, r float64, w Winding) *Path {
	if 2*r > x1-x0 {
		r = (x1 - x0) / 2
	}
	if 2*r > y1-y0 {
		r = (y1 - y0) / 2
	}
	// Build a clockwise screen outline starting at the top straight edge.
	var q Path
	q.MoveTo(x0+r, y0)
	q.LineTo(x1-r, y0)
	q.QuadTo(x1, y0, x1, y0+r)
	q.LineTo(x1, y1-r)
	q.QuadTo(x1, y1, x1-r, y1)
	q.LineTo(x0+r, y1)
	q.QuadTo(x0, y1, x0, y1-r)
	q.LineTo(x0, y0+r)
	q.QuadTo(x0, y0, x0+r, y0)
	q.Close()
	loop := OrientLoop(q.Flatten(0.5)[0], w)
	p.MoveTo(loop[0].X, loop[0].Y)
	for _, q := range loop[1 : len(loop)-1] {
		p.LineTo(q.X, q.Y)
	}
	return p.Close()
}

// AddCircle appends a circle approximated by four cubic Bézier arcs.
func AddCircle(p *Path, cx, cy, r float64, w Winding) *Path {
	const k = 0.5522847498307936 // 4/3*tan(pi/8)
	kp := k * r
	var q Path
	q.MoveTo(cx, cy-r)
	q.CubicTo(cx+kp, cy-r, cx+r, cy-kp, cx+r, cy)
	q.CubicTo(cx+r, cy+kp, cx+kp, cy+r, cx, cy+r)
	q.CubicTo(cx-kp, cy+r, cx-r, cy+kp, cx-r, cy)
	q.CubicTo(cx-r, cy-kp, cx-kp, cy-r, cx, cy-r)
	q.Close()
	loop := OrientLoop(q.Flatten(0.4)[0], w)
	p.MoveTo(loop[0].X, loop[0].Y)
	for _, pt := range loop[1 : len(loop)-1] {
		p.LineTo(pt.X, pt.Y)
	}
	return p.Close()
}

// AddPentagram appends the classic self-intersecting 5-point star as one
// continuous contour. Its central pentagon has winding number 2 but even
// odd parity 0, which is exactly where the two fill rules disagree.
func AddPentagram(p *Path, cx, cy, outerR float64) *Path {
	var pts []Vec2
	for i := 0; i < 5; i++ {
		a := -math.Pi/2 + float64(i)*4*math.Pi/5
		pts = append(pts, Vec2{
			X: round1(cx + outerR*math.Cos(a)),
			Y: round1(cy + outerR*math.Sin(a)),
		})
	}
	p.MoveTo(pts[0].X, pts[0].Y)
	for _, q := range pts[1:] {
		p.LineTo(q.X, q.Y)
	}
	return p.Close()
}

func round1(v float64) float64 { return math.Round(v*10) / 10 }
