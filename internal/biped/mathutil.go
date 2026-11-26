package biped

import "math"

type Vec2 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

func clamp(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func clamp01(v float64) float64 {
	return clamp(v, 0, 1)
}

func lerp(a, b, t float64) float64 {
	return a + (b-a)*t
}

func smoothStep(t float64) float64 {
	t = clamp01(t)
	return t * t * (3 - 2*t)
}

func fract(v float64) float64 {
	return v - math.Floor(v)
}

func expApproach(current, target, dt, timeConstant float64) float64 {
	if timeConstant <= 0 {
		return target
	}
	k := 1 - math.Exp(-dt/timeConstant)
	return lerp(current, target, k)
}

func wrapPhase(v float64) float64 {
	return fract(v)
}
