package skinsss

import "math"

// Vec3 is a minimal 3D vector used for points, directions and colors.
type Vec3 struct {
	X, Y, Z float64
}

func (a Vec3) Add(b Vec3) Vec3      { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3      { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Mul(b Vec3) Vec3      { return Vec3{a.X * b.X, a.Y * b.Y, a.Z * b.Z} }
func (a Vec3) Dot(b Vec3) float64   { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		X: a.Y*b.Z - a.Z*b.Y,
		Y: a.Z*b.X - a.X*b.Z,
		Z: a.X*b.Y - a.Y*b.X,
	}
}

func (a Vec3) Length() float64 {
	return sqrt(a.Dot(a))
}

func (a Vec3) Normalize() Vec3 {
	n := a.Length()
	if n < 1e-12 {
		return Vec3{}
	}
	return a.Scale(1 / n)
}

// Lerp interpolates between a and b by t in [0,1].
func Lerp(a, b Vec3, t float64) Vec3 {
	return Vec3{
		X: a.X + (b.X-a.X)*t,
		Y: a.Y + (b.Y-a.Y)*t,
		Z: a.Z + (b.Z-a.Z)*t,
	}
}

// Clamp01 clamps each component into [0,1].
func (a Vec3) Clamp01() Vec3 {
	return Vec3{clamp01(a.X), clamp01(a.Y), clamp01(a.Z)}
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

func sqrt(x float64) float64 {
	return math.Sqrt(x)
}

func abs(x float64) float64 { return math.Abs(x) }

func clamp(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

// smoothstep performs Hermite interpolation between edge0 and edge1.
func smoothstep(edge0, edge1, x float64) float64 {
	t := clamp((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}
