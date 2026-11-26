package report

import (
	"math"

	"orientation-interp/interp"
	"orientation-interp/rotmath"
)

// Series holds every evaluated quantity for one interpolation method.
type Series struct {
	Method     interp.Method
	Label      string
	Samples    []interp.Sample
	Speed      []float64 // |omega| in rad/s, NaN at a degenerate sample
	OmegaX     []float64
	OmegaY     []float64
	OmegaZ     []float64
	Ortho      []float64 // orthogonality error, NaN at a degenerate sample
	BlendNorm  []float64 // pre-normalization quaternion norm
	SweptAngle float64   // total shortest-arc rotation across the timeline
	Degenerate int
	MaxSpeed   float64
	MaxOrtho   float64
}

// MethodLabel maps a method id to a human-readable chart label.
func MethodLabel(m interp.Method) string {
	switch m {
	case interp.MethodEuler:
		return "Euler angle lerp (no unwrap)"
	case interp.MethodNlerp:
		return "Raw quaternion nlerp (no sign fix)"
	default:
		return "Canonicalized shortest-arc slerp"
	}
}

// BuildSeries evaluates one method over the full scenario.
func BuildSeries(method interp.Method, keys []interp.Keyframe, dt float64) Series {
	samples := interp.Timeline(method, keys, interp.ScenarioStart, interp.ScenarioEnd, dt)
	n := len(samples)
	s := Series{
		Method:    method,
		Label:     MethodLabel(method),
		Samples:   samples,
		Speed:     make([]float64, n),
		OmegaX:    make([]float64, n),
		OmegaY:    make([]float64, n),
		OmegaZ:    make([]float64, n),
		Ortho:     make([]float64, n),
		BlendNorm: make([]float64, n),
	}

	for i, sample := range samples {
		s.BlendNorm[i] = sample.Blend.Norm()
		if sample.Degenerate {
			s.Degenerate++
			s.Speed[i] = math.NaN()
			s.OmegaX[i], s.OmegaY[i], s.OmegaZ[i] = math.NaN(), math.NaN(), math.NaN()
			s.Ortho[i] = math.NaN()
			continue
		}
		s.Ortho[i] = sample.OrthoError
		s.MaxOrtho = math.Max(s.MaxOrtho, sample.OrthoError)
	}

	for i := 1; i < n; i++ {
		if samples[i].Degenerate || samples[i-1].Degenerate {
			s.Speed[i] = math.NaN()
			s.OmegaX[i], s.OmegaY[i], s.OmegaZ[i] = math.NaN(), math.NaN(), math.NaN()
			continue
		}
		omega, angle := rotmath.AngularVelocity(samples[i-1].Matrix, samples[i].Matrix, dt)
		s.OmegaX[i], s.OmegaY[i], s.OmegaZ[i] = omega[0], omega[1], omega[2]
		s.Speed[i] = math.Sqrt(omega[0]*omega[0] + omega[1]*omega[1] + omega[2]*omega[2])
		s.MaxSpeed = math.Max(s.MaxSpeed, s.Speed[i])
		s.SweptAngle += angle
	}
	return s
}

// KeyframeTimes returns the scenario keyframe timestamps for chart markers.
func KeyframeTimes(keys []interp.Keyframe) []float64 {
	out := make([]float64, len(keys))
	for i, k := range keys {
		out[i] = k.T
	}
	return out
}

// ArcBudget sums the true shortest arcs between adjacent keyframes. It is
// the minimum total rotation any correct interpolation must produce.
func ArcBudget(keys []interp.Keyframe) float64 {
	total := 0.0
	for i := 0; i+1 < len(keys); i++ {
		total += interp.SegmentArc(keys[i], keys[i+1])
	}
	return total
}
