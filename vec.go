package main

import (
	"math"
)

type Vec3 struct {
	X, Y, Z float64
}

func V(x, y, z float64) Vec3 { return Vec3{x, y, z} }

func (a Vec3) Add(b Vec3) Vec3 { return V(a.X+b.X, a.Y+b.Y, a.Z+b.Z) }

func (a Vec3) Sub(b Vec3) Vec3 { return V(a.X-b.X, a.Y-b.Y, a.Z-b.Z) }

func (a Vec3) Mul(s float64) Vec3 { return V(a.X*s, a.Y*s, a.Z*s) }

func (a Vec3) Div(s float64) Vec3 { return V(a.X/s, a.Y/s, a.Z/s) }

func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return V(
		a.Y*b.Z-a.Z*b.Y,
		a.Z*b.X-a.X*b.Z,
		a.X*b.Y-a.Y*b.X,
	)
}

func (a Vec3) Norm2() float64 { return a.Dot(a) }

func (a Vec3) Norm() float64 { return math.Sqrt(a.Norm2()) }

func (a Vec3) Normalize() Vec3 {
	n := a.Norm()
	if n < 1e-15 {
		return V(0, 1, 0)
	}
	return a.Div(n)
}

func (a Vec3) Abs() Vec3 { return V(math.Abs(a.X), math.Abs(a.Y), math.Abs(a.Z)) }

func (a Vec3) Min(b Vec3) Vec3 {
	return V(math.Min(a.X, b.X), math.Min(a.Y, b.Y), math.Min(a.Z, b.Z))
}

func (a Vec3) Max(b Vec3) Vec3 {
	return V(math.Max(a.X, b.X), math.Max(a.Y, b.Y), math.Max(a.Z, b.Z))
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

func clampInt(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}
