package main

import "math"

// Vec3 is a right-handed world vector: X forward, Y left, Z up.
type Vec3 struct {
	X, Y, Z float64
}

func v3(x, y, z float64) Vec3 { return Vec3{x, y, z} }

func (a Vec3) Add(b Vec3) Vec3      { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3      { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Dot(b Vec3) float64   { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

func (a Vec3) Len() float64 { return math.Sqrt(a.Dot(a)) }

func (a Vec3) Normalized() Vec3 {
	n := a.Len()
	if n < 1e-12 {
		return Vec3{}
	}
	return a.Scale(1 / n)
}

// Lerp returns a + (b-a)*t.
func Lerp(a, b Vec3, t float64) Vec3 {
	return a.Add(b.Sub(a).Scale(t))
}

// Smoothstep eases t with 3t^2-2t^3 and clamps outside [0,1].
func Smoothstep(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

// Yaw rotates around the world Z (up) axis by angle radians.
func (a Vec3) Yaw(angle float64) Vec3 {
	c, s := math.Cos(angle), math.Sin(angle)
	return Vec3{
		X: c*a.X - s*a.Y,
		Y: s*a.X + c*a.Y,
		Z: a.Z,
	}
}

// Clamp constrains v to [lo, hi].
func Clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Deg converts degrees to radians.
func Deg(d float64) float64 { return d * math.Pi / 180 }
