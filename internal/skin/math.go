package skin

import "math"

// Vec3 is a point or direction in the bind-pose coordinate system.
type Vec3 struct {
	X, Y, Z float64
}

func (v Vec3) Add(other Vec3) Vec3 {
	return Vec3{v.X + other.X, v.Y + other.Y, v.Z + other.Z}
}

func (v Vec3) Sub(other Vec3) Vec3 {
	return Vec3{v.X - other.X, v.Y - other.Y, v.Z - other.Z}
}

func (v Vec3) Scale(s float64) Vec3 {
	return Vec3{v.X * s, v.Y * s, v.Z * s}
}

func (v Vec3) Dot(other Vec3) float64 {
	return v.X*other.X + v.Y*other.Y + v.Z*other.Z
}

func (v Vec3) Cross(other Vec3) Vec3 {
	return Vec3{
		X: v.Y*other.Z - v.Z*other.Y,
		Y: v.Z*other.X - v.X*other.Z,
		Z: v.X*other.Y - v.Y*other.X,
	}
}

func (v Vec3) Length() float64 {
	return math.Sqrt(v.Dot(v))
}

func (v Vec3) Normalize() Vec3 {
	length := v.Length()
	if length == 0 {
		return Vec3{}
	}
	return v.Scale(1 / length)
}

func lerp(a, b Vec3, t float64) Vec3 {
	return a.Scale(1 - t).Add(b.Scale(t))
}

// Quat uses the scalar-first convention (w, x, y, z).
type Quat struct {
	W, X, Y, Z float64
}

func QuatIdentity() Quat {
	return Quat{W: 1}
}

func QuatFromAxisAngle(axis Vec3, radians float64) Quat {
	axis = axis.Normalize()
	sine, cosine := math.Sincos(radians * 0.5)
	return Quat{W: cosine, X: axis.X * sine, Y: axis.Y * sine, Z: axis.Z * sine}
}

func (q Quat) Dot(other Quat) float64 {
	return q.W*other.W + q.X*other.X + q.Y*other.Y + q.Z*other.Z
}

func (q Quat) Add(other Quat) Quat {
	return Quat{q.W + other.W, q.X + other.X, q.Y + other.Y, q.Z + other.Z}
}

func (q Quat) Sub(other Quat) Quat {
	return Quat{q.W - other.W, q.X - other.X, q.Y - other.Y, q.Z - other.Z}
}

func (q Quat) Scale(s float64) Quat {
	return Quat{q.W * s, q.X * s, q.Y * s, q.Z * s}
}

func (q Quat) Length() float64 {
	return math.Sqrt(q.Dot(q))
}

func (q Quat) Normalize() Quat {
	length := q.Length()
	if length == 0 {
		return QuatIdentity()
	}
	scale := 1 / length
	return Quat{q.W * scale, q.X * scale, q.Y * scale, q.Z * scale}
}

func (q Quat) Conjugate() Quat {
	return Quat{W: q.W, X: -q.X, Y: -q.Y, Z: -q.Z}
}

func (q Quat) Negate() Quat {
	return Quat{-q.W, -q.X, -q.Y, -q.Z}
}

// Mul composes rotations as q * other: other is applied first, then q.
func (q Quat) Mul(other Quat) Quat {
	return Quat{
		W: q.W*other.W - q.X*other.X - q.Y*other.Y - q.Z*other.Z,
		X: q.W*other.X + q.X*other.W + q.Y*other.Z - q.Z*other.Y,
		Y: q.W*other.Y - q.X*other.Z + q.Y*other.W + q.Z*other.X,
		Z: q.W*other.Z + q.X*other.Y - q.Y*other.X + q.Z*other.W,
	}
}

func (q Quat) Rotate(v Vec3) Vec3 {
	u := Vec3{q.X, q.Y, q.Z}
	uv := u.Cross(v)
	uuv := u.Cross(uv)
	return v.Add(uv.Scale(2 * q.W)).Add(uuv.Scale(2))
}

// Transform is a rigid transform p -> R p + t.
type Transform struct {
	Rotation    Quat
	Translation Vec3
}

func IdentityTransform() Transform {
	return Transform{Rotation: QuatIdentity()}
}

func RotationAbout(axis Vec3, radians float64) Transform {
	return Transform{Rotation: QuatFromAxisAngle(axis, radians)}
}

func (t Transform) Apply(v Vec3) Vec3 {
	return t.Rotation.Rotate(v).Add(t.Translation)
}

func (t Transform) Inverse() Transform {
	inverse := t.Rotation.Conjugate()
	return Transform{
		Rotation:    inverse,
		Translation: inverse.Rotate(t.Translation).Scale(-1),
	}
}

func (t Transform) IsIdentity() bool {
	const epsilon = 1e-15
	return math.Abs(t.Rotation.W-1) <= epsilon &&
		math.Abs(t.Rotation.X) <= epsilon &&
		math.Abs(t.Rotation.Y) <= epsilon &&
		math.Abs(t.Rotation.Z) <= epsilon &&
		math.Abs(t.Translation.X) <= epsilon &&
		math.Abs(t.Translation.Y) <= epsilon &&
		math.Abs(t.Translation.Z) <= epsilon
}
