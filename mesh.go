package holefill

import (
	"fmt"
	"math"
)

type Tri [3]int

type FaceKind int

const (
	KindScan FaceKind = iota
	KindPatch
	KindBottom
	KindWall
)

type FaceTag struct {
	Kind   FaceKind
	Class  HoleClass
	LoopID int
}

type Mesh struct {
	V    []Vec3
	T    []Tri
	Norm []Vec3
	Tags []FaceTag
}

func (m *Mesh) validateIndex(i int) error {
	if i < 0 || i >= len(m.V) {
		return fmt.Errorf("vertex index %d out of range [0,%d)", i, len(m.V))
	}
	return nil
}

func (m *Mesh) Validate() error {
	if len(m.Tags) != len(m.T) {
		return fmt.Errorf("tags length %d != triangles length %d", len(m.Tags), len(m.T))
	}
	for ti, tri := range m.T {
		for _, idx := range tri {
			if err := m.validateIndex(idx); err != nil {
				return fmt.Errorf("tri %d: %w", ti, err)
			}
		}
		if tri[0] == tri[1] || tri[1] == tri[2] || tri[0] == tri[2] {
			return fmt.Errorf("tri %d is degenerate: %v", ti, tri)
		}
	}
	return nil
}

func (m *Mesh) AppendVertex(p Vec3) int {
	m.V = append(m.V, p)
	return len(m.V) - 1
}

func (m *Mesh) AppendTri(tri Tri, tag FaceTag) {
	m.T = append(m.T, tri)
	m.Tags = append(m.Tags, tag)
}

func (m *Mesh) FaceNormal(ti int) Vec3 {
	t := m.T[ti]
	a := m.V[t[0]]
	b := m.V[t[1]]
	c := m.V[t[2]]
	return b.Sub(a).Cross(c.Sub(a)).Normalize()
}

func (m *Mesh) ComputeNormals() []Vec3 {
	n := make([]Vec3, len(m.V))
	for ti, t := range m.T {
		a := m.V[t[0]]
		b := m.V[t[1]]
		c := m.V[t[2]]
		fn := b.Sub(a).Cross(c.Sub(a))
		for _, idx := range t {
			n[idx] = n[idx].Add(fn)
		}
		_ = ti
	}
	for i := range n {
		nn := n[i].Normalize()
		if nn.Len() < 0.5 {
			n[i] = Vec3{0, 0, 1}
		} else {
			n[i] = nn
		}
	}
	m.Norm = n
	return n
}

type edgeKey struct {
	A, B int
}

func undirectedEdge(a, b int) edgeKey {
	if a > b {
		a, b = b, a
	}
	return edgeKey{a, b}
}

type BoundaryEdge struct {
	A, B int
}

func (m *Mesh) BoundaryEdges() []BoundaryEdge {
	count := make(map[edgeKey]int, len(m.T)*3)
	for _, t := range m.T {
		for i := 0; i < 3; i++ {
			a, b := t[i], t[(i+1)%3]
			count[undirectedEdge(a, b)]++
		}
	}
	var out []BoundaryEdge
	for _, t := range m.T {
		dirs := [3][2]int{{t[0], t[1]}, {t[1], t[2]}, {t[2], t[0]}}
		for _, e := range dirs {
			if count[undirectedEdge(e[0], e[1])] == 1 {
				out = append(out, BoundaryEdge{A: e[0], B: e[1]})
			}
		}
	}
	return out
}

func (m *Mesh) EdgeUseCounts() map[edgeKey]int {
	count := make(map[edgeKey]int, len(m.T)*3)
	for _, t := range m.T {
		for i := 0; i < 3; i++ {
			a, b := t[i], t[(i+1)%3]
			count[undirectedEdge(a, b)]++
		}
	}
	return count
}

func (m *Mesh) IsClosedManifold() bool {
	if err := m.Validate(); err != nil {
		return false
	}
	for _, c := range m.EdgeUseCounts() {
		if c != 2 {
			return false
		}
	}
	return true
}

func approxEqualFloat(a, b, eps float64) bool {
	return math.Abs(a-b) <= eps
}
