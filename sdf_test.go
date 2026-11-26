package skinsss

import "testing"

func TestEllipsoidSDFSign(t *testing.T) {
	e := Ellipsoid{Center: Vec3{}, Radii: Vec3{1, 1, 1}}
	cases := []struct {
		p    Vec3
		want int // expected sign: -1 inside, 0 surface, +1 outside
	}{
		{Vec3{}, -1},
		{Vec3{1, 0, 0}, 0},
		{Vec3{2, 0, 0}, 1},
	}
	for _, c := range cases {
		got := e.SDF(c.p)
		sign := 0
		if got < -1e-9 {
			sign = -1
		} else if got > 1e-9 {
			sign = 1
		}
		if sign != c.want {
			t.Fatalf("SDF(%v) = %v, want sign %d", c.p, got, c.want)
		}
	}
}

func TestChordThicknessMonotone(t *testing.T) {
	scene := DefaultScene()
	// A ray starting deep inside and heading outward travels farther than one
	// starting just under the near surface.
	deep := scene.ChordThickness(Vec3{0, 0, -0.7}, Vec3{0, 0, 1})
	shallow := scene.ChordThickness(Vec3{0, 0, 0.7}, Vec3{0, 0, 1})
	if deep <= shallow {
		t.Fatalf("deep chord %v should exceed shallow chord %v", deep, shallow)
	}
	// A ray pointing outward from an outside start finds no interior.
	if scene.ChordThickness(Vec3{0, 0, 5}, Vec3{0, 0, 1}) != 0 {
		t.Fatal("exterior ray should report zero chord")
	}
}

func TestCurvatureConvexPositive(t *testing.T) {
	scene := DefaultScene()
	if k := scene.Curvature(Vec3{0, 0, 0.7}); k <= 0 {
		t.Fatalf("convex head surface curvature should be positive, got %v", k)
	}
}
