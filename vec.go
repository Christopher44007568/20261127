package main

import "math"

type Vec2 struct {
	X, Y float64
}

func (a Vec2) Add(b Vec2) Vec2 { return Vec2{a.X + b.X, a.Y + b.Y} }
func (a Vec2) Sub(b Vec2) Vec2 { return Vec2{a.X - b.X, a.Y - b.Y} }
func (a Vec2) Mul(s float64) Vec2 {
	return Vec2{a.X * s, a.Y * s}
}

func (a Vec2) Dot(b Vec2) float64 { return a.X*b.X + a.Y*b.Y }

func (a Vec2) Len() float64 { return math.Hypot(a.X, a.Y) }

type Vec3 struct {
	X, Y, Z float64
}

func V3(x, y, z float64) Vec3 { return Vec3{x, y, z} }

func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Mul(s float64) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
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

func (a Vec3) Normalized() Vec3 {
	l := a.Len()
	if l < 1e-12 {
		return Vec3{}
	}
	return a.Mul(1 / l)
}

func clampFloat(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

func lerpFloat(a, b, t float64) float64 { return a + (b-a)*t }

func gauss2(x, y, cx, cy, sx, sy float64) float64 {
	dx := (x - cx) / sx
	dy := (y - cy) / sy
	return math.Exp(-0.5 * (dx*dx + dy*dy))
}
