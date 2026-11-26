// Package mesh defines the triangle mesh container and procedural
// acceptance models (closed hole, open sharp edge, tangent surfaces).
package mesh

import (
	"math"

	"featurelines/internal/linalg"
)

// Vec3 is re-exported so callers can use mesh.Vec3 as a short alias.
type Vec3 = linalg.Vec3

// Mesh is an indexed triangle mesh.
type Mesh struct {
	Name string
	V    []Vec3
	T    [][3]int

	// FN are per-triangle geometric normals (unit length).
	FN []Vec3
	// VN are area-weighted vertex normals (unit length).
	VN []Vec3
}

// New creates an empty named mesh.
func New(name string) *Mesh {
	return &Mesh{Name: name}
}

// AddVertex appends a vertex and returns its index.
func (m *Mesh) AddVertex(p Vec3) int {
	m.V = append(m.V, p)
	return len(m.V) - 1
}

// AddTri appends a triangle.
func (m *Mesh) AddTri(a, b, c int) {
	m.T = append(m.T, [3]int{a, b, c})
}

// AddTriOriented appends (a,b,c) after swapping b and c when its geometric
// normal points opposite to want. Procedural builders use this to guarantee
// consistent outward winding without hand-deriving every quad order.
func (m *Mesh) AddTriOriented(a, b, c int, want Vec3) {
	n := m.faceNormal(a, b, c)
	if n.Dot(want) < 0 {
		b, c = c, b
	}
	m.AddTri(a, b, c)
}

func (m *Mesh) faceNormal(a, b, c int) Vec3 {
	e1 := m.V[b].Sub(m.V[a])
	e2 := m.V[c].Sub(m.V[a])
	return e1.Cross(e2)
}

// ComputeNormals rebuilds face and area-weighted vertex normals.
func (m *Mesh) ComputeNormals() {
	m.FN = make([]Vec3, len(m.T))
	m.VN = make([]Vec3, len(m.V))
	for ti, t := range m.T {
		n := m.faceNormal(t[0], t[1], t[2])
		m.FN[ti] = n.Normalized()
		// Cross product length equals twice the triangle area, so adding
		// the raw normal weights each corner by triangle area.
		m.VN[t[0]] = m.VN[t[0]].Add(n)
		m.VN[t[1]] = m.VN[t[1]].Add(n)
		m.VN[t[2]] = m.VN[t[2]].Add(n)
	}
	for i := range m.VN {
		m.VN[i] = m.VN[i].Normalized()
	}
}

// Bounds returns the axis-aligned bounding box.
func (m *Mesh) Bounds() (min, max Vec3) {
	if len(m.V) == 0 {
		return Vec3{}, Vec3{}
	}
	min = m.V[0]
	max = m.V[0]
	for _, p := range m.V[1:] {
		min = Vec3{X: math.Min(min.X, p.X), Y: math.Min(min.Y, p.Y), Z: math.Min(min.Z, p.Z)}
		max = Vec3{X: math.Max(max.X, p.X), Y: math.Max(max.Y, p.Y), Z: math.Max(max.Z, p.Z)}
	}
	return min, max
}

// CenterAndSize returns bounding-box center and largest axis extent.
func (m *Mesh) CenterAndSize() (center Vec3, size float64) {
	lo, hi := m.Bounds()
	center = lo.Add(hi).Scale(0.5)
	d := hi.Sub(lo)
	size = math.Max(d.X, math.Max(d.Y, d.Z))
	return center, size
}
