package stroke

import "math"

// Point is a point in user space.
type Point struct {
	X float64
	Y float64
}

// Subpath is one move-to/line-to sequence. Closed subpaths contain the
// implicit closing segment from the last point back to the first point.
type Subpath struct {
	Points []Point
	Closed bool
}

// Path is a collection of subpaths. Dash splitting never joins separate
// subpaths.
type Path struct {
	Subpaths []Subpath
	current  int
}

// NewPath returns an empty path with a path builder cursor.
func NewPath() *Path {
	return &Path{current: -1}
}

// MoveTo starts a new open subpath.
func (p *Path) MoveTo(pt Point) {
	p.Subpaths = append(p.Subpaths, Subpath{Points: []Point{pt}})
	p.current = len(p.Subpaths) - 1
}

// LineTo appends a point to the current subpath. If no subpath is active it
// behaves like MoveTo.
func (p *Path) LineTo(pt Point) {
	if p.current < 0 || p.current >= len(p.Subpaths) {
		p.MoveTo(pt)
		return
	}
	p.Subpaths[p.current].Points = append(p.Subpaths[p.current].Points, pt)
}

// Close closes the current subpath.
func (p *Path) Close() {
	if p.current >= 0 && p.current < len(p.Subpaths) {
		p.Subpaths[p.current].Closed = true
	}
	p.current = -1
}

// Append adds a subpath and leaves the builder cursor inactive.
func (p *Path) Append(sp Subpath) {
	p.Subpaths = append(p.Subpaths, sp)
	p.current = -1
}

// Clone returns a deep copy of the path.
func (p Path) Clone() Path {
	out := Path{Subpaths: make([]Subpath, len(p.Subpaths)), current: -1}
	for i, sp := range p.Subpaths {
		out.Subpaths[i] = Subpath{
			Points: append([]Point(nil), sp.Points...),
			Closed: sp.Closed,
		}
	}
	return out
}

// Triangle is a filled triangle. Triangle winding is not significant.
type Triangle struct {
	A Point
	B Point
	C Point
}

// Mesh is a triangle soup. The tessellator emits flat triangles; there are no
// shared indices or curve segments.
type Mesh struct {
	Triangles []Triangle
}

// Append adds a triangle, discarding numerically degenerate triangles.
func (m *Mesh) Append(t Triangle) {
	if math.Abs(signedArea(t.A, t.B, t.C)) > 1e-12 {
		m.Triangles = append(m.Triangles, t)
	}
}

// AppendQuad splits a counter-clockwise-or-clockwise quad into two triangles.
func (m *Mesh) AppendQuad(a, b, c, d Point) {
	m.Append(Triangle{A: a, B: b, C: c})
	m.Append(Triangle{A: a, B: c, C: d})
}

// Bounds returns the tight bounds of all mesh vertices.
func (m Mesh) Bounds() Bounds {
	if len(m.Triangles) == 0 {
		return Bounds{}
	}
	b := Bounds{Min: m.Triangles[0].A, Max: m.Triangles[0].A}
	for _, t := range m.Triangles {
		b = b.extend(t.A)
		b = b.extend(t.B)
		b = b.extend(t.C)
	}
	return b
}

// Bounds is an axis-aligned rectangle.
type Bounds struct {
	Min Point
	Max Point
}

func (b Bounds) empty() bool {
	return b.Min.X > b.Max.X || b.Min.Y > b.Max.Y
}

func (b Bounds) extend(p Point) Bounds {
	if b.empty() {
		return Bounds{Min: p, Max: p}
	}
	b.Min.X = math.Min(b.Min.X, p.X)
	b.Min.Y = math.Min(b.Min.Y, p.Y)
	b.Max.X = math.Max(b.Max.X, p.X)
	b.Max.Y = math.Max(b.Max.Y, p.Y)
	return b
}

func (b Bounds) width() float64 {
	return math.Max(0, b.Max.X-b.Min.X)
}

func (b Bounds) height() float64 {
	return math.Max(0, b.Max.Y-b.Min.Y)
}

// Affine is the 2D affine transform
//
//	x' = A*x + C*y + E
//	y' = B*x + D*y + F
type Affine struct {
	A, B, C, D, E, F float64
}

// Identity is the identity affine transform.
func Identity() Affine {
	return Affine{A: 1, D: 1}
}

