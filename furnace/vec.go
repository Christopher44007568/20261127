package furnace

import "math"

// Vec3 is both a direction and a linear RGB color.
type Vec3 struct {
	X, Y, Z float64
}

func V(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }

func (a Vec3) Add(b Vec3) Vec3 { return V(a.X+b.X, a.Y+b.Y, a.Z+b.Z) }

func (a Vec3) Sub(b Vec3) Vec3 { return V(a.X-b.X, a.Y-b.Y, a.Z-b.Z) }

func (a Vec3) Mul(s float64) Vec3 { return V(a.X*s, a.Y*s, a.Z*s) }

func (a Vec3) Div(s float64) Vec3 { return V(a.X/s, a.Y/s, a.Z/s) }

func (a Vec3) MulV(b Vec3) Vec3 { return V(a.X*b.X, a.Y*b.Y, a.Z*b.Z) }

func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return V(
		a.Y*b.Z-a.Z*b.Y,
		a.Z*b.X-a.X*b.Z,
		a.X*b.Y-a.Y*b.X,
	)
}

func (a Vec3) LengthSqr() float64 { return a.Dot(a) }

func (a Vec3) Length() float64 { return math.Sqrt(a.LengthSqr()) }

func (a Vec3) Normalize() Vec3 {
	l := a.Length()
	if l < 1e-12 || math.IsNaN(l) || math.IsInf(l, 0) {
		return V(0, 0, 1)
	}
	return a.Div(l)
}

func (a Vec3) Saturate() Vec3 { return V(saturate(a.X), saturate(a.Y), saturate(a.Z)) }

func (a Vec3) Clamp(lo, hi float64) Vec3 {
	return V(clamp(a.X, lo, hi), clamp(a.Y, lo, hi), clamp(a.Z, lo, hi))
}

func (a Vec3) MaxChannel() float64 { return math.Max(a.X, math.Max(a.Y, a.Z)) }

func (a Vec3) MinChannel() float64 { return math.Min(a.X, math.Min(a.Y, a.Z)) }

func saturate(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func pow5(x float64) float64 {
	x = saturate(x)
	x2 := x * x
	return x2 * x2 * x
}

func finite(x float64) bool {
	return !math.IsNaN(x) && !math.IsInf(x, 0)
}

func finiteVec(v Vec3) bool {
	return finite(v.X) && finite(v.Y) && finite(v.Z)
}
