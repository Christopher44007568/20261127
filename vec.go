package main

import "math"

type Vec3 struct {
	X, Y, Z float64
}

func V(x, y, z float64) Vec3 { return Vec3{x, y, z} }

func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }

func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }

func (a Vec3) Mul(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }

func (a Vec3) MulVec(b Vec3) Vec3 { return Vec3{a.X * b.X, a.Y * b.Y, a.Z * b.Z} }

func (a Vec3) Div(s float64) Vec3 { return Vec3{a.X / s, a.Y / s, a.Z / s} }

func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return V(
		a.Y*b.Z-a.Z*b.Y,
		a.Z*b.X-a.X*b.Z,
		a.X*b.Y-a.Y*b.X,
	)
}

func (a Vec3) Length() float64 { return math.Sqrt(a.Dot(a)) }

func (a Vec3) Normalize() Vec3 {
	l := a.Length()
	if l < 1e-12 {
		return V(0, 0, 1)
	}
	return a.Div(l)
}

func (a Vec3) Clamp01() Vec3 {
	return V(clamp01(a.X), clamp01(a.Y), clamp01(a.Z))
}

func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}

func clampFloat(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func lerpFloat(a, b, t float64) float64 { return a + (b-a)*t }

func smoothstepFloat(lo, hi, x float64) float64 {
	t := clamp01((x - lo) / (hi - lo))
	return t * t * (3 - 2*t)
}
