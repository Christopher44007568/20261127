package qmesh

import (
	"math"
	"testing"

	"meshdiagnostics/internal/mesh"
)

func TestMetricsPlanarSquare(t *testing.T) {
	q := &Mesh{
		Vertices: []mesh.Vec3{
			mesh.V(0, 0, 0), mesh.V(1, 0, 0), mesh.V(1, 1, 0), mesh.V(0, 1, 0),
		},
		Faces: []Face{{0, 1, 2, 3}},
	}
	m := q.Metrics(q.Faces[0])
	if !m.Convex {
		t.Fatal("square should be convex")
	}
	if math.Abs(m.MinAngleDeg-90) > 1e-6 || math.Abs(m.MaxAngleDeg-90) > 1e-6 {
		t.Fatalf("expected 90 degree corners, got %.3f..%.3f", m.MinAngleDeg, m.MaxAngleDeg)
	}
}

func TestMetricsConcaveQuad(t *testing.T) {
	q := &Mesh{
		Vertices: []mesh.Vec3{
			mesh.V(0, 0, 0), mesh.V(2, 0, 0), mesh.V(1, 0.5, 0), mesh.V(0, 2, 0),
		},
		Faces: []Face{{0, 1, 2, 3}},
	}
	m := q.Metrics(q.Faces[0])
	if m.Convex {
		t.Fatal("dart quad should be concave")
	}
	if m.MaxAngleDeg < 200 {
		t.Fatalf("expected a reflex corner, got max %.2f", m.MaxAngleDeg)
	}
}
