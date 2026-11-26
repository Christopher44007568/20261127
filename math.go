package main

import "math"

// Vec3 is a 3D vector in metres.
type Vec3 struct {
	X, Y, Z float64
}

// Quat is a unit quaternion (X, Y, Z, W).
type Quat struct {
	X, Y, Z, W float64
}

func (v Vec3) Add(o Vec3) Vec3      { return Vec3{v.X + o.X, v.Y + o.Y, v.Z + o.Z} }
func (v Vec3) Sub(o Vec3) Vec3      { return Vec3{v.X - o.X, v.Y - o.Y, v.Z - o.Z} }
func (v Vec3) Scale(s float64) Vec3 { return Vec3{v.X * s, v.Y * s, v.Z * s} }
func (v Vec3) Dot(o Vec3) float64   { return v.X*o.X + v.Y*o.Y + v.Z*o.Z }

func (v Vec3) Cross(o Vec3) Vec3 {
	return Vec3{
		v.Y*o.Z - v.Z*o.Y,
		v.Z*o.X - v.X*o.Z,
		v.X*o.Y - v.Y*o.X,
	}
}

func (v Vec3) Len() float64 { return math.Sqrt(v.Dot(v)) }

func (v Vec3) Normalized() Vec3 {
	n := v.Len()
	if n < 1e-12 {
		return Vec3{}
	}
	return v.Scale(1 / n)
}

func lerp(a, b, u float64) float64 { return a + (b-a)*u }

func lerpVec3(a, b Vec3, u float64) Vec3 {
	return Vec3{lerp(a.X, b.X, u), lerp(a.Y, b.Y, u), lerp(a.Z, b.Z, u)}
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

func smoothstep(u float64) float64 {
	u = clamp(u, 0, 1)
	return u * u * (3 - 2*u)
}

// Rotate rotates v by the (unit) quaternion q.
func (q Quat) Rotate(v Vec3) Vec3 {
	u := Vec3{q.X, q.Y, q.Z}
	s := q.W
	return u.Scale(2 * u.Dot(v)).
		Add(v.Scale(s*s - u.Dot(u))).
		Add(u.Cross(v).Scale(2 * s))
}

// Mul returns q * r (apply r first, then q).
func (q Quat) Mul(r Quat) Quat {
	return Quat{
		W: q.W*r.W - q.X*r.X - q.Y*r.Y - q.Z*r.Z,
		X: q.W*r.X + q.X*r.W + q.Y*r.Z - q.Z*r.Y,
		Y: q.W*r.Y - q.X*r.Z + q.Y*r.W + q.Z*r.X,
		Z: q.W*r.Z + q.X*r.Y - q.Y*r.X + q.Z*r.W,
	}
}

func (q Quat) Normalized() Quat {
	n := math.Sqrt(q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z)
	if n < 1e-12 {
		return Quat{W: 1}
	}
	return Quat{q.X / n, q.Y / n, q.Z / n, q.W / n}
}

// Inv returns the inverse of a unit quaternion.
func (q Quat) Inv() Quat { return Quat{-q.X, -q.Y, -q.Z, q.W} }

// Dot returns the quaternion 4D dot product.
func (q Quat) Dot(r Quat) float64 {
	return q.W*r.W + q.X*r.X + q.Y*r.Y + q.Z*r.Z
}

// RotX returns a rotation of angle radians around the X axis.
func RotX(a float64) Quat {
	c, s := math.Cos(a/2), math.Sin(a/2)
	return Quat{X: s, W: c}
}

// RotY returns a rotation of angle radians around the Y axis (yaw).
func RotY(a float64) Quat {
	c, s := math.Cos(a/2), math.Sin(a/2)
	return Quat{Y: s, W: c}
}

// FromTo returns the shortest rotation mapping unit vector a onto b.
func FromTo(a, b Vec3) Quat {
	a = a.Normalized()
	b = b.Normalized()
	d := a.Dot(b)
	if d > 1-1e-10 {
		return Quat{W: 1}
	}
	if d < -1+1e-10 {
		axis := Vec3{1, 0, 0}.Cross(a)
		if axis.Len() < 1e-8 {
			axis = Vec3{0, 0, 1}.Cross(a)
		}
		axis = axis.Normalized()
		return Quat{X: axis.X, Y: axis.Y, Z: axis.Z, W: 0}
	}
	axis := a.Cross(b)
	return Quat{X: axis.X, Y: axis.Y, Z: axis.Z, W: 1 + d}.Normalized()
}

// Nlerp performs normalized linear quaternion interpolation, taking the
// shortest arc.
func Nlerp(a, b Quat, u float64) Quat {
	if a.Dot(b) < 0 {
		b = Quat{-b.X, -b.Y, -b.Z, -b.W}
	}
	return Quat{
		lerp(a.X, b.X, u),
		lerp(a.Y, b.Y, u),
		lerp(a.Z, b.Z, u),
		lerp(a.W, b.W, u),
	}.Normalized()
}
