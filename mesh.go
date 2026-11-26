package qem

import (
	"fmt"
	"math"
)

// Vertex is a stable-ID vertex. Inactive vertices are retained so collapse
// records can be reversed without renumbering.
type Vertex struct {
	Pos    Vec3 `json:"position"`
	Active bool `json:"active"`
}

// Triangle uses stable vertex IDs.
type Triangle struct {
	A        int  `json:"a"`
	B        int  `json:"b"`
	C        int  `json:"c"`
	Material int  `json:"material"`
	Active   bool `json:"active"`
}

func (t Triangle) Vertices() [3]int { return [3]int{t.A, t.B, t.C} }

func (t *Triangle) SetVertices(v [3]int) { t.A, t.B, t.C = v[0], v[1], v[2] }

// Mesh is an indexed triangle mesh. Materials are integer face labels; the
// names are optional and used by OBJ output.
type Mesh struct {
	Vertices      []Vertex   `json:"vertices"`
	Triangles     []Triangle `json:"triangles"`
	MaterialNames []string   `json:"material_names"`
	Name          string     `json:"name,omitempty"`
}

func NewMesh() *Mesh {
	return &Mesh{MaterialNames: []string{"default"}}
}

func (m *Mesh) AddVertex(p Vec3) int {
	m.Vertices = append(m.Vertices, Vertex{Pos: p, Active: true})
	return len(m.Vertices) - 1
}

func (m *Mesh) AddTriangle(a, b, c, material int) (int, error) {
	if a == b || b == c || a == c {
		return -1, fmt.Errorf("degenerate triangle (%d,%d,%d)", a, b, c)
	}
	if material < 0 {
		material = 0
	}
	for len(m.MaterialNames) <= material {
		m.MaterialNames = append(m.MaterialNames, fmt.Sprintf("material_%d", len(m.MaterialNames)))
	}
	m.Triangles = append(m.Triangles, Triangle{
		A: a, B: b, C: c, Material: material, Active: true,
	})
	return len(m.Triangles) - 1, nil
}

func (m *Mesh) AddMaterial(name string) int {
	for i, existing := range m.MaterialNames {
		if existing == name {
			return i
		}
	}
	m.MaterialNames = append(m.MaterialNames, name)
	return len(m.MaterialNames) - 1
}

func (m *Mesh) Clone() *Mesh {
	cp := &Mesh{
		Vertices:      append([]Vertex(nil), m.Vertices...),
		Triangles:     append([]Triangle(nil), m.Triangles...),
		MaterialNames: append([]string(nil), m.MaterialNames...),
		Name:          m.Name,
	}
	return cp
}

func (m *Mesh) ActiveTriangleCount() int {
	n := 0
	for _, t := range m.Triangles {
		if t.Active {
			n++
		}
	}
	return n
}

func (m *Mesh) ActiveVertexCount() int {
	n := 0
	for _, v := range m.Vertices {
		if v.Active {
			n++
		}
	}
	return n
}

func (m *Mesh) VertexPosition(id int) Vec3 { return m.Vertices[id].Pos }

func (m *Mesh) Bounds() (Vec3, Vec3) {
	min := Vec3{X: math.Inf(1), Y: math.Inf(1), Z: math.Inf(1)}
	max := Vec3{X: math.Inf(-1), Y: math.Inf(-1), Z: math.Inf(-1)}
	found := false
	for _, v := range m.Vertices {
		if !v.Active {
			continue
		}
		found = true
		min.X = math.Min(min.X, v.Pos.X)
		min.Y = math.Min(min.Y, v.Pos.Y)
		min.Z = math.Min(min.Z, v.Pos.Z)
		max.X = math.Max(max.X, v.Pos.X)
		max.Y = math.Max(max.Y, v.Pos.Y)
		max.Z = math.Max(max.Z, v.Pos.Z)
	}
	if !found {
		return Vec3{}, Vec3{}
	}
	return min, max
}

// MaxDimension returns the longest bounding-box side. It is never zero for a
// mesh with usable geometry.
func (m *Mesh) MaxDimension() float64 {
	min, max := m.Bounds()
	d := math.Max(max.X-min.X, math.Max(max.Y-min.Y, max.Z-min.Z))
	if d < 1e-12 {
		return 1
	}
	return d
}

func (m *Mesh) Center() Vec3 {
	min, max := m.Bounds()
	return min.Add(max).Mul(0.5)
}

// SignedVolume6 returns six times signed volume as the sum of triangle
// determinants. The factor six is omitted so candidate volume deltas need no
// repeated division.
func (m *Mesh) SignedVolume6() float64 {
	var sum float64
	for _, t := range m.Triangles {
		if !t.Active {
			continue
		}
		a := m.Vertices[t.A].Pos
		b := m.Vertices[t.B].Pos
		c := m.Vertices[t.C].Pos
		sum += det6(a, b, c)
	}
	return sum
}

func (m *Mesh) SignedVolume() float64 { return m.SignedVolume6() / 6 }

// FaceGeometry returns the unnormalized outward normal and twice area.
func (m *Mesh) FaceGeometry(id int) (normal Vec3, area2 float64) {
	t := m.Triangles[id]
	a := m.Vertices[t.A].Pos
	b := m.Vertices[t.B].Pos
	c := m.Vertices[t.C].Pos
	n := b.Sub(a).Cross(c.Sub(a))
	return n, n.Len()
}

func (m *Mesh) FaceNormal(id int) Vec3 {
	n, _ := m.FaceGeometry(id)
	return n.Normalized()
}
