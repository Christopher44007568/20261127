package bezierstroke

import "math"

// JoinStyle controls how two flattened edges meet.
type JoinStyle int

const (
	JoinRound JoinStyle = iota
	JoinMiter
	JoinBevel
)

// CapStyle controls the ends of an open stroke.
type CapStyle int

const (
	CapButt CapStyle = iota
	CapRound
	CapSquare
)

// StrokeStyle is expressed in the coordinate space of the polyline.
type StrokeStyle struct {
	Width      float64
	Join       JoinStyle
	Cap        CapStyle
	Tolerance  float64
	MiterLimit float64
}

// Polygon is a filled, simple polygon. All polygons emitted by StrokePolyline
// have the same winding direction so a nonzero coverage fill implements their
// union without cancellation at overlaps.
type Polygon []Point

type StrokeSegment struct {
	A, B          Point
	LeftA, RightA Point
	LeftB, RightB Point
	Polygon       Polygon
}

type StrokeJoin struct {
	Center  Point
	Radius  float64
	Polygon Polygon
}

type StrokeCap struct {
	Center  Point
	Polygon Polygon
}

type StrokeMesh struct {
	Centerline []Point
	Segments   []StrokeSegment
	Joins      []StrokeJoin
	Caps       []StrokeCap
	Polygons   []Polygon
}

// StrokePolyline turns a centerline into a union-ready stroke mesh. Segment
// width is independent of segment length; joins and caps are tessellated at
// the supplied chord tolerance rather than with a fixed number of steps.
func StrokePolyline(points []Point, style StrokeStyle) StrokeMesh {
	points = dedupePoints(points, 1e-10)
	width := style.Width
	if !(width > 0) {
		return StrokeMesh{Centerline: points}
	}
	tol := style.Tolerance
	if !(tol > 0) || math.IsInf(tol, 0) {
		tol = 0.1
	}
	if style.MiterLimit <= 0 {
		style.MiterLimit = 4
	}

	mesh := StrokeMesh{Centerline: points}
	half := width * 0.5
	for i := 0; i+1 < len(points); i++ {
		a, b := points[i], points[i+1]
		dir := b.Sub(a).Unit()
		if dir == (Point{}) {
			continue
		}
		n := dir.Perp().Mul(half)
		seg := StrokeSegment{
			A:      a,
			B:      b,
			LeftA:  a.Add(n),
			RightA: a.Sub(n),
			LeftB:  b.Add(n),
			RightB: b.Sub(n),
		}
		seg.Polygon = ccwPolygon([]Point{seg.LeftA, seg.LeftB, seg.RightB, seg.RightA})
		mesh.Segments = append(mesh.Segments, seg)
		mesh.Polygons = append(mesh.Polygons, seg.Polygon)
	}

	if len(mesh.Segments) == 0 {
		return mesh
	}
	for i := 1; i+1 < len(points); i++ {
		prevDir := points[i].Sub(points[i-1]).Unit()
		nextDir := points[i+1].Sub(points[i]).Unit()
		if prevDir == (Point{}) || nextDir == (Point{}) {
			continue
		}
		turn := math.Atan2(math.Abs(prevDir.Cross(nextDir)),
			prevDir.Dot(nextDir))
		if turn <= 1e-9 {
			continue
		}
		poly := joinPolygon(points[i], prevDir, nextDir, half, turn, style, tol)
		if len(poly) < 3 {
			continue
		}
		join := StrokeJoin{Center: points[i], Radius: half, Polygon: poly}
		mesh.Joins = append(mesh.Joins, join)
		mesh.Polygons = append(mesh.Polygons, join.Polygon)
	}

	addCaps(&mesh, style, tol)
	return mesh
}

