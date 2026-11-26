// Package linalg provides the small 3D vector toolkit used by the
// feature-line extraction and software renderer. It depends only on the
// Go standard library.
package linalg

import "math"

// Vec3 is a float64 3-component vector or point.
type Vec3 struct {
	X, Y, Z float64
}

func (a Vec3) Add(b Vec3) Vec3 { return Vec3{X: a.X + b.X, Y: a.Y + b.Y, Z: a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{X: a.X - b.X, Y: a.Y - b.Y, Z: a.Z - b.Z} }
func (a Vec3) Scale(s float64) Vec3 {
	return Vec3{X: a.X * s, Y: a.Y * s, Z: a.Z * s}
}
func (a Vec3) Mul(b Vec3) Vec3 { return Vec3{X: a.X * b.X, Y: a.Y * b.Y, Z: a.Z * b.Z} }

func (a Vec3) Dot(b Vec3) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{X: a.Y*b.Z - a.Z*b.Y, Y: a.Z*b.X - a.X*b.Z, Z: a.X*b.Y - a.Y*b.X}

}

func (a Vec3) Len2() float64 { return a.Dot(a) }
func (a Vec3) Len() float64  { return math.Sqrt(a.Len2()) }

func (a Vec3) Dist(b Vec3) float64 { return a.Sub(b).Len() }

func (a Vec3) Normalized() Vec3 {
	l := a.Len()
	if l < 1e-15 {
		return Vec3{}
	}
	return a.Scale(1 / l)
}

// Angle returns the unsigned angle in [0, pi] between a and b.
func Angle(a, b Vec3) float64 {
	an := a.Normalized()
	bn := b.Normalized()
	if an.Len2() == 0 || bn.Len2() == 0 {
		return 0
	}
	c := clamp(an.Dot(bn), -1, 1)
	return math.Acos(c)
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

// Clamp restricts v to [lo, hi].
func Clamp(v, lo, hi float64) float64 { return clamp(v, lo, hi) }

// Lerp linearly interpolates between a and b.
func Lerp(a, b Vec3, t float64) Vec3 {
	return Vec3{X: a.X + (b.X-a.X)*t, Y: a.Y + (b.Y-a.Y)*t, Z: a.Z + (b.Z-a.Z)*t}

}
