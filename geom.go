package main

import "math"

// Pt is a 2D point in scene user units (1 unit == 1 SVG px).
type Pt struct {
	X, Y float64
}

func (a Pt) add(b Pt) Pt      { return Pt{a.X + b.X, a.Y + b.Y} }
func (a Pt) sub(b Pt) Pt      { return Pt{a.X - b.X, a.Y - b.Y} }
func (a Pt) mul(k float64) Pt { return Pt{a.X * k, a.Y * k} }

func cross(a, b Pt) float64 { return a.X*b.Y - a.Y*b.X }

// Cmd is one path command. Op is M/L/Q/C/Z; P holds control/end points:
// M,L: P[0]=point; Q: P[0]=control, P[1]=end; C: P[0],P[1]=controls, P[2]=end.
type Cmd struct {
	Op byte
	P  [3]Pt
}

// Path is an editable vector path. Commands are preserved for SVG export.
type Path struct {
	Cmds []Cmd
}

func (p *Path) move(x, y float64) { p.Cmds = append(p.Cmds, Cmd{Op: 'M', P: [3]Pt{{x, y}}}) }
func (p *Path) line(x, y float64) { p.Cmds = append(p.Cmds, Cmd{Op: 'L', P: [3]Pt{{x, y}}}) }
func (p *Path) quad(cx, cy, x, y float64) {
	p.Cmds = append(p.Cmds, Cmd{Op: 'Q', P: [3]Pt{{cx, cy}, {x, y}}})
}
func (p *Path) cubic(c1x, c1y, c2x, c2y, x, y float64) {
	p.Cmds = append(p.Cmds, Cmd{Op: 'C', P: [3]Pt{{c1x, c1y}, {c2x, c2y}, {x, y}}})
}
func (p *Path) close() { p.Cmds = append(p.Cmds, Cmd{Op: 'Z'}) }

// Sub is a flattened polyline sub-path.
type Sub struct {
	Pts    []Pt
	Closed bool
}

// Flatten adaptively subdivides curves until their chord error is below tol.
func (p *Path) Flatten(tol float64) []Sub {
	var subs []Sub
	var cur, start Pt
	var pts []Pt
	closed := false

	flush := func() {
		if len(pts) >= 2 {
			subs = append(subs, Sub{Pts: pts, Closed: closed})
		}
		pts = nil
		closed = false
	}

	for _, c := range p.Cmds {
		switch c.Op {
		case 'M':
			flush()
			cur = c.P[0]
			start = cur
			pts = []Pt{cur}
		case 'L':
			cur = c.P[0]
			pts = append(pts, cur)
		case 'Q':
			q := [3]Pt{cur, c.P[0], c.P[1]}
			flattenQuad(q, tol, 0, &pts)
			cur = c.P[1]
		case 'C':
			q := [4]Pt{cur, c.P[0], c.P[1], c.P[2]}
			flattenCubic(q, tol, 0, &pts)
			cur = c.P[2]
		case 'Z':
			closed = true
			cur = start
		}
	}
	flush()
	return subs
}

func flatEnough(pts []Pt, tol float64) bool {
	a, b := pts[0], pts[len(pts)-1]
	ab := b.sub(a)
	l2 := ab.X*ab.X + ab.Y*ab.Y
	worst := 0.0
	for _, p := range pts[1 : len(pts)-1] {
		var d float64
		if l2 == 0 {
			d = math.Hypot(p.X-a.X, p.Y-a.Y)
		} else {
			t := ((p.X-a.X)*ab.X + (p.Y-a.Y)*ab.Y) / l2
			if t < 0 {
				t = 0
			} else if t > 1 {
				t = 1
			}
			proj := a.add(ab.mul(t))
			d = math.Hypot(p.X-proj.X, p.Y-proj.Y)
		}
		if d > worst {
			worst = d
		}
	}
	return worst <= tol
}

func flattenQuad(q [3]Pt, tol float64, depth int, out *[]Pt) {
	if depth > 16 || flatEnough([]Pt{q[0], q[1], q[2]}, tol) {
		*out = append(*out, q[2])
		return
	}
	q01 := mid(q[0], q[1])
	q12 := mid(q[1], q[2])
	qm := mid(q01, q12)
	flattenQuad([3]Pt{q[0], q01, qm}, tol, depth+1, out)
	flattenQuad([3]Pt{qm, q12, q[2]}, tol, depth+1, out)
}

func flattenCubic(q [4]Pt, tol float64, depth int, out *[]Pt) {
	if depth > 18 || flatEnough([]Pt{q[0], q[1], q[2], q[3]}, tol) {
		*out = append(*out, q[3])
		return
	}
	q01 := mid(q[0], q[1])
	q12 := mid(q[1], q[2])
	q23 := mid(q[2], q[3])
	q012 := mid(q01, q12)
	q123 := mid(q12, q23)
	qm := mid(q012, q123)
	flattenCubic([4]Pt{q[0], q01, q012, qm}, tol, depth+1, out)
	flattenCubic([4]Pt{qm, q123, q23, q[3]}, tol, depth+1, out)
}

func mid(a, b Pt) Pt { return Pt{(a.X + b.X) / 2, (a.Y + b.Y) / 2} }

