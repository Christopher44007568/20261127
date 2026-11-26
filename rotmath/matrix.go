package rotmath

import "math"

// Mat3 is a row-major 3x3 rotation matrix.
type Mat3 [3][3]float64

// ToMatrix converts a unit quaternion to its rotation matrix.
func (q Quat) ToMatrix() Mat3 {
	w, x, y, z := q.W, q.X, q.Y, q.Z
	return Mat3{
		{1 - 2*(y*y+z*z), 2 * (x*y - z*w), 2 * (x*z + y*w)},
		{2 * (x*y + z*w), 1 - 2*(x*x+z*z), 2 * (y*z - x*w)},
		{2 * (x*z - y*w), 2 * (y*z + x*w), 1 - 2*(x*x+y*y)},
	}
}

// EulerMatrix builds R = Rz(yaw) * Ry(pitch) * Rx(roll), the intrinsic
// ZYX convention matching FromEuler.
func EulerMatrix(e Euler) Mat3 {
	cr, sr := math.Cos(e.Roll), math.Sin(e.Roll)
	cp, sp := math.Cos(e.Pitch), math.Sin(e.Pitch)
	cy, sy := math.Cos(e.Yaw), math.Sin(e.Yaw)

	rx := Mat3{
		{1, 0, 0},
		{0, cr, -sr},
		{0, sr, cr},
	}
	ry := Mat3{
		{cp, 0, sp},
		{0, 1, 0},
		{-sp, 0, cp},
	}
	rz := Mat3{
		{cy, -sy, 0},
		{sy, cy, 0},
		{0, 0, 1},
	}
	return mul3(mul3(rz, ry), rx)
}

func mul3(a, b Mat3) Mat3 {
	var r Mat3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			r[i][j] = a[i][0]*b[0][j] + a[i][1]*b[1][j] + a[i][2]*b[2][j]
		}
	}
	return r
}

// OrthoError measures how far the matrix deviates from a true rotation
// matrix by returning max(||R^T R - I||_F, ||det(R) - 1||). A valid
// rotation stays at machine-epsilon scale (~1e-15).
func (m Mat3) OrthoError() float64 {
	var residual float64
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			var sum float64
			for k := 0; k < 3; k++ {
				sum += m[k][i] * m[k][j]
			}
			want := 0.0
			if i == j {
				want = 1
			}
			residual += (sum - want) * (sum - want)
		}
	}
	detErr := math.Abs(m.Det() - 1)
	frob := math.Sqrt(residual)
	if detErr > frob {
		return detErr
	}
	return frob
}

// Det returns the matrix determinant.
func (m Mat3) Det() float64 {
	return m[0][0]*(m[1][1]*m[2][2]-m[1][2]*m[2][1]) -
		m[0][1]*(m[1][0]*m[2][2]-m[1][2]*m[2][0]) +
		m[0][2]*(m[1][0]*m[2][1]-m[1][1]*m[2][0])
}

// AngularVelocity extracts the body-frame angular velocity from the
// relative rotation delta = R(t)^T R(t+dt): omega_body = angle/dt * axis.
// The sign of the relative quaternion is normalized to the real axis so
// that the reported angle never exceeds pi (shortest-arc convention).
func AngularVelocity(prev, next Mat3, dt float64) (omega [3]float64, angle float64) {
	rel := transpose(prev).mul(next)
	q := matrixToQuat(rel)
	if q.W < 0 {
		q = q.Neg()
	}
	angle = q.RotationAngle()
	if angle < 1e-12 {
		return [3]float64{0, 0, 0}, 0
	}
	axisScale := 1 / math.Sqrt(1-clamp(q.W*q.W, 0, 1))
	return [3]float64{
		angle * q.X * axisScale / dt,
		angle * q.Y * axisScale / dt,
		angle * q.Z * axisScale / dt,
	}, angle
}

func (m Mat3) mul(o Mat3) Mat3 {
	return mul3(m, o)
}

func transpose(m Mat3) Mat3 {
	var r Mat3
	for i := 0; i < 3; i++ {
		for j := 0; j < 3; j++ {
			r[i][j] = m[j][i]
		}
	}
	return r
}

// matrixToQuat converts a rotation matrix back to a unit quaternion
// using Shepperd's numerically stable branch selection.
func matrixToQuat(m Mat3) Quat {
	trace := m[0][0] + m[1][1] + m[2][2]
	switch {
	case trace > 0:
		s := math.Sqrt(trace+1) * 2
		return Quat{
			W: 0.25 * s,
			X: (m[2][1] - m[1][2]) / s,
			Y: (m[0][2] - m[2][0]) / s,
			Z: (m[1][0] - m[0][1]) / s,
		}
	case m[0][0] > m[1][1] && m[0][0] > m[2][2]:
		s := math.Sqrt(1+m[0][0]-m[1][1]-m[2][2]) * 2
		return Quat{
			W: (m[2][1] - m[1][2]) / s,
			X: 0.25 * s,
			Y: (m[0][1] + m[1][0]) / s,
			Z: (m[0][2] + m[2][0]) / s,
		}
	case m[1][1] > m[2][2]:
		s := math.Sqrt(1+m[1][1]-m[0][0]-m[2][2]) * 2
		return Quat{
			W: (m[0][2] - m[2][0]) / s,
			X: (m[0][1] + m[1][0]) / s,
			Y: 0.25 * s,
			Z: (m[1][2] + m[2][1]) / s,
		}
	default:
		s := math.Sqrt(1+m[2][2]-m[0][0]-m[1][1]) * 2
		return Quat{
			W: (m[1][0] - m[0][1]) / s,
			X: (m[0][2] + m[2][0]) / s,
			Y: (m[1][2] + m[2][1]) / s,
			Z: 0.25 * s,
		}
	}
}
