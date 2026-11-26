package csg

import (
	"math"
	"testing"

	"meshbool/internal/mesh"
)

func TestBoxBooleanVolumes(t *testing.T) {
	a := mesh.Box(mesh.Vec3{X: -1, Y: -1, Z: -1}, mesh.Vec3{X: 0.5, Y: 1, Z: 1})
	b := mesh.Box(mesh.Vec3{X: -0.5, Y: -0.5, Z: -0.5}, mesh.Vec3{X: 1.5, Y: 1.5, Z: 1.5})
	expected := map[Operation]float64{
		Union:        11.75,
		Intersection: 2.25,
		Difference:   3.75,
	}
	for operation, want := range expected {
		result := Apply(a, b, operation)
		if got := result.Mesh.Volume(); math.Abs(got-want) > 1e-5 {
			t.Errorf("%s volume = %.9f, want %.9f (%d triangles)",
				operation, got, want, len(result.Mesh.Triangles))
		}
		if len(result.Mesh.Triangles) == 0 {
			t.Errorf("%s returned an empty mesh", operation)
		}
	}
}

func TestSpherePlaneVolumes(t *testing.T) {
	sphere := mesh.Sphere(mesh.Vec3{}, 1, 24, 12)
	lowerHalfSpace := mesh.Box(
		mesh.Vec3{X: -2, Y: -2, Z: -2},
		mesh.Vec3{X: 2, Y: 0, Z: 2},
	)
	halfSphere := 2 * math.Pi / 3
	expected := map[Operation]float64{
		Union:        32 + halfSphere,
		Intersection: halfSphere,
		Difference:   halfSphere,
	}
	for operation, want := range expected {
		result := Apply(sphere, lowerHalfSpace, operation)
		got := result.Mesh.Volume()
		if math.Abs(got-want) > 0.06 {
			t.Errorf("%s volume = %.6f, want %.6f", operation, got, want)
		}
	}
}

func TestCoplanarThinSheets(t *testing.T) {
	a := mesh.Box(
		mesh.Vec3{X: -1.2, Y: -0.45, Z: -0.1},
		mesh.Vec3{X: 0.2, Y: 0.45, Z: 0.1},
	)
	b := mesh.Box(
		mesh.Vec3{X: -0.2, Y: -0.45, Z: -0.1},
		mesh.Vec3{X: 1.2, Y: 0.45, Z: 0.1},
	)
	expected := map[Operation]float64{
		Union:        0.432,
		Intersection: 0.072,
		Difference:   0.18,
	}
	for operation, want := range expected {
		result := Apply(a, b, operation)
		if got := result.Mesh.Volume(); math.Abs(got-want) > 1e-6 {
			t.Errorf("%s volume = %.9f, want %.9f (%d triangles)",
				operation, got, want, len(result.Mesh.Triangles))
		}
	}
}

func TestSharedEdgeThinSheets(t *testing.T) {
	a := mesh.Box(
		mesh.Vec3{X: -1, Y: -0.5, Z: -0.1},
		mesh.Vec3{X: 0, Y: 0.5, Z: 0.1},
	)
	b := mesh.Box(
		mesh.Vec3{X: 0, Y: -0.5, Z: -0.1},
		mesh.Vec3{X: 1, Y: 0.5, Z: 0.1},
	)
	union := Apply(a, b, Union)
	if got, want := union.Mesh.Volume(), 0.4; math.Abs(got-want) > 1e-7 {
		t.Fatalf("union volume = %.9f, want %.9f", got, want)
	}
	difference := Apply(a, b, Difference)
	if got, want := difference.Mesh.Volume(), 0.2; math.Abs(got-want) > 1e-7 {
		t.Fatalf("difference volume = %.9f, want %.9f", got, want)
	}
	intersection := Apply(a, b, Intersection)
	if got := intersection.Mesh.Volume(); got > 1e-8 {
		t.Fatalf("shared-edge intersection volume = %.9f, want zero", got)
	}
}

func TestNoDuplicateOrNonManifoldFaces(t *testing.T) {
	a := mesh.Box(mesh.Vec3{X: -1.15, Y: -1, Z: -1}, mesh.Vec3{X: 0.45, Y: 1, Z: 1})
	b := mesh.Box(mesh.Vec3{X: -0.45, Y: -0.45, Z: -0.45}, mesh.Vec3{X: 1.25, Y: 1.25, Z: 1.25})
	for _, operation := range []Operation{Union, Intersection, Difference} {
		stats := Apply(a, b, operation).Mesh.AnalyzeTopology()
		if stats.DuplicateFaces != 0 {
			t.Errorf("%s duplicate faces = %d", operation, stats.DuplicateFaces)
		}
		if stats.NonManifoldEdges != 0 {
			t.Errorf("%s non-manifold edges = %d", operation, stats.NonManifoldEdges)
		}
	}
}
