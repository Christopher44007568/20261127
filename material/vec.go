package material

import "math"

// Vec3 is a small float64 vector used by the CPU shader.
type Vec3 struct {
	X, Y, Z float64
}

func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Mul(s float64) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
}
func (a Vec3) MulVec(b Vec3) Vec3 {
	return Vec3{a.X * b.X, a.Y * b.Y, a.Z * b.Z}
}
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Length() float64    { return math.Sqrt(a.Dot(a)) }

func (a Vec3) Normalize() Vec3 {
	l := a.Length()
	if l < 1e-12 {
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

func mix(a, b, t float64) float64 { return a + (b-a)*t }

func smoothStep(edge0, edge1, x float64) float64 {
	t := clamp01((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}
