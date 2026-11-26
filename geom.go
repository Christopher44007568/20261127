package main

import "math"

// Pt is a 2D point in an arbitrary pixel-oriented coordinate space (y down).
type Pt struct {
	X, Y float64
}

func (p Pt) add(q Pt) Pt      { return Pt{p.X + q.X, p.Y + q.Y} }
func (p Pt) sub(q Pt) Pt      { return Pt{p.X - q.X, p.Y - q.Y} }
func (p Pt) mul(k float64) Pt { return Pt{p.X * k, p.Y * k} }

func (p Pt) dist(q Pt) float64 { return math.Hypot(p.X-q.X, p.Y-q.Y) }

// Contour is a flattened closed polyline.
type Contour struct {
	Pts    []Pt
	Closed bool
}

// Outline is a set of contours in one shared coordinate space.
type Outline struct {
	Contours []Contour
	Bounds   Rect
}

// GlyphOutline converts raw font-unit contours to pixel-oriented local space:
// x = fontX*scale, y = -fontY*scale (baseline at y=0, y grows downward).
func GlyphOutline(g *Glyph, unitsPerEm int, ppem, tol float64) Outline {
	s := ppem / float64(unitsPerEm)
	o := Outline{}
	for _, rc := range g.Contours {
		if len(rc) == 0 {
			continue
		}
		mapped := make([]rawP, len(rc))
		for i, p := range rc {
			mapped[i] = rawP{x: float64(p.X) * s, y: -float64(p.Y) * s, on: p.OnCur}
		}
		poly := flattenTTContour(mapped, tol)
		if len(poly) >= 2 {
			o.Contours = append(o.Contours, Contour{Pts: poly, Closed: true})
		}
	}
	o.Bounds = Rect{
		MinX: g.Bounds.MinX * s,
		MinY: -g.Bounds.MaxY * s,
		MaxX: g.Bounds.MaxX * s,
		MaxY: -g.Bounds.MinY * s,
	}
	return o
}

type rawP struct {
	x, y float64
	on   bool
}

func midP(a, b rawP) rawP { return rawP{(a.x + b.x) / 2, (a.y + b.y) / 2, true} }

// flattenTTContour turns TrueType on/off-curve points into a line polyline,
// expanding implied on-curve midpoints between consecutive off-curve points.
func flattenTTContour(src []rawP, tol float64) []Pt {
	n := len(src)
	if n == 0 {
		return nil
	}
	allOff := true
	for _, p := range src {
		if p.on {
			allOff = false
			break
		}
	}
	var start rawP
	var seq []rawP
	if allOff {
		// Implied on-curve midpoints all around.
		out := []Pt{{(src[n-1].x + src[0].x) / 2, (src[n-1].y + src[0].y) / 2}}
		for i := 0; i < n; i++ {
			ctrl := Pt{src[i].x, src[i].y}
			next := src[(i+1)%n]
			end := Pt{(src[i].x + next.x) / 2, (src[i].y + next.y) / 2}
			out = appendQuad(out, out[len(out)-1], ctrl, end, tol)
		}
		return out
	}
	// Rotate so the first point is on-curve.
	firstOn := 0
	for i, p := range src {
		if p.on {
			firstOn = i
			break
		}
	}
	seq = make([]rawP, n)
	for i := 0; i < n; i++ {
		seq[i] = src[(firstOn+i)%n]
	}
	start = seq[0]
	out := []Pt{{start.x, start.y}}
	i := 1
	for i < n {
		p := seq[i]
		if p.on {
			out = append(out, Pt{p.x, p.y})
			i++
			continue
		}
		// p is an off-curve control point.
		var end rawP
		next := seq[(i+1)%n]
		skip := 1
		if next.on {
			end = next
			skip = 2
		} else {
			end = midP(p, next)
		}
		c0 := out[len(out)-1]
		out = appendQuad(out, c0, Pt{p.x, p.y}, Pt{end.x, end.y}, tol)
		i += skip
	}
	// Close back to the starting on-curve point.
	last := out[len(out)-1]
	if last.dist(Pt{start.x, start.y}) > tol*0.5 {
		out = append(out, Pt{start.x, start.y})
	}
	return out
}

// appendQuad appends a flattened quadratic Bezier; the first point already exists.
func appendQuad(out []Pt, p0, p1, p2 Pt, tol float64) []Pt {
	// Flatness: distance of control point from chord, in px.
	chord := p2.sub(p0)
	length := math.Hypot(chord.X, chord.Y)
	var d float64
	if length < 1e-9 {
		d = p1.dist(p0)
	} else {
		// Perpendicular distance of p1 to line p0-p2.
		d = math.Abs(chord.X*(p0.Y-p1.Y)-chord.Y*(p0.X-p1.X)) / length
	}
	if d <= tol {
		return append(out, p2)
	}
	p01 := Pt{(p0.X + p1.X) / 2, (p0.Y + p1.Y) / 2}
	p12 := Pt{(p1.X + p2.X) / 2, (p1.Y + p2.Y) / 2}
	p012 := Pt{(p01.X + p12.X) / 2, (p01.Y + p12.Y) / 2}
	out = appendQuad(out, p0, p01, p012, tol)
	out = appendQuad(out, p012, p12, p2, tol)
	return out
}

