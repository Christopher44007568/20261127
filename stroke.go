package stroke

import "math"

// JoinStyle controls how two consecutive segments meet.
type JoinStyle uint8

const (
	JoinMiter JoinStyle = iota
	JoinRound
	JoinBevel
)

// CapStyle controls the ends of open subpaths and dashed segments.
type CapStyle uint8

const (
	CapButt CapStyle = iota
	CapRound
	CapSquare
)

// Style describes a stroke. Width is in local user space. A non-positive
// width produces an empty mesh.
type Style struct {
	Width      float64
	Join       JoinStyle
	Cap        CapStyle
	MiterLimit float64

	// Dashes alternates on/off lengths. An odd-length pattern is repeated
	// once to make it even. DashOffset advances along the pattern at the
	// start of every subpath.
	Dashes     []float64
	DashOffset float64

	// ArcTolerance is the maximum chord error for round joins/caps. Values
	// less than or equal to zero use 0.05 user-space units.
	ArcTolerance float64
}

const defaultMiterLimit = 4.0

// Stroke tessellates path into filled triangles.
func Stroke(path Path, style Style) Mesh {
	if style.Width <= 0 || len(path.Subpaths) == 0 {
		return Mesh{}
	}
	style = normalizeStyle(style)
	var mesh Mesh
	for _, sp := range path.Subpaths {
		strokeSubpath(&mesh, sp, style)
	}
	return mesh
}

// StrokeTransformed applies a and a declared width policy. WidthFollowTransform
// is the geometrically accurate anisotropic result: local stroke geometry is
// tessellated first and then transformed. Other policies transform the
// centerline first and use one scalar device-space width.
func StrokeTransformed(path Path, style Style, a Affine, policy WidthPolicy) Mesh {
	if style.Width <= 0 {
		return Mesh{}
	}
	if policy == WidthFollowTransform {
		return a.ApplyMesh(Stroke(path, style))
	}
	scale := a.ScaleWidth(1, policy)
	style.Width *= scale
	style.DashOffset *= scale
	if len(style.Dashes) > 0 {
		style.Dashes = append([]float64(nil), style.Dashes...)
		for i := range style.Dashes {
			style.Dashes[i] *= scale
		}
	}
	if style.ArcTolerance > 0 {
		style.ArcTolerance *= scale
	}
	return Stroke(a.ApplyPath(path), style)
}

func normalizeStyle(style Style) Style {
	if style.MiterLimit <= 0 {
		style.MiterLimit = defaultMiterLimit
	}
	if style.MiterLimit < 1 {
		style.MiterLimit = 1
	}
	if style.Join > JoinBevel {
		style.Join = JoinMiter
	}
	if style.Cap > CapSquare {
		style.Cap = CapButt
	}
	if style.ArcTolerance <= 0 {
		style.ArcTolerance = 0.05
	}
	return style
}

func strokeSubpath(mesh *Mesh, sp Subpath, style Style) {
	pts, closed := cleanSubpath(sp)
	if len(pts) == 0 {
		return
	}
	if len(pts) == 1 {
		if styleAllowsPoint(style) {
			addPointMarker(mesh, pts[0], style.Width/2, style.Cap, style)
		}
		return
	}
	if !hasDash(style.Dashes) {
		strokeRuns(mesh, splitAtCusps(pts, closed, style), style)
		return
	}
	pattern := expandDashPattern(style.Dashes)
	if len(pattern) == 0 {
		strokeRuns(mesh, splitAtCusps(pts, closed, style), style)
		return
	}
	pieces := dashPolyline(pts, closed, pattern, style.DashOffset)
	for _, piece := range pieces {
		if piece.closed {
			strokeRuns(mesh, splitAtCusps(piece.pts, true, style), style)
			continue
		}
		pts := pruneTerminalStubs(piece.pts, style.Width/2)
		if len(pts) == 1 {
			addPointMarker(mesh, pts[0], style.Width/2, style.Cap, style)
			continue
		}
		run := run{pts: pts, startCap: style.Cap, endCap: style.Cap}
		strokeRuns(mesh, splitRunAtCusps(run, cuspCap(style.Join)), style)
	}
}

// pruneTerminalStubs removes a dash endpoint that sits so close to a join that
// its cap would overlap the join fan. Starting or ending at the vertex instead
// keeps the union shape while avoiding duplicate triangle coverage.
func pruneTerminalStubs(pts []Point, radius float64) []Point {
	if radius <= 0 || len(pts) <= 2 {
		return pts
	}
	for len(pts) > 2 {
		din := direction(pts[0], pts[1])
		dout := direction(pts[1], pts[2])
		if pointDistance(pts[0], pts[1]) >= joinInnerReach(din, dout, radius) {
			break
		}
		pts = pts[1:]
	}
	for len(pts) > 2 {
		n := len(pts)
		din := direction(pts[n-3], pts[n-2])
		dout := direction(pts[n-2], pts[n-1])
		if pointDistance(pts[n-2], pts[n-1]) >= joinInnerReach(din, dout, radius) {
			break
		}
		pts = pts[:len(pts)-1]
	}
	return pts
}

