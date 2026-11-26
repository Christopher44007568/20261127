package interp

import (
	"math"

	"orientation-interp/rotmath"
)

// Method identifies one of the three compared interpolation strategies.
type Method string

const (
	// MethodEuler linearly interpolates raw Euler angles. Angles are not
	// unwrapped, so a keyframe jump across the +/-180 degree seam takes
	// the long way around.
	MethodEuler Method = "euler-lerp"
	// MethodNlerp is normalized linear interpolation without any
	// shortest-arc sign correction, reproducing the classic sign-flip
	// flicker and the antipodal midpoint singularity.
	MethodNlerp Method = "quat-nlerp-raw"
	// MethodSlerp canonicalizes keyframe quaternion signs and always
	// interpolates along the stable shortest arc.
	MethodSlerp Method = "quat-slerp-short"
)

// Keyframe is an oriented pose at a point in time. Quat holds the stored
// quaternion as supplied (its sign may be intentionally flipped by the
// scenario), while Euler carries the authoring angles.
type Keyframe struct {
	T     float64
	Euler rotmath.Euler
	Quat  rotmath.Quat
}

// Sample is one rendered pose on the output timeline.
type Sample struct {
	T          float64
	Quat       rotmath.Quat
	Matrix     rotmath.Mat3
	OrthoError float64
	// Degenerate is true when the interpolation produced an un-normalizable
	// quaternion (raw nlerp at an antipodal midpoint). The matrix then
	// falls back to the identity and the failure is visible in the report.
	Degenerate bool
	// Blend stores the pre-normalization quaternion so the report can plot
	// the norm collapse that drives nlerp flicker.
	Blend rotmath.Quat
}

// Timeline renders every sample in [start, end] at the fixed step dt.
func Timeline(method Method, keys []Keyframe, start, end, dt float64) []Sample {
	prepared := prepare(method, keys)
	count := int(math.Round((end-start)/dt)) + 1
	out := make([]Sample, 0, count)
	for i := 0; i < count; i++ {
		t := start + float64(i)*dt
		out = append(out, sampleAt(method, prepared, t))
	}
	return out
}

type preparedKey struct {
	Keyframe
	canonical rotmath.Quat
}

func prepare(method Method, keys []Keyframe) []preparedKey {
	out := make([]preparedKey, len(keys))
	for i, k := range keys {
		out[i] = preparedKey{Keyframe: k, canonical: k.Quat}
	}
	if method != MethodSlerp {
		return out
	}
	// Rebuild every canonical quaternion directly from the authoring Euler
	// angles. The construction is deterministic, so identical orientations
	// produce bit-identical quaternions (a repeated pose stays perfectly
	// still) and every adjacent pair lies on the same hemisphere, making
	// each segment a shortest arc. The original stored signs remain
	// available for the raw nlerp method, which must still see the flips.
	for i := range out {
		out[i].canonical = rotmath.FromEuler(out[i].Euler)
	}
	return out
}

func sampleAt(method Method, keys []preparedKey, t float64) Sample {
	i0, i1, u := locate(keys, t)
	switch method {
	case MethodEuler:
		return eulerSample(keys[i0].Euler, keys[i1].Euler, t, u)
	case MethodNlerp:
		return nlerpSample(keys[i0].Quat, keys[i1].Quat, t, u)
	default:
		return slerpSample(keys[i0].canonical, keys[i1].canonical, t, u)
	}
}

func locate(keys []preparedKey, t float64) (int, int, float64) {
	if t <= keys[0].T {
		return 0, 0, 0
	}
	last := len(keys) - 1
	if t >= keys[last].T {
		return last, last, 1
	}
	for i := 0; i < last; i++ {
		if t < keys[i+1].T {
			span := keys[i+1].T - keys[i].T
			u := (t - keys[i].T) / span
			if u < 0 {
				u = 0
			}
			return i, i + 1, u
		}
	}
	return last, last, 0
}

func eulerSample(a, b rotmath.Euler, t, u float64) Sample {
	// Deliberately no angle unwrapping: the raw component lerp is the
	// behavior being evaluated against the 180 degree seam.
	e := rotmath.Euler{
		Roll:  lerp(a.Roll, b.Roll, u),
		Pitch: lerp(a.Pitch, b.Pitch, u),
		Yaw:   lerp(a.Yaw, b.Yaw, u),
	}
	matrix := rotmath.EulerMatrix(e)
	return Sample{
		T:          t,
		Quat:       rotmath.FromEuler(e),
		Matrix:     matrix,
		OrthoError: matrix.OrthoError(),
		Blend:      rotmath.FromEuler(e),
	}
}

func nlerpSample(a, b rotmath.Quat, t, u float64) Sample {
	blend, ok := rotmath.NlerpRaw(a, b, u)
	q, valid := blend.Normalized()
	s := Sample{T: t, Quat: q, Blend: blend, Degenerate: !ok || !valid}
	if s.Degenerate {
		// Keep the pipeline running; the failure is surfaced explicitly.
		s.Quat = rotmath.Quat{W: 1}
	}
	s.Matrix = s.Quat.ToMatrix()
	s.OrthoError = s.Matrix.OrthoError()
	return s
}

func slerpSample(a, b rotmath.Quat, t, u float64) Sample {
	q := rotmath.Slerp(a, b, u)
	matrix := q.ToMatrix()
	return Sample{
		T:          t,
		Quat:       q,
		Matrix:     matrix,
		OrthoError: matrix.OrthoError(),
		Blend:      q,
	}
}

func lerp(a, b, u float64) float64 {
	return a + (b-a)*u
}
