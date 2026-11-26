package mesh

import "math"

// Vec3 is a three-dimensional vector used for positions and normals.
type Vec3 struct {
	X, Y, Z float64
}

func V(x, y, z float64) Vec3 { return Vec3{x, y, z} }

func (a Vec3) Add(b Vec3) Vec3 { return Vec3{a.X + b.X, a.Y + b.Y, a.Z + b.Z} }
func (a Vec3) Sub(b Vec3) Vec3 { return Vec3{a.X - b.X, a.Y - b.Y, a.Z - b.Z} }
func (a Vec3) Scale(s float64) Vec3 {
	return Vec3{a.X * s, a.Y * s, a.Z * s}
}
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Cross(b Vec3) Vec3 {
	return Vec3{
		a.Y*b.Z - a.Z*b.Y,
		a.Z*b.X - a.X*b.Z,
		a.X*b.Y - a.Y*b.X,
	}
}
func (a Vec3) Len2() float64   { return a.Dot(a) }
func (a Vec3) Len() float64    { return math.Sqrt(a.Len2()) }
func (a Vec3) Min(b Vec3) Vec3 { return V(math.Min(a.X, b.X), math.Min(a.Y, b.Y), math.Min(a.Z, b.Z)) }
func (a Vec3) Max(b Vec3) Vec3 { return V(math.Max(a.X, b.X), math.Max(a.Y, b.Y), math.Max(a.Z, b.Z)) }
func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return a.Add(b.Sub(a).Scale(t))
}
func (a Vec3) Normalize() Vec3 {
	l := a.Len()
	if l < 1e-14 {
		return Vec3{}
	}
	return a.Scale(1 / l)
}

func Bounds(vertices []Vec3) (Vec3, Vec3) {
	if len(vertices) == 0 {
		return Vec3{}, Vec3{}
	}
	lo, hi := vertices[0], vertices[0]
	for _, p := range vertices[1:] {
		lo = lo.Min(p)
		hi = hi.Max(p)
	}
	return lo, hi
}