func addCaps(mesh *StrokeMesh, style StrokeStyle, tol float64) {
	if style.Cap == CapButt {
		return
	}
	pts := mesh.Centerline
	start, end := pts[0], pts[len(pts)-1]
	r := style.Width * 0.5
	if style.Cap == CapRound {
		for _, center := range []Point{start, end} {
			poly := circlePolygon(center, r, tol)
			mesh.Caps = append(mesh.Caps, StrokeCap{Center: center, Polygon: poly})
			mesh.Polygons = append(mesh.Polygons, poly)
		}
		return
	}

	startDir := pts[0].Sub(pts[1]).Unit()
	endDir := pts[len(pts)-1].Sub(pts[len(pts)-2]).Unit()
	for _, cap := range []struct {
		center Point
		dir    Point
	}{{start, startDir}, {end, endDir}} {
		ext := cap.dir.Mul(r)
		n := cap.dir.Perp().Mul(r)
		poly := ccwPolygon([]Point{
			cap.center.Add(n),
			cap.center.Add(n).Add(ext),
			cap.center.Sub(n).Add(ext),
			cap.center.Sub(n),
		})
		mesh.Caps = append(mesh.Caps, StrokeCap{Center: cap.center, Polygon: poly})
		mesh.Polygons = append(mesh.Polygons, poly)
	}
}

func joinPolygon(c Point, prevDir, nextDir Point, r, turn float64,
	style StrokeStyle, tol float64) Polygon {
	// The outside of a turn is the side away from its center.
	outer := prevDir.Perp()
	sweep := turn
	if prevDir.Cross(nextDir) > 0 {
		outer = outer.Mul(-1)
	} else {
		sweep = -turn
	}
	nextOuter := nextDir.Perp()
	if prevDir.Cross(nextDir) > 0 {
		nextOuter = nextOuter.Mul(-1)
	}

	switch style.Join {
	case JoinMiter:
		ratio := 1 / math.Cos(turn*0.5)
		if turn < math.Pi-1e-9 && ratio <= style.MiterLimit {
			if m, ok := lineIntersection(
				c.Add(outer.Mul(r)), prevDir,
				c.Add(nextOuter.Mul(r)), nextDir,
			); ok {
				return ccwPolygon([]Point{
					c,
					c.Add(outer.Mul(r)),
					m,
					c.Add(nextOuter.Mul(r)),
				})
			}
		}
		fallthrough
	case JoinBevel:
		return ccwPolygon([]Point{
			c,
			c.Add(outer.Mul(r)),
			c.Add(nextOuter.Mul(r)),
		})
	default:
		start := math.Atan2(outer.Y, outer.X)
		arc := arcPoints(c, r, start, sweep, tol)
		poly := append([]Point{c}, arc...)
		return ccwPolygon(poly)
	}
}

func arcPoints(center Point, r, start, sweep, tol float64) []Point {
	if math.Abs(sweep) <= 1e-12 {
		return []Point{center.Add(Point{math.Cos(start), math.Sin(start)}.Mul(r))}
	}
	if tol >= r {
		tol = r * 0.999
	}
	maxAngle := 2 * math.Acos(1-tol/r)
	n := int(math.Ceil(math.Abs(sweep) / maxAngle))
	if n < 1 {
		n = 1
	}
	out := make([]Point, 0, n+1)
	for i := 0; i <= n; i++ {
		a := start + sweep*float64(i)/float64(n)
		out = append(out, center.Add(Point{math.Cos(a), math.Sin(a)}.Mul(r)))
	}
	return out
}

func circlePolygon(center Point, r, tol float64) Polygon {
	return ccwPolygon(arcPoints(center, r, 0, 2*math.Pi, tol))
}

func lineIntersection(p, d, q, e Point) (Point, bool) {
	den := d.Cross(e)
	if math.Abs(den) <= 1e-12 {
		return Point{}, false
	}
	t := q.Sub(p).Cross(e) / den
	return p.Add(d.Mul(t)), true
}

func ccwPolygon(p []Point) Polygon {
	area := signedArea(p)
	if area < 0 {
		for i, j := 0, len(p)-1; i < j; i, j = i+1, j-1 {
			p[i], p[j] = p[j], p[i]
		}
	}
	return Polygon(p)
}

func signedArea(p []Point) float64 {
	area := 0.0
	for i := range p {
		j := (i + 1) % len(p)
		area += p[i].X*p[j].Y - p[j].X*p[i].Y
	}
	return area * 0.5
}

func dedupePoints(p []Point, eps float64) []Point {
	if len(p) == 0 {
		return nil
	}
	out := make([]Point, 0, len(p))
	for _, q := range p {
		if len(out) == 0 || q.Distance(out[len(out)-1]) > eps {
			out = append(out, q)
		}
	}
	return out
}
