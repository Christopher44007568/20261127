package mesh

import (
	"bytes"
	"math"
	"testing"
)

func TestPrimitiveVolumesAndNormals(t *testing.T) {
	box := Box(Vec3{X: -1, Y: -1, Z: -1}, Vec3{X: 1, Y: 1, Z: 1})
	if got := box.Volume(); math.Abs(got-8) > 1e-9 {
		t.Fatalf("box volume = %.9f, want 8", got)
	}
	if box.SignedVolume() < 0 {
		t.Fatal("box normals are not outward")
	}
	sphere := Sphere(Vec3{}, 1, 32, 16)
	if got, want := sphere.Volume(), 4*math.Pi/3; math.Abs(got-want) > 0.08 {
		t.Fatalf("sphere volume = %.9f, want %.9f", got, want)
	}
	if sphere.SignedVolume() < 0 {
		t.Fatal("sphere normals are not outward")
	}
}

func TestOrientOutwardCavity(t *testing.T) {
	outer := Box(Vec3{X: -1, Y: -1, Z: -1}, Vec3{X: 1, Y: 1, Z: 1})
	inner := Box(Vec3{X: -0.5, Y: -0.5, Z: -0.5}, Vec3{X: 0.5, Y: 0.5, Z: 0.5})
	combined := Mesh{Triangles: append(outer.Triangles, inner.Triangles...)}
	oriented := combined.OrientOutward()
	if got, want := oriented.Volume(), 7.0; math.Abs(got-want) > 1e-9 {
		t.Fatalf("cavity volume = %.9f, want %.9f", got, want)
	}
}

func TestOBJRoundTrip(t *testing.T) {
	original := Sphere(Vec3{X: 0.25, Y: -0.1, Z: 0.05}, 1, 24, 12)
	var buffer bytes.Buffer
	if err := WriteOBJ(&buffer, original); err != nil {
		t.Fatal(err)
	}
	roundTrip, err := ReadOBJ(&buffer)
	if err != nil {
		t.Fatal(err)
	}
	if len(roundTrip.Triangles) != len(original.Triangles) {
		t.Fatalf("triangle count = %d, want %d", len(roundTrip.Triangles), len(original.Triangles))
	}
	if got, want := roundTrip.Volume(), original.Volume(); math.Abs(got-want) > 1e-12 {
		t.Fatalf("round-trip volume = %.15f, want %.15f", got, want)
	}
}
