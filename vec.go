package main

import "math"

type Vec2 struct {
	X, Y float64
}

func (a Vec2) Add(b Vec2) Vec2      { return Vec2{a.X + b.X, a.Y + b.Y} }
func (a Vec2) Sub(b Vec2) Vec2      { return Vec2{a.X - b.X, a.Y - b.Y} }
func (a Vec2) Mul(s float64) Vec2   { return Vec2{a.X * s, a.Y * s} }
func (a Vec2) Dot(b Vec2) float64   { return a.X*b.X + a.Y*b.Y }
func (a Vec2) Cross(b Vec2) float64 { return a.X*b.Y - a.Y*b.X }
func (a Vec2) Len() float64         { return math.Hypot(a.X, a.Y) }

func (a Vec2) Unit() Vec2 {
	n := a.Len()
	if n < 1e-12 {
		return Vec2{1, 0}
	}
	return a.Mul(1 / n)
}

type Vec3 struct {
	X, Y, Z float64
}

func (a Vec3) Add(b Vec3) Vec3    { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3    { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Mul(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Len() float64       { return math.Sqrt(a.Dot(a)) }

func (a Vec3) Unit() Vec3 {
	n := a.Len()
	if n < 1e-12 {
		return Vec3{0, 0, 1}
	}
	return a.Mul(1 / n)
}

func cross3(a, b Vec3) Vec3 {
	return Vec3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

type RGB struct {
	R, G, B float64
}

func (c RGB) Add(d RGB) RGB     { return RGB{c.R + d.R, c.G + d.G, c.B + d.B} }
func (c RGB) Sub(d RGB) RGB     { return RGB{c.R - d.R, c.G - d.G, c.B - d.B} }
func (c RGB) Mul(s float64) RGB { return RGB{c.R * s, c.G * s, c.B * s} }
func (c RGB) Scale(d RGB) RGB   { return RGB{c.R * d.R, c.G * d.G, c.B * d.B} }

func (c RGB) Clamp() RGB {
	return RGB{
		R: math.Max(0, math.Min(1, c.R)),
		G: math.Max(0, math.Min(1, c.G)),
		B: math.Max(0, math.Min(1, c.B)),
	}
}

func (c RGB) Luma() float64 {
	return 0.2126*c.R + 0.7152*c.G + 0.0722*c.B
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func mix(a, b, t float64) float64 {
	return a + (b-a)*t
}