func joinInnerReach(din, dout Point, radius float64) float64 {
	dotv := math.Max(-1, math.Min(1, dot(din, dout)))
	if math.Abs(cross(din, dout)) <= 1e-12 && dotv > 0 {
		return radius
	}
	if dotv <= -0.7071067811865476 {
		return 3 * radius
	}
	phi := math.Acos(dotv)
	return radius / math.Cos(phi/2)
}

type run struct {
	pts      []Point
	closed   bool
	startCap CapStyle
	endCap   CapStyle
}

func strokeRuns(mesh *Mesh, runs []run, style Style) {
	for _, r := range runs {
		if len(r.pts) == 0 {
			continue
		}
		if len(r.pts) == 1 {
			addPointMarker(mesh, r.pts[0], style.Width/2, r.startCap, style)
			continue
		}
		strokePolyline(mesh, r, style)
	}
}

func cleanSubpath(sp Subpath) ([]Point, bool) {
	if len(sp.Points) == 0 {
		return nil, sp.Closed
	}
	pts := make([]Point, 0, len(sp.Points))
	for _, p := range sp.Points {
		if len(pts) == 0 || !pointsEqual(pts[len(pts)-1], p) {
			pts = append(pts, p)
		}
	}
	if sp.Closed && len(pts) > 1 && pointsEqual(pts[0], pts[len(pts)-1]) {
		pts = pts[:len(pts)-1]
	}
	return pts, sp.Closed
}

func splitAtCusps(pts []Point, closed bool, style Style) []run {
	if !closed {
		base := run{
			pts:      pts,
			startCap: style.Cap,
			endCap:   style.Cap,
		}
		return splitRunAtCusps(base, cuspCap(style.Join))
	}
	cusps := cuspIndices(pts, true)
	if len(cusps) == 0 {
		return []run{{pts: pts, closed: true}}
	}
	caps := cuspCap(style.Join)
	var runs []run
	for k, start := range cusps {
		end := cusps[0]
		if k+1 < len(cusps) {
			end = cusps[k+1]
		}
		steps := (end - start + len(pts)) % len(pts)
		if steps == 0 {
			steps = len(pts)
		}
		part := make([]Point, 0, steps+1)
		for i := 0; i <= steps; i++ {
			part = append(part, pts[(start+i)%len(pts)])
		}
		if len(part) >= 2 {
			runs = append(runs, run{
				pts:      part,
				startCap: caps,
				endCap:   caps,
			})
		}
	}
	return runs
}

func splitRunAtCusps(base run, capAtCusp CapStyle) []run {
	if base.closed || len(base.pts) < 3 {
		return []run{base}
	}
	cusps := cuspIndices(base.pts, false)
	if len(cusps) == 0 {
		return []run{base}
	}
	runs := make([]run, 0, len(cusps)+1)
	start := 0
	startCap := base.startCap
	for _, cusp := range cusps {
		runs = append(runs, run{
			pts:      append([]Point(nil), base.pts[start:cusp+1]...),
			startCap: startCap,
			endCap:   capAtCusp,
		})
		start = cusp
		startCap = capAtCusp
	}
	runs = append(runs, run{
		pts:      append([]Point(nil), base.pts[start:]...),
		startCap: startCap,
		endCap:   base.endCap,
	})
	return runs
}

func cuspIndices(pts []Point, closed bool) []int {
	n := len(pts)
	if n < 2 {
		return nil
	}
	var out []int
	first, last := 1, n-1
	if closed {
		first, last = 0, n-1
	}
	for i := first; i <= last; i++ {
		if !closed && (i == 0 || i == n-1) {
			continue
		}
		din := direction(pts[(i-1+n)%n], pts[i])
		dout := direction(pts[i], pts[(i+1)%n])
		if dot(din, dout) <= -0.7071067811865476 {
			out = append(out, i)
		}
	}
	return out
}

func cuspCap(join JoinStyle) CapStyle {
	if join == JoinRound {
		return CapRound
	}
	return CapButt
}