// BBox returns the control-point hull bounds (a superset of curve bounds).
func (p *Path) BBox() (minX, minY, maxX, maxY float64) {
	minX, minY = math.Inf(1), math.Inf(1)
	maxX, maxY = math.Inf(-1), math.Inf(-1)
	for _, c := range p.Cmds {
		pts := c.P[:]
		n := 1
		if c.Op == 'Q' {
			n = 2
		} else if c.Op == 'C' {
			n = 3
		}
		if c.Op == 'Z' {
			continue
		}
		if c.Op == 'M' || c.Op == 'L' {
			n = 1
		}
		for i := 0; i < n; i++ {
			pt := pts[i]
			minX = math.Min(minX, pt.X)
			maxX = math.Max(maxX, pt.X)
			minY = math.Min(minY, pt.Y)
			maxY = math.Max(maxY, pt.Y)
		}
	}
	return
}

// Contains reports q inside any path using nonzero winding (fill is implicitly
// closed, matching SVG fill semantics).
func Contains(paths []*Path, q Pt) bool {
	wn := 0
	for _, p := range paths {
		for _, s := range p.Flatten(0.2) {
			n := len(s.Pts)
			if n < 3 {
				continue
			}
			edges := n
			for i := 0; i < edges; i++ {
				a := s.Pts[i]
				b := s.Pts[(i+1)%n]
				if !s.Closed && i == n-1 {
					break
				}
				if a.Y <= q.Y {
					if b.Y > q.Y && cross(b.sub(a), q.sub(a)) > 0 {
						wn++
					}
				} else if b.Y <= q.Y && cross(b.sub(a), q.sub(a)) < 0 {
					wn--
				}
			}
		}
	}
	return wn != 0
}

// StrokeHit reports q within halfW of the flattened polyline. Distance to a
// segment union equals a round-cap / round-join stroke, which is exactly the
// line style this scene exports.
func StrokeHit(paths []*Path, q Pt, halfW, tol float64) bool {
	for _, p := range paths {
		for _, s := range p.Flatten(0.25) {
			n := len(s.Pts)
			if n < 2 {
				continue
			}
			last := n - 1
			if s.Closed {
				last = n
			}
			for i := 0; i < last; i++ {
				a := s.Pts[i%n]
				b := s.Pts[(i+1)%n]
				if distToSeg(q, a, b) <= halfW+tol {
					return true
				}
			}
		}
	}
	return false
}

func distToSeg(p, a, b Pt) float64 {
	ab := b.sub(a)
	l2 := ab.X*ab.X + ab.Y*ab.Y
	if l2 == 0 {
		return math.Hypot(p.X-a.X, p.Y-a.Y)
	}
	t := ((p.X-a.X)*ab.X + (p.Y-a.Y)*ab.Y) / l2
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	dx := a.X + ab.X*t - p.X
	dy := a.Y + ab.Y*t - p.Y
	return math.Hypot(dx, dy)
}

// ---- shape constructors ----

func RectPath(x, y, w, h float64) *Path {
	p := &Path{}
	p.move(x, y)
	p.line(x+w, y)
	p.line(x+w, y+h)
	p.line(x, y+h)
	p.close()
	return p
}

func RoundRectPath(x, y, w, h, r float64) *Path {
	if r > w/2 {
		r = w / 2
	}
	if r > h/2 {
		r = h / 2
	}
	p := &Path{}
	p.move(x+r, y)
	p.line(x+w-r, y)
	p.quad(x+w, y, x+w, y+r)
	p.line(x+w, y+h-r)
	p.quad(x+w, y+h, x+w-r, y+h)
	p.line(x+r, y+h)
	p.quad(x, y+h, x, y+h-r)
	p.line(x, y+r)
	p.quad(x, y, x+r, y)
	p.close()
	return p
}

func EllipsePath(cx, cy, rx, ry float64) *Path {
	p := &Path{}
	k := 0.5522847498
	p.move(cx, cy-ry)
	p.cubic(cx+k*rx, cy-ry, cx+rx, cy-k*ry, cx+rx, cy)
	p.cubic(cx+rx, cy+k*ry, cx+k*rx, cy+ry, cx, cy+ry)
	p.cubic(cx-k*rx, cy+ry, cx-rx, cy+k*ry, cx-rx, cy)
	p.cubic(cx-rx, cy-k*ry, cx-k*rx, cy-ry, cx, cy-ry)
	p.close()
	return p
}

func PolygonPath(cx, cy, radius float64, sides int, star float64, rot float64) *Path {
	p := &Path{}
	total := sides * 2
	for i := 0; i < total; i++ {
		rr := radius
		if i%2 == 1 {
			rr = radius * star
		}
		a := rot + float64(i)*math.Pi/float64(sides)
		x := cx + rr*math.Cos(a)
		y := cy + rr*math.Sin(a)
		if i == 0 {
			p.move(x, y)
		} else {
			p.line(x, y)
		}
	}
	p.close()
	return p
}

func PolylinePath(pts ...float64) *Path {
	p := &Path{}
	for i := 0; i+1 < len(pts); i += 2 {
		if i == 0 {
			p.move(pts[i], pts[i+1])
		} else {
			p.line(pts[i], pts[i+1])
		}
	}
	return p
}