// Apply transforms a point.
func (a Affine) Apply(p Point) Point {
	return Point{
		X: a.A*p.X + a.C*p.Y + a.E,
		Y: a.B*p.X + a.D*p.Y + a.F,
	}
}

// ApplyMesh transforms every triangle vertex.
func (a Affine) ApplyMesh(m Mesh) Mesh {
	out := Mesh{Triangles: make([]Triangle, len(m.Triangles))}
	for i, t := range m.Triangles {
		out.Triangles[i] = Triangle{
			A: a.Apply(t.A),
			B: a.Apply(t.B),
			C: a.Apply(t.C),
		}
	}
	return out
}

// ApplyPath transforms every path point.
func (a Affine) ApplyPath(p Path) Path {
	out := p.Clone()
	for i := range out.Subpaths {
		for j := range out.Subpaths[i].Points {
			out.Subpaths[i].Points[j] = a.Apply(out.Subpaths[i].Points[j])
		}
	}
	return out
}

// WidthPolicy declares how a scalar stroke width is mapped through a
// non-uniform affine transform.
type WidthPolicy int

const (
	// WidthFollowTransform strokes in local space and then transforms the
	// mesh. The cross-section becomes an ellipse under non-uniform scaling.
	WidthFollowTransform WidthPolicy = iota
	// WidthByXScale uses the length of the transformed local X axis.
	WidthByXScale
	// WidthByYScale uses the length of the transformed local Y axis.
	WidthByYScale
	// WidthByAverageScale uses the arithmetic mean of the two singular values.
	WidthByAverageScale
	// WidthByGeometricScale uses sqrt(abs(det(A))).
	WidthByGeometricScale
	// WidthByMaxScale uses the larger singular value.
	WidthByMaxScale
	// WidthByMinScale uses the smaller singular value.
	WidthByMinScale
)

// ScaleWidth maps a local-space width to device-space width according to
// policy. WidthFollowTransform cannot be represented by one scalar and uses
// the geometric-mean singular value as its documented fallback.
func (a Affine) ScaleWidth(width float64, policy WidthPolicy) float64 {
	if width == 0 {
		return 0
	}
	x := math.Hypot(a.A, a.B)
	y := math.Hypot(a.C, a.D)
	s1, s2 := a.singularValues()
	switch policy {
	case WidthByXScale:
		return width * x
	case WidthByYScale:
		return width * y
	case WidthByAverageScale:
		return width * (s1 + s2) / 2
	case WidthByMaxScale:
		return width * s1
	case WidthByMinScale:
		return width * s2
	case WidthFollowTransform, WidthByGeometricScale:
		return width * math.Sqrt(math.Abs(a.A*a.D-a.B*a.C))
	default:
		return width * math.Sqrt(math.Abs(a.A*a.D-a.B*a.C))
	}
}

func (a Affine) singularValues() (float64, float64) {
	s11 := a.A*a.A + a.B*a.B
	s22 := a.C*a.C + a.D*a.D
	s12 := a.A*a.C + a.B*a.D
	trace := s11 + s22
	disc := math.Hypot(s11-s22, 2*s12)
	s1 := math.Sqrt(math.Max(0, (trace+disc)/2))
	s2 := math.Sqrt(math.Max(0, (trace-disc)/2))
	return s1, s2
}

// FitTransform returns a uniform transform that maps b into a width by
// height image, leaving padding device pixels on every side.
func FitTransform(b Bounds, width, height int, padding float64) Affine {
	if b.empty() {
		return Identity()
	}
	bw := math.Max(b.width(), 1e-9)
	bh := math.Max(b.height(), 1e-9)
	availW := math.Max(1, float64(width)-2*padding)
	availH := math.Max(1, float64(height)-2*padding)
	scale := math.Min(availW/bw, availH/bh)
	if !isFinite(scale) || scale <= 0 {
		scale = 1
	}
	ox := padding + (availW-bw*scale)/2 - b.Min.X*scale
	oy := padding + (availH-bh*scale)/2 - b.Min.Y*scale
	return Affine{A: scale, D: scale, E: ox, F: oy}
}

func signedArea(a, b, c Point) float64 {
	return (b.X-a.X)*(c.Y-a.Y) - (b.Y-a.Y)*(c.X-a.X)
}

func isFinite(v float64) bool {
	return !math.IsNaN(v) && !math.IsInf(v, 0)
}
