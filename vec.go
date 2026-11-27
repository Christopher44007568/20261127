package main

import "math"

type Vec3 struct {
	X, Y, Z float64
}

func V(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }

func (a Vec3) Add(b Vec3) Vec3 { return V(a.X+b.X, a.Y+b.Y, a.Z+b.Z) }

func (a Vec3) Sub(b Vec3) Vec3 { return V(a.X-b.X, a.Y-b.Y, a.Z-b.Z) }

func (a Vec3) Mul(b Vec3) Vec3 { return V(a.X*b.X, a.Y*b.Y, a.Z*b.Z) }

func (a Vec3) Scale(s float64) Vec3 { return V(a.X*s, a.Y*s, a.Z*s) }

func (a Vec3) Div(b Vec3) Vec3 { return V(a.X/b.X, a.Y/b.Y, a.Z/b.Z) }

func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return V(
		a.Y*b.Z-a.Z*b.Y,
		a.Z*b.X-a.X*b.Z,
		a.X*b.Y-a.Y*b.X,
	)
}

func (a Vec3) LengthSquared() float64 { return a.Dot(a) }

func (a Vec3) Length() float64 { return math.Sqrt(a.LengthSquared()) }

func (a Vec3) Normalize() Vec3 {
	length := a.Length()
	if length < 1e-12 {
		return V(0, 1, 0)
	}
	return a.Scale(1 / length)
}

func (a Vec3) Abs() Vec3 { return V(math.Abs(a.X), math.Abs(a.Y), math.Abs(a.Z)) }

func clampFloat(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func clamp01(x float64) float64 { return clampFloat(x, 0, 1) }

func smoothstep(edge0, edge1, x float64) float64 {
	t := clamp01((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}

func lerpFloat(a, b, t float64) float64 { return a + (b-a)*t }

func lerpVec(a, b Vec3, t float64) Vec3 {
	return V(lerpFloat(a.X, b.X, t), lerpFloat(a.Y, b.Y, t), lerpFloat(a.Z, b.Z, t))
}
