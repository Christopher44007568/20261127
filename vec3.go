package main

import "math"

// Vec3 is a minimal 3D vector used for positions, normals and tangents.
type Vec3 struct {
	X, Y, Z float64
}

func (v Vec3) Add(o Vec3) Vec3 {
	return Vec3{X: v.X + o.X, Y: v.Y + o.Y, Z: v.Z + o.Z}
}

func (v Vec3) Sub(o Vec3) Vec3 {
	return Vec3{X: v.X - o.X, Y: v.Y - o.Y, Z: v.Z - o.Z}
}

func (v Vec3) Scale(s float64) Vec3 {
	return Vec3{X: v.X * s, Y: v.Y * s, Z: v.Z * s}
}

func (v Vec3) Dot(o Vec3) float64 {
	return v.X*o.X + v.Y*o.Y + v.Z*o.Z
}

func (v Vec3) Cross(o Vec3) Vec3 {
	return Vec3{
		X: v.Y*o.Z - v.Z*o.Y,
		Y: v.Z*o.X - v.X*o.Z,
		Z: v.X*o.Y - v.Y*o.X,
	}
}

func (v Vec3) LengthSquared() float64 {
	return v.Dot(v)
}

func (v Vec3) Length() float64 {
	return math.Sqrt(v.LengthSquared())
}

// Normalized returns the unit-length vector. A zero vector is returned
// unchanged so callers never see NaN from a degenerate frame.
func (v Vec3) Normalized() Vec3 {
	length := v.Length()
	if length == 0 {
		return Vec3{}
	}
	return v.Scale(1 / length)
}

func clampScalar(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
