package main

import "math"

type Vec3 struct {
	X, Y, Z float64
}

func v(x, y, z float64) Vec3 { return Vec3{x, y, z} }

func (a Vec3) Add(b Vec3) Vec3      { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3      { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Mul(b Vec3) Vec3      { return Vec3{a.X * b.X, a.Y * b.Y, a.Z * b.Z} }
func (a Vec3) Scale(s float64) Vec3 { return Vec3{a.X * s, a.Y * s, a.Z * s} }
func (a Vec3) Neg() Vec3            { return Vec3{-a.X, -a.Y, -a.Z} }
func (a Vec3) Dot(b Vec3) float64   { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Len2() float64        { return a.Dot(a) }
func (a Vec3) Len() float64         { return math.Sqrt(a.Len2()) }

func (a Vec3) Normalize() Vec3 {
	l := a.Len()
	if l < 1e-20 {
		return Vec3{0, 0, 1}
	}
	return a.Scale(1 / l)
}

func cross(a, b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}

// reflect returns the reflection of d around normal n (both unit vectors).
func reflect(d, n Vec3) Vec3 {
	return d.Sub(n.Scale(2 * d.Dot(n)))
}

// safeMax0 clamps negative components to zero and neutralizes NaN/Inf.
func safeMax0(x float64) float64 {
	if math.IsNaN(x) || x < 0 {
		return 0
	}
	if math.IsInf(x, 1) {
		return math.MaxFloat32
	}
	return x
}

func clampf(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func clamp01(x float64) float64 { return clampf(x, 0, 1) }

func mix(a, b, t float64) float64 { return a + (b-a)*t }

// rotateTangent rotates unit vector t by angle rad in the plane
// perpendicular to normal n, keeping t orthogonal to n.
func rotateTangent(t, n Vec3, rad float64) Vec3 {
	b := cross(n, t).Normalize()
	return t.Scale(math.Cos(rad)).Add(b.Scale(math.Sin(rad))).Normalize()
}

// makeFrame builds an orthonormal frame from normal n and tangent angle rad.
func makeFrame(n Vec3, rad float64) Frame {
	var ref Vec3
	if math.Abs(n.Y) < 0.95 {
		ref = Vec3{0, 1, 0}
	} else {
		ref = Vec3{1, 0, 0}
	}
	t0 := ref.Sub(n.Scale(ref.Dot(n))).Normalize()
	t := rotateTangent(t0, n, rad)
	b := cross(n, t).Normalize()
	return Frame{N: n, T: t, B: b}
}

// Frame is a tangent-space frame; T follows the fiber direction.
type Frame struct {
	N, T, B Vec3
}

func (f Frame) local(dir Vec3) (x, y, z float64) {
	return dir.Dot(f.T), dir.Dot(f.B), dir.Dot(f.N)
}
