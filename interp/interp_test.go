package interp

import (
	"math"
	"testing"

	"orientation-interp/rotmath"
)

const eps = 1e-12

func sweepAngle(t *testing.T, method Method, keys []Keyframe, t0, t1 float64) float64 {
	t.Helper()
	samples := Timeline(method, keys, t0, t1, ScenarioStep)
	total := 0.0
	for i := 1; i < len(samples); i++ {
		rel := samples[i-1].Quat.Conj().Mul(samples[i].Quat)
		if rel.W < 0 {
			rel = rel.Neg()
		}
		total += rel.RotationAngle()
	}
	return total
}

func TestSlerpRepeatedPoseProducesNoRotation(t *testing.T) {
	// Exact duplicate pose with an intentionally flipped stored sign.
	e := rotmath.Euler{Yaw: 0.9}
	keys := []Keyframe{
		{T: 0, Euler: e, Quat: rotmath.FromEuler(e)},
		{T: 1, Euler: e, Quat: rotmath.FromEuler(e).Neg()},
	}
	angle := sweepAngle(t, MethodSlerp, keys, 0, 1)
	if angle > 1e-6 {
		t.Fatalf("stable slerp swept %v rad across a repeated pose", angle)
	}
	samples := Timeline(MethodSlerp, keys, 0, 1, 0.02)
	ref := samples[0].Quat
	for _, s := range samples {
		if s.Quat.Conj().Mul(ref).RotationAngle() > eps {
			t.Fatal("stable slerp must hold a constant quaternion on repeated poses")
		}
		if s.OrthoError > 1e-12 {
			t.Fatalf("ortho error drift: %v", s.OrthoError)
		}
	}
}

func TestSlerpNearRepeatedFlippedPoseTakesShortArc(t *testing.T) {
	keys := Scenario()
	angle := sweepAngle(t, MethodSlerp, keys, 2, 3)
	if math.Abs(angle-2*math.Pi/180) > 0.01 {
		t.Fatalf("stable slerp swept %v rad, want 2 degrees", angle)
	}
}

func TestNlerpTakesLongWayOnFlippedNearRepeatedPose(t *testing.T) {
	keys := Scenario()
	angle := sweepAngle(t, MethodNlerp, keys, 2, 3)
	if math.Abs(angle-358*math.Pi/180) > 0.05 {
		t.Fatalf("raw nlerp swept %v rad, want ~358 degrees", angle)
	}
}

func TestNlerpDegeneratesOnExactAntipodalMidpoint(t *testing.T) {
	e := rotmath.Euler{Yaw: 0.7}
	keys := []Keyframe{
		{T: 0, Euler: e, Quat: rotmath.FromEuler(e)},
		{T: 1, Euler: e, Quat: rotmath.FromEuler(e).Neg()},
	}
	samples := Timeline(MethodNlerp, keys, 0, 1, 0.02)
	degenerate := 0
	for _, s := range samples {
		if s.Degenerate {
			degenerate++
		}
	}
	if degenerate == 0 {
		t.Fatal("raw nlerp must report the antipodal midpoint as degenerate")
	}
}

func TestEulerLerpTakesLongWayAcrossSeam(t *testing.T) {
	keys := Scenario()
	// Segment [1,2]: true shortest arc is 20 degrees; raw Euler lerp
	// must sweep 340 degrees the long way.
	long := sweepAngle(t, MethodEuler, keys, 1, 2)
	if math.Abs(long-340*math.Pi/180) > 0.02 {
		t.Fatalf("euler lerp swept %v rad, want 340 degrees", long)
	}
	short := sweepAngle(t, MethodSlerp, keys, 1, 2)
	if math.Abs(short-20*math.Pi/180) > 0.02 {
		t.Fatalf("slerp swept %v rad, want 20 degrees", short)
	}
}

func TestSlerpEverySegmentIsShortArc(t *testing.T) {
	keys := Scenario()
	for i := 0; i+1 < len(keys); i++ {
		arc := SegmentArc(keys[i], keys[i+1])
		if arc > math.Pi+eps {
			t.Fatalf("segment %d shortest arc %v exceeds pi", i, arc)
		}
		got := sweepAngle(t, MethodSlerp, keys, keys[i].T, keys[i+1].T)
		if math.Abs(got-arc) > 0.02 {
			t.Fatalf("segment %d slerp swept %v, want %v", i, got, arc)
		}
	}
}

func TestAllMethodsHitKeyframeEndpoints(t *testing.T) {
	keys := Scenario()
	for _, method := range []Method{MethodEuler, MethodNlerp, MethodSlerp} {
		samples := Timeline(method, keys, ScenarioStart, ScenarioEnd, ScenarioStep)
		for _, k := range keys {
			idx := int(math.Round((k.T - ScenarioStart) / ScenarioStep))
			got := samples[idx].Quat.Conj().Mul(k.Quat)
			if got.RotationAngle() > 1e-6 {
				t.Fatalf("%s misses keyframe at t=%v", method, k.T)
			}
		}
	}
}

func TestSlerpAngularVelocityZeroOnRepeatedPose(t *testing.T) {
	e := rotmath.Euler{Pitch: 0.4, Yaw: -1.1}
	keys := []Keyframe{
		{T: 0, Euler: e, Quat: rotmath.FromEuler(e)},
		{T: 1, Euler: e, Quat: rotmath.FromEuler(e).Neg()},
	}
	samples := Timeline(MethodSlerp, keys, 0, 1, 0.02)
	for i := 1; i < len(samples); i++ {
		omega, _ := rotmath.AngularVelocity(samples[i-1].Matrix, samples[i].Matrix, ScenarioStep)
		norm := math.Sqrt(omega[0]*omega[0] + omega[1]*omega[1] + omega[2]*omega[2])
		if norm > 1e-9 {
			t.Fatalf("nonzero omega %v on repeated pose", norm)
		}
	}
}
