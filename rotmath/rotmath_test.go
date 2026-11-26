package rotmath

import (
	"math"
	"testing"
)

func approxEq(t *testing.T, name string, got, want, tol float64) {
	t.Helper()
	if math.Abs(got-want) > tol {
		t.Errorf("%s = %v, want %v (tol %v)", name, got, want, tol)
	}
}

func TestEulerQuatRoundTrip(t *testing.T) {
	cases := []Euler{
		{},
		{Roll: 0.3, Pitch: -0.7, Yaw: 1.2},
		{Roll: rad(20), Pitch: rad(-45), Yaw: rad(170)},
		{Roll: rad(-10), Pitch: rad(60), Yaw: rad(-150)},
	}
	for _, e := range cases {
		got := FromEuler(e).ToEuler()
		approxEq(t, "roll", got.Roll, e.Roll, 1e-12)
		approxEq(t, "pitch", got.Pitch, e.Pitch, 1e-12)
		approxEq(t, "yaw", got.Yaw, e.Yaw, 1e-12)
	}
}

func TestQuatMatrixAgreesWithEulerMatrix(t *testing.T) {
	e := Euler{Roll: 0.4, Pitch: 0.9, Yaw: -2.1}
	qm := FromEuler(e).ToMatrix()
	em := EulerMatrix(e)
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			approxEq(t, "matrix entry", qm[i][j], em[i][j], 1e-12)
		}
	}
}

func TestSlerpShortArcAndEndpoints(t *testing.T) {
	a := FromEuler(Euler{Yaw: rad(170)})
	b := FromEuler(Euler{Yaw: rad(-170)}) // 20 degrees away
	got := Slerp(a, b.Neg(), 0.5)         // flipped stored sign
	want := FromEuler(Euler{Yaw: math.Pi})
	diff := got.Conj().Mul(want)
	if diff.RotationAngle() > 1e-6 {
		t.Fatalf("slerp midpoint angle off by %v rad", diff.RotationAngle())
	}

	start := Slerp(a, b, 0)
	if start.Conj().Mul(a).RotationAngle() > 1e-6 {
		t.Fatal("slerp must return the start pose at u=0")
	}
	end := Slerp(a, b.Neg(), 1)
	if end.Conj().Mul(b).RotationAngle() > 1e-6 {
		t.Fatal("slerp must return the end pose at u=1 regardless of sign")
	}
}

func TestSlerpConstantSpeed(t *testing.T) {
	a := FromEuler(Euler{Roll: 0.2, Yaw: 1.0})
	b := FromEuler(Euler{Roll: -0.5, Pitch: 0.6})
	total := a.Conj().Mul(b).RotationAngle()
	prev := a
	for k := 1; k <= 8; k++ {
		cur := Slerp(a, b, float64(k)/8)
		step := prev.Conj().Mul(cur).RotationAngle()
		approxEq(t, "incremental arc", step, total/8, 1e-12)
		prev = cur
	}
}

func TestNlerpRawAntipodalCollapse(t *testing.T) {
	q := FromEuler(Euler{Yaw: 0.5})
	blend, ok := NlerpRaw(q, q.Neg(), 0.5)
	if ok {
		t.Fatal("antipodal midpoint must be flagged un-normalizable")
	}
	if blend.Norm() > 1e-15 {
		t.Fatalf("blended norm = %v, want collapse to zero", blend.Norm())
	}
}

func TestOrthoErrorOfRotations(t *testing.T) {
	for _, e := range []Euler{
		{},
		{Roll: 1, Pitch: 2, Yaw: 3},
		{Yaw: rad(179)},
	} {
		if err := FromEuler(e).ToMatrix().OrthoError(); err > 1e-12 {
			t.Fatalf("rotation matrix ortho error too large: %v", err)
		}
		if err := EulerMatrix(e).OrthoError(); err > 1e-12 {
			t.Fatalf("euler matrix ortho error too large: %v", err)
		}
	}
}

func TestAngularVelocityMagnitude(t *testing.T) {
	dt := 0.01
	axisSpeed := 2.0 // rad/s about body X
	r0 := Mat3{{1, 0, 0}, {0, 1, 0}, {0, 0, 1}}
	r1 := FromEuler(Euler{Roll: axisSpeed * dt}).ToMatrix()
	omega, angle := AngularVelocity(r0, r1, dt)
	approxEq(t, "angle", angle, axisSpeed*dt, 1e-10)
	approxEq(t, "omega x", omega[0], axisSpeed, 1e-9)
	approxEq(t, "omega y", omega[1], 0, 1e-9)
	approxEq(t, "omega z", omega[2], 0, 1e-9)
}

func rad(deg float64) float64 { return deg * math.Pi / 180 }
