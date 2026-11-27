package flakepaint

import "math"

// Vec3 is a small, allocation-free vector used by the CPU renderer.
type Vec3 struct {
	X, Y, Z float64
}

func V(x, y, z float64) Vec3 { return Vec3{x, y, z} }

func (a Vec3) Add(b Vec3) Vec3 { return V(a.X+b.X, a.Y+b.Y, a.Z+b.Z) }
func (a Vec3) Sub(b Vec3) Vec3 { return V(a.X-b.X, a.Y-b.Y, a.Z-b.Z) }
func (a Vec3) Mul(s float64) Vec3 {
	return V(a.X*s, a.Y*s, a.Z*s)
}
func (a Vec3) MulV(b Vec3) Vec3 {
	return V(a.X*b.X, a.Y*b.Y, a.Z*b.Z)
}
func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }
func (a Vec3) Cross(b Vec3) Vec3 {
	return V(a.Y*b.Z-a.Z*b.Y, a.Z*b.X-a.X*b.Z, a.X*b.Y-a.Y*b.X)
}
func (a Vec3) Length() float64 { return math.Sqrt(a.Dot(a)) }
func (a Vec3) Normalize() Vec3 {
	l := a.Length()
	if l < 1e-12 {
		return V(0, 0, 0)
	}
	return a.Mul(1 / l)
}
func (a Vec3) Lerp(b Vec3, t float64) Vec3 {
	return a.Mul(1 - t).Add(b.Mul(t))
}
func (a Vec3) Max0() Vec3 {
	return V(math.Max(0, a.X), math.Max(0, a.Y), math.Max(0, a.Z))
}
func (a Vec3) Luminance() float64 {
	return 0.2126*a.X + 0.7152*a.Y + 0.0722*a.Z
}

func reflectDir(in, n Vec3) Vec3 {
	return in.Sub(n.Mul(2 * in.Dot(n)))
}

func clamp(x, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, x))
}

func mix(a, b, t float64) float64 { return a + (b-a)*t }

func smoothstep(edge0, edge1, x float64) float64 {
	if edge0 == edge1 {
		if x < edge0 {
			return 0
		}
		return 1
	}
	t := clamp((x-edge0)/(edge1-edge0), 0, 1)
	return t * t * (3 - 2*t)
}

func fract(x float64) float64 { return x - math.Floor(x) }

func pow5(x float64) float64 {
	x2 := x * x
	return x2 * x2 * x
}

// Frame is an orthonormal tangent frame used to apply flake normals.
type Frame struct {
	N, T, B Vec3
}

func frameFromNormal(n Vec3) Frame {
	n = n.Normalize()
	helper := V(0, 1, 0)
	if math.Abs(n.Y) > 0.92 {
		helper = V(1, 0, 0)
	}
	t := helper.Cross(n).Normalize()
	b := n.Cross(t).Normalize()
	return Frame{N: n, T: t, B: b}
}

// Ray is a unit-direction world-space ray.
type Ray struct {
	Origin, Dir Vec3
}

// Camera is a perspective pinhole camera with an arbitrary aspect ratio.
type Camera struct {
	Position Vec3
	Target   Vec3
	Up       Vec3
	FovY     float64
}

func (c Camera) Ray(x, y, aspect float64) Ray {
	forward := c.Target.Sub(c.Position).Normalize()
	right := forward.Cross(c.Up).Normalize()
	up := right.Cross(forward).Normalize()
	halfH := math.Tan(c.FovY * 0.5)
	dir := forward.
		Add(right.Mul((x - 0.5) * 2 * halfH * aspect)).
		Add(up.Mul((0.5 - y) * 2 * halfH)).
		Normalize()
	return Ray{Origin: c.Position, Dir: dir}
}

func (c Camera) PixelAngle(height int) float64 {
	return 2 * math.Tan(c.FovY*0.5) / float64(height)
}
