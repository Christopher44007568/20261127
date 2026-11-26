package render

import "math"

// Vec3 is a compact linear-space RGB color or a 3D direction.
type Vec3 struct {
	X, Y, Z float64
}

func V(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }

func (a Vec3) Add(b Vec3) Vec3 { return V(a.X+b.X, a.Y+b.Y, a.Z+b.Z) }
func (a Vec3) Sub(b Vec3) Vec3 { return V(a.X-b.X, a.Y-b.Y, a.Z-b.Z) }
func (a Vec3) Mul(s float64) Vec3 {
	return V(a.X*s, a.Y*s, a.Z*s)
}
func (a Vec3) MulVec(b Vec3) Vec3 {
	return V(a.X*b.X, a.Y*b.Y, a.Z*b.Z)
}
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return V(
		a.Y*b.Z-a.Z*b.Y,
		a.Z*b.X-a.X*b.Z,
		a.X*b.Y-a.Y*b.X,
	)
}

func (a Vec3) Length() float64 { return math.Sqrt(a.Dot(a)) }

func (a Vec3) Normalize() Vec3 {
	n := a.Length()
	if n < 1e-12 {
		return V(0, 0, 0)
	}
	return a.Mul(1 / n)
}

func (a Vec3) Clamp01() Vec3 {
	return V(Clamp01(a.X), Clamp01(a.Y), Clamp01(a.Z))
}

func (a Vec3) Luminance() float64 {
	return 0.2126*a.X + 0.7152*a.Y + 0.0722*a.Z
}

func Lerp(a, b, t float64) float64 { return a + (b-a)*t }

func LerpVec(a, b Vec3, t float64) Vec3 {
	return a.Add(b.Sub(a).Mul(t))
}

func Clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func Clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func Smoothstep(edge0, edge1, x float64) float64 {
	t := Clamp01((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}

// Reflect returns the mirror direction for an incident direction d and normal n.
func Reflect(d, n Vec3) Vec3 {
	return d.Sub(n.Mul(2 * d.Dot(n)))
}

// Basis builds an orthonormal tangent frame around n.
func Basis(n Vec3) (Vec3, Vec3) {
	helper := V(0, 1, 0)
	if math.Abs(n.Y) > 0.95 {
		helper = V(1, 0, 0)
	}
	t := n.Cross(helper).Normalize()
	b := n.Cross(t)
	return t, b
}
