package mesh

import (
	"math"
	"strings"
	"testing"
)

func TestParseOBJPolygonAndNegativeIndex(t *testing.T) {
	obj := `
# square plus triangle
v 0 0 0
v 1 0 0
v 1 1 0
v 0 1 0
f 1 2 3 4
f -4 -3 -1
`
	m, err := ParseOBJ(strings.NewReader(obj))
	if err != nil {
		t.Fatal(err)
	}
	if len(m.Vertices) != 4 {
		t.Fatalf("vertices = %d, want 4", len(m.Vertices))
	}
	if len(m.Faces) != 3 {
		t.Fatalf("triangles = %d, want 3 (fan plus one)", len(m.Faces))
	}
}

func TestWeldDuplicateVerticesAndFaces(t *testing.T) {
	m := &TriangleMesh{
		Vertices: []Vec3{
			V(0, 0, 0), V(1, 0, 0), V(0, 1, 0),
			V(0, 0, 0), V(1, 0, 0), V(0, 1, 0),
		},
		Faces: []Tri{{0, 1, 2}, {5, 4, 3}},
	}
	w := m.Weld(1e-7)
	if len(w.Vertices) != 3 {
		t.Fatalf("welded vertices = %d, want 3", len(w.Vertices))
	}
	if len(w.Faces) != 1 {
		t.Fatalf("welded faces = %d, want 1", len(w.Faces))
	}
}

func TestDetectFeaturesOnRightAngleCrease(t *testing.T) {
	m := &TriangleMesh{
		Vertices: []Vec3{
			V(0, 0, 0), V(1, 0, 0), V(0.5, 1, 0), V(0.5, 0, 1),
		},
		Faces: []Tri{{0, 1, 2}, {1, 0, 3}},
	}
	info := m.DetectFeatures(FeatureOptions{CreaseAngleDeg: 45})
	if info.Crease != 1 {
		t.Fatalf("crease edges = %d, want 1", info.Crease)
	}
	if info.Boundary != 4 {
		t.Fatalf("boundary edges = %d, want 4", info.Boundary)
	}
	if !info.Edges[Edge(0, 1)] {
		t.Fatal("shared edge should be a feature")
	}
	if info.NonManifold != 0 {
		t.Fatalf("unexpected non-manifold edges: %d", info.NonManifold)
	}
}

func TestOrientMakesWindingConsistent(t *testing.T) {
	m := &TriangleMesh{
		Vertices: []Vec3{
			V(0, 0, 0), V(1, 0, 0), V(0, 1, 0), V(0, 0, 1), V(0, 1, 1),
		},
		Faces: []Tri{{0, 1, 2}, {0, 1, 3}, {0, 3, 4}},
	}
	m.Orient()
	for key, edge := range m.EdgeMap() {
		if len(edge.Faces) != 2 {
			continue
		}
		if edgeDirection(m.Faces[edge.Faces[0]], key) == edgeDirection(m.Faces[edge.Faces[1]], key) {
			t.Fatalf("adjacent faces traverse edge %v in the same direction", key)
		}
	}
}

func TestEdgeNormalAndBounds(t *testing.T) {
	p, q := V(1, 2, 3), V(2, 3, 4)
	if math.Abs(p.Sub(q).Len()-math.Sqrt(3)) > 1e-12 {
		t.Fatal("unexpected vector length")
	}
	lo, hi := Bounds([]Vec3{p, q, V(-1, 4, 0)})
	if lo != V(-1, 2, 0) || hi != V(2, 4, 4) {
		t.Fatalf("bounds = %v..%v", lo, hi)
	}
}
