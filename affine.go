package main

import "math"

// Affine is a 2D affine transform for column vectors:
//
//	x' = A*x + B*y + TX
//	y' = C*x + D*y + TY
type Affine struct {
	A, B, C, D, TX, TY float64
}

func Identity() Affine {
	return Affine{A: 1, D: 1}
}

func Translate(tx, ty float64) Affine {
	return Affine{A: 1, D: 1, TX: tx, TY: ty}
}

// Rotate returns a counter-clockwise rotation in a y-up coordinate system.
func Rotate(theta float64) Affine {
	c, s := math.Cos(theta), math.Sin(theta)
	return Affine{A: c, B: -s, C: s, D: c}
}

func Scale(sx, sy float64) Affine {
	return Affine{A: sx, D: sy}
}

// MirrorX reflects across the local y axis (x maps to -x).
func MirrorX() Affine {
	return Affine{A: -1, D: 1}
}

// MirrorY reflects across the local x axis (y maps to -y).
func MirrorY() Affine {
	return Affine{A: 1, D: -1}
}

// MirrorAbout reflects across the lines x=pivotX and y=pivotY.
func MirrorAbout(pivotX, pivotY float64) Affine {
	return Translate(2*pivotX, 2*pivotY).Mul(MirrorX()).Mul(MirrorY())
}

// Mul returns m*n. The right-hand transform is applied first.
func (m Affine) Mul(n Affine) Affine {
	return Affine{
		A:  m.A*n.A + m.B*n.C,
		B:  m.A*n.B + m.B*n.D,
		C:  m.C*n.A + m.D*n.C,
		D:  m.C*n.B + m.D*n.D,
		TX: m.A*n.TX + m.B*n.TY + m.TX,
		TY: m.C*n.TX + m.D*n.TY + m.TY,
	}
}

func (m Affine) Apply(x, y float64) (float64, float64) {
	return m.A*x + m.B*y + m.TX, m.C*x + m.D*y + m.TY
}

func (m Affine) Determinant() float64 {
	return m.A*m.D - m.B*m.C
}

func (m Affine) Inverse() (Affine, bool) {
	det := m.Determinant()
	if math.Abs(det) < 1e-12 {
		return Affine{}, false
	}
	a := m.D / det
	b := -m.B / det
	c := -m.C / det
	d := m.A / det
	return Affine{
		A: a, B: b, C: c, D: d,
		TX: -(a*m.TX + b*m.TY),
		TY: -(c*m.TX + d*m.TY),
	}, true
}
