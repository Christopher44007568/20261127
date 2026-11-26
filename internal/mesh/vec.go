package mesh

import "math"

const Epsilon = 1e-8

// Vec3 is a three dimensional vector used by the geometry kernel.
type Vec3 struct {
	X, Y, Z float64
}

func (a Vec3) Add(b Vec3) Vec3 {
	return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z}
}

func (a Vec3) Sub(b Vec3) Vec3 {
	return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z}
}

func (a Vec3) Mul(s float64) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
}

func (a Vec3) Dot(b Vec3) float64 {
	return a.X*b.X + a.Y*b.Y + a.Z*b.Z
}

func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

func (a Vec3) Length() float64 {
	return math.Sqrt(a.Dot(a))
}

func (a Vec3) Normalize() Vec3 {
	n := a.Length()
	if n <= Epsilon {
		return Vec3{}
	}
	return a.Mul(1 / n)
}

func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return a.Add(b.Sub(a).Mul(t))
}

func (a Vec3) Negate() Vec3 {
	return Vec3{-a.X, -a.Y, -a.Z}
}

func AlmostEqual(a, b Vec3) bool {
	return math.Abs(a.X-b.X) <= Epsilon &&
		math.Abs(a.Y-b.Y) <= Epsilon &&
		math.Abs(a.Z-b.Z) <= Epsilon
}
