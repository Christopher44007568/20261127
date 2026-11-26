package main

import "math"

// Vec3 is a 3D vector in either bind space or world space.
type Vec3 struct{ X, Y, Z float64 }

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
func (v Vec3) Norm() Vec3 {
	n := v.Len()
	if n < 1e-12 {
		return Vec3{}
	}
	return v.Scale(1 / n)
}

// Mat4 is a row-major 4x4 transform.
type Mat4 struct{ M [16]float64 }

func Identity() Mat4 {
	var m Mat4
	m.M[0], m.M[5], m.M[10], m.M[15] = 1, 1, 1, 1
	return m
}

func Translate(t Vec3) Mat4 {
	m := Identity()
	m.M[3], m.M[7], m.M[11] = t.X, t.Y, t.Z
	return m
}

// RotY rotates about the vertical Y axis (used for the slow body turn).
func RotY(a float64) Mat4 {
	c, s := math.Cos(a), math.Sin(a)
	return Mat4{M: [16]float64{
		c, 0, s, 0,
		0, 1, 0, 0,
		-s, 0, c, 0,
		0, 0, 0, 1,
	}}
}

// RotZ rotates about the depth-facing Z axis (used for arm waving in
// the screen plane).
func RotZ(a float64) Mat4 {
	c, s := math.Cos(a), math.Sin(a)
	return Mat4{M: [16]float64{
		c, -s, 0, 0,
		s, c, 0, 0,
		0, 0, 1, 0,
		0, 0, 0, 1,
	}}
}

// RotX rotates about the X axis (used for small spine lean).
func RotX(a float64) Mat4 {
	c, s := math.Cos(a), math.Sin(a)
	return Mat4{M: [16]float64{
		1, 0, 0, 0,
		0, c, -s, 0,
		0, s, c, 0,
		0, 0, 0, 1,
	}}
}

func (a Mat4) Mul(b Mat4) Mat4 {
	var r Mat4
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			s := 0.0
			for k := 0; k < 4; k++ {
				s += a.M[i*4+k] * b.M[k*4+j]
			}
			r.M[i*4+j] = s
		}
	}
	return r
}

func (a Mat4) TransformPoint(v Vec3) Vec3 {
	m := a.M
	x := m[0]*v.X + m[1]*v.Y + m[2]*v.Z + m[3]
	y := m[4]*v.X + m[5]*v.Y + m[6]*v.Z + m[7]
	z := m[8]*v.X + m[9]*v.Y + m[10]*v.Z + m[11]
	return Vec3{x, y, z}
}

// Perspective returns an OpenGL-style perspective matrix (NDC z in [-1,1]).
func Perspective(fovyRad, aspect, near, far float64) Mat4 {
	f := 1 / math.Tan(fovyRad/2)
	nf := 1 / (near - far)
	return Mat4{M: [16]float64{
		f / aspect, 0, 0, 0,
		0, f, 0, 0,
		0, 0, (far + near) * nf, 2 * far * near * nf,
		0, 0, -1, 0,
	}}
}

// LookAt builds a view matrix for a camera at eye looking at center.
func LookAt(eye, center, up Vec3) Mat4 {
	f := center.Sub(eye).Norm()
	s := f.Cross(up).Norm()
	u := s.Cross(f)
	return Mat4{M: [16]float64{
		s.X, s.Y, s.Z, -s.Dot(eye),
		u.X, u.Y, u.Z, -u.Dot(eye),
		-f.X, -f.Y, -f.Z, f.Dot(eye),
		0, 0, 0, 1,
	}}
}

// Invert returns the inverse. The matrix is row-major (element m[i][j]
// is M[4i+j]); cofactors are assembled directly in that same layout so
// the result cannot accidentally transpose.
func (a Mat4) Invert() Mat4 {
	// Row-major Gauss-Jordan elimination on an augmented 4x8 matrix.
	var g [4][8]float64
	for r := 0; r < 4; r++ {
		for c := 0; c < 4; c++ {
			g[r][c] = a.M[r*4+c]
		}
		g[r][r+4] = 1
	}
	for col := 0; col < 4; col++ {
		pivot := col
		for r := col + 1; r < 4; r++ {
			if math.Abs(g[r][col]) > math.Abs(g[pivot][col]) {
				pivot = r
			}
		}
		g[col], g[pivot] = g[pivot], g[col]
		p := g[col][col]
		if math.Abs(p) < 1e-15 {
			return Mat4{}
		}
		for c := 0; c < 8; c++ {
			g[col][c] /= p
		}
		for r := 0; r < 4; r++ {
			if r == col {
				continue
			}
			factor := g[r][col]
			for c := 0; c < 8; c++ {
				g[r][c] -= factor * g[col][c]
			}
		}
	}
	var r Mat4
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			r.M[i*4+j] = g[i][j+4]
		}
	}
	return r
}