// Translate returns a translated copy of the outline.
func (o Outline) Translate(dx, dy float64) Outline {
	r := Outline{Bounds: Rect{o.Bounds.MinX + dx, o.Bounds.MinY + dy, o.Bounds.MaxX + dx, o.Bounds.MaxY + dy}}
	r.Contours = make([]Contour, len(o.Contours))
	for i, c := range o.Contours {
		nc := Contour{Closed: c.Closed, Pts: make([]Pt, len(c.Pts))}
		for j, p := range c.Pts {
			nc.Pts[j] = Pt{p.X + dx, p.Y + dy}
		}
		r.Contours[i] = nc
	}
	return r
}

// PolylineLength sums segment lengths of a (possibly closed) contour.
func (c Contour) Length() float64 {
	total := 0.0
	for i := 1; i < len(c.Pts); i++ {
		total += c.Pts[i-1].dist(c.Pts[i])
	}
	if c.Closed && len(c.Pts) > 2 {
		// Last point usually equals the first; guard anyway.
		if c.Pts[len(c.Pts)-1] != c.Pts[0] {
			total += c.Pts[len(c.Pts)-1].dist(c.Pts[0])
		}
	}
	return total
}

// ArcSample is one evenly-spaced point on a contour.
type ArcSample struct {
	Pos     Pt
	Tangent Pt // unit tangent in travel direction
	Normal  Pt // unit normal (rotated +90deg in y-down space)
	Dist    float64
}

// SampleArcLength places n points at equal arc-length intervals around a
// closed contour, starting at arc offset 0.
func (c Contour) SampleArcLength(n int) []ArcSample {
	if n <= 0 || len(c.Pts) < 2 {
		return nil
	}
	// Build segment table including closing segment.
	type seg struct {
		a, b Pt
		l    float64
		acc  float64
	}
	var segs []seg
	acc := 0.0
	add := func(a, b Pt) {
		l := a.dist(b)
		if l < 1e-9 {
			return
		}
		acc += l
		segs = append(segs, seg{a: a, b: b, l: l, acc: acc})
	}
	for i := 1; i < len(c.Pts); i++ {
		add(c.Pts[i-1], c.Pts[i])
	}
	if c.Closed {
		add(c.Pts[len(c.Pts)-1], c.Pts[0])
	}
	total := acc
	if total < 1e-9 || len(segs) == 0 {
		return nil
	}
	out := make([]ArcSample, 0, n)
	si := 0
	for k := 0; k < n; k++ {
		target := total * float64(k) / float64(n)
		for si < len(segs)-1 && segs[si].acc < target {
			si++
		}
		s := segs[si]
		base := s.acc - s.l
		t := 0.0
		if s.l > 1e-9 {
			t = (target - base) / s.l
		}
		pos := Pt{s.a.X + (s.b.X-s.a.X)*t, s.a.Y + (s.b.Y-s.a.Y)*t}
		tan := Pt{(s.b.X - s.a.X) / s.l, (s.b.Y - s.a.Y) / s.l}
		out = append(out, ArcSample{
			Pos:     pos,
			Tangent: tan,
			Normal:  Pt{-tan.Y, tan.X},
			Dist:    target,
		})
	}
	return out
}

// SampleSpacing chooses a point count from a target arc spacing (px).
func (c Contour) SampleSpacing(spacing float64) []ArcSample {
	if spacing <= 0 {
		return nil
	}
	n := int(math.Round(c.Length() / spacing))
	if n < 3 {
		n = 3
	}
	return c.SampleArcLength(n)
}

// Fit is a uniform scale+translate mapping a source rect into a destination.
type Fit struct {
	K      float64
	Tx, Ty float64
}

// FitRect returns a uniform fit centered in dst.
func FitRect(src Rect, dst Rect) Fit {
	k := math.Min(dst.Width()/src.Width(), dst.Height()/src.Height())
	scx := (src.MinX + src.MaxX) / 2
	scy := (src.MinY + src.MaxY) / 2
	dcx := (dst.MinX + dst.MaxX) / 2
	dcy := (dst.MinY + dst.MaxY) / 2
	return Fit{K: k, Tx: dcx - k*scx, Ty: dcy - k*scy}
}

func (ft Fit) P(p Pt) Pt { return Pt{p.X*ft.K + ft.Tx, p.Y*ft.K + ft.Ty} }

func (ft Fit) Rect(r Rect) Rect {
	a := ft.P(Pt{r.MinX, r.MinY})
	b := ft.P(Pt{r.MaxX, r.MaxY})
	return Rect{MinX: math.Min(a.X, b.X), MinY: math.Min(a.Y, b.Y),
		MaxX: math.Max(a.X, b.X), MaxY: math.Max(a.Y, b.Y)}
}

func (ft Fit) Outline(o Outline) Outline {
	out := Outline{Bounds: ft.Rect(o.Bounds)}
	out.Contours = make([]Contour, len(o.Contours))
	for i, c := range o.Contours {
		nc := Contour{Closed: c.Closed, Pts: make([]Pt, len(c.Pts))}
		for j, p := range c.Pts {
			nc.Pts[j] = ft.P(p)
		}
		out.Contours[i] = nc
	}
	return out
}

// RawPixelContours maps raw font-unit points into y-down pixel-local space
// at ppem (baseline y=0).
func RawPixelContours(g *Glyph, unitsPerEm int, ppem float64) [][]Pt {
	s := ppem / float64(unitsPerEm)
	out := make([][]Pt, len(g.Contours))
	for i, rc := range g.Contours {
		out[i] = make([]Pt, len(rc))
		for j, p := range rc {
			out[i][j] = Pt{float64(p.X) * s, -float64(p.Y) * s}
		}
	}
	return out
}