func strokePolyline(mesh *Mesh, r run, style Style) {
	pts := r.pts
	n := len(pts)
	radius := style.Width / 2
	sides := make([]sideSet, n)

	if !r.closed {
		d0 := direction(pts[0], pts[1])
		n0 := leftNormal(d0)
		sides[0].leftOut = add(pts[0], scale(n0, radius))
		sides[0].rightOut = add(pts[0], scale(n0, -radius))
		d1 := direction(pts[n-2], pts[n-1])
		n1 := leftNormal(d1)
		sides[n-1].leftIn = add(pts[n-1], scale(n1, radius))
		sides[n-1].rightIn = add(pts[n-1], scale(n1, -radius))
	}

	first, last := 1, n-1
	if r.closed {
		first, last = 0, n-1
	}
	for i := first; i <= last; i++ {
		if !r.closed && (i == 0 || i == n-1) {
			continue
		}
		din := direction(pts[(i-1+n)%n], pts[i])
		dout := direction(pts[i], pts[(i+1)%n])
		ss, fill := joinSides(pts[i], din, dout, radius, style)
		sides[i] = ss
		if len(fill) >= 3 {
			addFan(mesh, fill)
		}
	}

	segments := n - 1
	if r.closed {
		segments = n
	}
	for i := 0; i < segments; i++ {
		j := (i + 1) % n
		mesh.AppendQuad(
			sides[i].leftOut,
			sides[j].leftIn,
			sides[j].rightIn,
			sides[i].rightOut,
		)
	}

	if !r.closed {
		startDir := direction(pts[0], pts[1])
		addCap(mesh, pts[0], sides[0].leftOut, sides[0].rightOut,
			scale(startDir, -1), radius, r.startCap, style)
		endDir := direction(pts[n-2], pts[n-1])
		addCap(mesh, pts[n-1], sides[n-1].leftIn, sides[n-1].rightIn,
			endDir, radius, r.endCap, style)
	}
}

func addPointMarker(mesh *Mesh, p Point, radius float64, cap CapStyle, style Style) {
	switch cap {
	case CapRound:
		segs := arcSegments(radius, 2*math.Pi, style.ArcTolerance)
		ring := make([]Point, segs+1)
		for i := 0; i <= segs; i++ {
			a := 2 * math.Pi * float64(i) / float64(segs)
			ring[i] = add(p, Point{X: radius * math.Cos(a), Y: radius * math.Sin(a)})
		}
		for i := 0; i < segs; i++ {
			mesh.Append(Triangle{A: p, B: ring[i], C: ring[i+1]})
		}
	case CapSquare:
		a := Point{X: p.X - radius, Y: p.Y - radius}
		b := Point{X: p.X + radius, Y: p.Y - radius}
		c := Point{X: p.X + radius, Y: p.Y + radius}
		d := Point{X: p.X - radius, Y: p.Y + radius}
		mesh.AppendQuad(a, b, c, d)
	}
}

func addCap(mesh *Mesh, p, left, right, outward Point, radius float64, cap CapStyle, style Style) {
	switch cap {
	case CapButt:
		return
	case CapSquare:
		ext := scale(outward, radius)
		mesh.AppendQuad(left, right, add(right, ext), add(left, ext))
	case CapRound:
		sweep := -math.Pi
		if cross(sub(left, p), outward) > 0 {
			sweep = math.Pi
		}
		poly := arcPolygon(p, left, right, radius, sweep, style.ArcTolerance)
		addFan(mesh, poly)
	}
}

func addFan(mesh *Mesh, poly []Point) {
	if len(poly) < 3 {
		return
	}
	for i := 1; i+1 < len(poly); i++ {
		mesh.Append(Triangle{A: poly[0], B: poly[i], C: poly[i+1]})
	}
}

func pointsEqual(a, b Point) bool {
	return math.Abs(a.X-b.X) <= 1e-12 && math.Abs(a.Y-b.Y) <= 1e-12
}

func direction(a, b Point) Point {
	dx, dy := b.X-a.X, b.Y-a.Y
	l := math.Hypot(dx, dy)
	if l == 0 {
		return Point{}
	}
	return Point{X: dx / l, Y: dy / l}
}

func leftNormal(d Point) Point {
	return Point{X: -d.Y, Y: d.X}
}

func add(a, b Point) Point {
	return Point{X: a.X + b.X, Y: a.Y + b.Y}
}

func sub(a, b Point) Point {
	return Point{X: a.X - b.X, Y: a.Y - b.Y}
}

func scale(p Point, s float64) Point {
	return Point{X: p.X * s, Y: p.Y * s}
}

func dot(a, b Point) float64 {
	return a.X*b.X + a.Y*b.Y
}

func cross(a, b Point) float64 {
	return a.X*b.Y - a.Y*b.X
}

func floorInt(v float64) int {
	return int(math.Floor(v))
}

func ceilInt(v float64) int {
	return int(math.Ceil(v))
}
