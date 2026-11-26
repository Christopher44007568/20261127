package hierbox

import "math"

// Vec3 is a float64 3D vector. The character uses X forward, Y up and Z side.
type Vec3 struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
}

func V(x, y, z float64) Vec3 { return Vec3{X: x, Y: y, Z: z} }

func (a Vec3) Add(b Vec3) Vec3 { return V(a.X+b.X, a.Y+b.Y, a.Z+b.Z) }

func (a Vec3) Sub(b Vec3) Vec3 { return V(a.X-b.X, a.Y-b.Y, a.Z-b.Z) }

func (a Vec3) Scale(s float64) Vec3 { return V(a.X*s, a.Y*s, a.Z*s) }

func (a Vec3) Dot(b Vec3) float64 { return a.X*b.X + a.Y*b.Y + a.Z*b.Z }

func (a Vec3) Cross(b Vec3) Vec3 {
	return V(
		a.Y*b.Z-a.Z*b.Y,
		a.Z*b.X-a.X*b.Z,
		a.X*b.Y-a.Y*b.X,
	)
}

func (a Vec3) Len() float64 { return math.Sqrt(a.Dot(a)) }

func (a Vec3) Normalized() Vec3 {
	l := a.Len()
	if l < 1e-12 {
		return Vec3{}
	}
	return a.Scale(1 / l)
}

func (a Vec3) Min(b Vec3) Vec3 {
	return V(math.Min(a.X, b.X), math.Min(a.Y, b.Y), math.Min(a.Z, b.Z))
}

func (a Vec3) Max(b Vec3) Vec3 {
	return V(math.Max(a.X, b.X), math.Max(a.Y, b.Y), math.Max(a.Z, b.Z))
}

// Quat is a unit quaternion in (X,Y,Z,W) order.
type Quat struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
	Z float64 `json:"z"`
	W float64 `json:"w"`
}

func IdentityQuat() Quat { return Quat{W: 1} }

func AxisAngle(axis Vec3, angle float64) Quat {
	n := axis.Normalized()
	h := angle * 0.5
	s := math.Sin(h)
	return Quat{X: n.X * s, Y: n.Y * s, Z: n.Z * s, W: math.Cos(h)}
}

func (q Quat) Mul(r Quat) Quat {
	return Quat{
		X: q.W*r.X + q.X*r.W + q.Y*r.Z - q.Z*r.Y,
		Y: q.W*r.Y - q.X*r.Z + q.Y*r.W + q.Z*r.X,
		Z: q.W*r.Z + q.X*r.Y - q.Y*r.X + q.Z*r.W,
		W: q.W*r.W - q.X*r.X - q.Y*r.Y - q.Z*r.Z,
	}
}

// Mat4 is row-major 4x4 affine transform data.
type Mat4 [4][4]float64

func IdentityMat4() Mat4 {
	return Mat4{
		{1, 0, 0, 0},
		{0, 1, 0, 0},
		{0, 0, 1, 0},
		{0, 0, 0, 1},
	}
}

func TranslationMat4(p Vec3) Mat4 {
	m := IdentityMat4()
	m[0][3] = p.X
	m[1][3] = p.Y
	m[2][3] = p.Z
	return m
}

func TRS(p Vec3, q Quat, s Vec3) Mat4 {
	x, y, z, w := q.X, q.Y, q.Z, q.W
	xx, yy, zz := x*x, y*y, z*z
	xy, xz, yz := x*y, x*z, y*z
	wx, wy, wz := w*x, w*y, w*z
	return Mat4{
		{(1 - 2*(yy+zz)) * s.X, 2 * (xy - wz) * s.X, 2 * (xz + wy) * s.X, p.X},
		{2 * (xy + wz) * s.Y, (1 - 2*(xx+zz)) * s.Y, 2 * (yz - wx) * s.Y, p.Y},
		{2 * (xz - wy) * s.Z, 2 * (yz + wx) * s.Z, (1 - 2*(xx+yy)) * s.Z, p.Z},
		{0, 0, 0, 1},
	}
}

func (m Mat4) Mul(n Mat4) Mat4 {
	var out Mat4
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			out[r][c] = m[r][0]*n[0][c] + m[r][1]*n[1][c] +
				m[r][2]*n[2][c] + m[r][3]*n[3][c]
		}
	}
	return out
}

func (m Mat4) TransformPoint(p Vec3) Vec3 {
	return V(
		m[0][0]*p.X+m[0][1]*p.Y+m[0][2]*p.Z+m[0][3],
		m[1][0]*p.X+m[1][1]*p.Y+m[1][2]*p.Z+m[1][3],
		m[2][0]*p.X+m[2][1]*p.Y+m[2][2]*p.Z+m[2][3],
	)
}
