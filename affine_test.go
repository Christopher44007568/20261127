package main

import (
	"math"
	"testing"
)

func TestAffineCompositionOrder(t *testing.T) {
	m := Translate(10, -5).
		Mul(Rotate(math.Pi / 2)).
		Mul(Scale(2, 3))

	x, y := m.Apply(1, 2)
	assertNear(t, x, 4)
	assertNear(t, y, -3)
}

func TestAffineInverseRoundTrip(t *testing.T) {
	m := Translate(31, -12).
		Mul(Rotate(degrees(-14))).
		Mul(Scale(1.25, 0.8))
	inverse, ok := m.Inverse()
	if !ok {
		t.Fatal("invertible matrix reported singular")
	}

	for _, p := range []point{{-100, 40}, {0, 0}, {512, 512}, {900, -45}} {
		x, y := m.Apply(p.x, p.y)
		backX, backY := inverse.Apply(x, y)
		assertNear(t, backX, p.x)
		assertNear(t, backY, p.y)
	}
}

func TestMirrorAboutFlipsDirectionMarker(t *testing.T) {
	const size = 512.0
	normalized := Translate(size/2, size/2).
		Mul(MirrorX()).
		Mul(Translate(-size/2, -size/2))

	tail := point{160, 310}
	tip := point{350, 200}
	mtailX, mtailY := normalized.Apply(tail.x, tail.y)
	mtipX, mtipY := normalized.Apply(tip.x, tip.y)

	// The original marker points right; the mirror must point left while
	// retaining its y orientation. A double flip would incorrectly preserve x.
	if mtipX >= mtailX {
		t.Fatalf("mirrored marker points the wrong way: tail x=%v tip x=%v", mtailX, mtipX)
	}
	assertNear(t, mtailY, tail.y)
	assertNear(t, mtipY, tip.y)

	if normalized.Mul(normalized).Determinant() < 0 {
		t.Fatal("two mirrors should produce a positive determinant")
	}
}

func TestMirrorDeterminant(t *testing.T) {
	for _, m := range []Affine{MirrorX(), MirrorY()} {
		if m.Determinant() != -1 {
			t.Fatalf("mirror determinant = %v, want -1", m.Determinant())
		}
	}
}

func assertNear(t *testing.T, got, want float64) {
	t.Helper()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("got %v, want %v", got, want)
	}
}
