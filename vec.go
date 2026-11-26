package main

import "math"

// Vec3 is a 3D vector/point.
type Vec3 struct {
	X, Y, Z float64
}

func (a Vec3) Add(b Vec3) Vec3      { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3      { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Dot(b Vec3) float64   { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{a.Y*b.Z - a.Z*b.Y, a.Z*b.X - a.X*b.Z, a.X*b.Y - a.Y*b.X}
}

func (a Vec3) Len() float64        { return math.Sqrt(a.Dot(a)) }
func (a Vec3) Dist(b Vec3) float64 { return a.Sub(b).Len() }

func (a Vec3) Norm() Vec3 {
	l := a.Len()
	if l < 1e-12 {
		return Vec3{}
	}
	return a.Scale(1 / l)
}

func (a Vec3) Lerp(b Vec3, t float64) Vec3 { return a.Add(b.Sub(a).Scale(t)) }

func angleBetween(a, b Vec3) float64 {
	d := a.Norm().Dot(b.Norm())
	if d > 1 {
		d = 1
	}
	if d < -1 {
		d = -1
	}
	return math.Acos(d)
}
