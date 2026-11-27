package main

import "math"

type Vec2 struct {
	X, Y float64
}

func v2(x, y float64) Vec2 { return Vec2{X: x, Y: y} }

func (a Vec2) Add(b Vec2) Vec2    { return Vec2{a.X + b.X, a.Y + b.Y} }
func (a Vec2) Sub(b Vec2) Vec2    { return Vec2{a.X - b.X, a.Y - b.Y} }
func (a Vec2) Mul(s float64) Vec2 { return Vec2{a.X * s, a.Y * s} }
func (a Vec2) MulV(b Vec2) Vec2   { return Vec2{a.X * b.X, a.Y * b.Y} }

func (a Vec2) Dot(b Vec2) float64 { return a.X*b.X + a.Y*b.Y }

func (a Vec2) Cross(b Vec2) float64 { return a.X*b.Y - a.Y*b.X }

func (a Vec2) Len() float64 { return math.Hypot(a.X, a.Y) }

func (a Vec2) Norm() Vec2 {
	l := a.Len()
	if l < 1e-15 {
		return Vec2{}
	}
	return a.Mul(1 / l)
}

func (a Vec2) Rotate(angle float64) Vec2 {
	c, s := math.Cos(angle), math.Sin(angle)
	return Vec2{a.X*c - a.Y*s, a.X*s + a.Y*c}
}

type Vec3 struct {
	X, Y, Z float64
}

func v3(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }

func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Mul(s float64) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
}

func (a Vec3) MulV(b Vec3) Vec3 {
	return Vec3{a.X * b.X, a.Y * b.Y, a.Z * b.Z}
}

func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

func (a Vec3) Len() float64 { return math.Sqrt(a.Dot(a)) }

func (a Vec3) Norm() Vec3 {
	l := a.Len()
	if l < 1e-15 {
		return Vec3{}
	}
	return a.Mul(1 / l)
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

func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func smoothstep(edge0, edge1, x float64) float64 {
	t := clamp01((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}

func mix(a, b, t float64) float64 { return a*(1-t) + b*t }
