package interp

import (
	"math"

	"orientation-interp/rotmath"
)

const (
	// ScenarioStart and ScenarioEnd bound the sampled timeline.
	ScenarioStart = 0.0
	ScenarioEnd   = 4.0
	// ScenarioStep is the fixed sampling period in seconds.
	ScenarioStep = 0.02
)

func rad(deg float64) float64 { return deg * math.Pi / 180 }

// Scenario builds the adversarial keyframe set.
//
//	t=0s  identity
//	t=1s  yaw +170 deg
//	t=2s  yaw -170 deg (crosses the +/-180 seam: only 20 deg of real
//	      rotation, but raw Euler lerp sweeps 340 deg the long way)
//	t=3s  yaw -168 deg stored with a sign-flipped quaternion: only 2
//	      degrees away from t=2s, but the flipped stored sign makes raw
//	      nlerp travel the ~358 degree long arc, while canonicalized
//	      slerp takes the 2 degree short arc.
//	t=4s  roll 180 deg: an exact antipodal quaternion pair relative to
//	      identity-like paths, exercising the slerp fallback branch.
func Scenario() []Keyframe {
	poses := []struct {
		t     float64
		euler rotmath.Euler
		flip  bool
	}{
		{0, rotmath.Euler{}, false},
		{1, rotmath.Euler{Yaw: rad(170)}, false},
		{2, rotmath.Euler{Yaw: rad(-170)}, false},
		{3, rotmath.Euler{Yaw: rad(-168)}, true},
		{4, rotmath.Euler{Roll: math.Pi}, false},
	}

	keys := make([]Keyframe, len(poses))
	for i, p := range poses {
		q := rotmath.FromEuler(p.euler)
		if p.flip {
			q = q.Neg()
		}
		keys[i] = Keyframe{T: p.t, Euler: p.euler, Quat: q}
	}
	return keys
}

// SegmentArc returns the shortest-arc angle between two keyframe
// orientations, ignoring stored quaternion signs. It is used by the
// report to quantify how far each method deviates from the required arc.
func SegmentArc(a, b Keyframe) float64 {
	qa, qb := a.Quat, b.Quat
	if qa.Dot(qb) < 0 {
		qb = qb.Neg()
	}
	return qa.Conj().Mul(qb).RotationAngle()
}
