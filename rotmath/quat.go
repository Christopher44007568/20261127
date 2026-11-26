package rotmath

import "math"

// Quat is a Hamilton quaternion q = W + Xi + Yj + Zk representing a
// rotation. Quaternions q and -q describe the same orientation.
type Quat struct {
	W, X, Y, Z float64
}

// Euler holds intrinsic ZYX (yaw about Z, pitch about Y, roll about X)
// angles in radians.
type Euler struct {
	Roll  float64
	Pitch float64
	Yaw   float64
}

// FromEuler converts intrinsic ZYX Euler angles to a unit quaternion.
func FromEuler(e Euler) Quat {
	cr := math.Cos(e.Roll / 2)
	sr := math.Sin(e.Roll / 2)
	cp := math.Cos(e.Pitch / 2)
	sp := math.Sin(e.Pitch / 2)
	cy := math.Cos(e.Yaw / 2)
	sy := math.Sin(e.Yaw / 2)

	return Quat{
		W: cr*cp*cy + sr*sp*sy,
		X: sr*cp*cy - cr*sp*sy,
		Y: cr*sp*cy + sr*cp*sy,
		Z: cr*cp*sy - sr*sp*cy,
	}
}

// ToEuler converts the quaternion back to intrinsic ZYX Euler angles.
func (q Quat) ToEuler() Euler {
	m := q.ToMatrix()
	// Element layout produced by EulerMatrix:
	//   -m[2][0] = sin(pitch)
	//   m[2][1] = cos(pitch)*sin(roll), m[2][2] = cos(pitch)*cos(roll)
	//   m[1][0] = cos(pitch)*sin(yaw),  m[0][0] = cos(pitch)*cos(yaw)
	sinPitch := -m[2][0]
	pitch := math.Asin(clamp(sinPitch, -1, 1))
	var roll, yaw float64
	if cosPitch := math.Cos(pitch); math.Abs(cosPitch) > 1e-9 {
		roll = math.Atan2(m[2][1], m[2][2])
		yaw = math.Atan2(m[1][0], m[0][0])
	} else {
		// Gimbal lock: the yaw/roll split is unobservable; pin yaw.
		roll = math.Atan2(-m[0][1], m[1][1])
	}
	return Euler{Roll: roll, Pitch: pitch, Yaw: yaw}
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// Norm returns the quaternion magnitude.
func (q Quat) Norm() float64 {
	return math.Sqrt(q.W*q.W + q.X*q.X + q.Y*q.Y + q.Z*q.Z)
}

// Normalized returns the unit quaternion. A near-zero quaternion returns
// the identity and ok=false so callers can flag the degenerate sample.
func (q Quat) Normalized() (Quat, bool) {
	n := q.Norm()
	if n < 1e-12 {
		return Quat{W: 1}, false
	}
	return Quat{q.W / n, q.X / n, q.Y / n, q.Z / n}, true
}

// Dot is the 4D inner product.
func (q Quat) Dot(o Quat) float64 {
	return q.W*o.W + q.X*o.X + q.Y*o.Y + q.Z*o.Z
}

// Neg flips all components. The rotation is unchanged.
func (q Quat) Neg() Quat {
	return Quat{-q.W, -q.X, -q.Y, -q.Z}
}

// Mul is the Hamilton product, applying o after q: r = q * o.
func (q Quat) Mul(o Quat) Quat {
	return Quat{
		W: q.W*o.W - q.X*o.X - q.Y*o.Y - q.Z*o.Z,
		X: q.W*o.X + q.X*o.W + q.Y*o.Z - q.Z*o.Y,
		Y: q.W*o.Y - q.X*o.Z + q.Y*o.W + q.Z*o.X,
		Z: q.W*o.Z + q.X*o.Y - q.Y*o.X + q.Z*o.W,
	}
}

// Conj returns the conjugate (inverse for unit quaternions).
func (q Quat) Conj() Quat {
	return Quat{W: q.W, X: -q.X, Y: -q.Y, Z: -q.Z}
}

// RotationAngle returns the unsigned rotation angle in [0, pi]. atan2 is
// used instead of acos so the angle stays accurate down to zero rotation.
func (q Quat) RotationAngle() float64 {
	vec := math.Sqrt(q.X*q.X + q.Y*q.Y + q.Z*q.Z)
	return 2 * math.Atan2(vec, math.Abs(q.W))
}

// Slerp performs spherical linear interpolation along the shortest arc.
// The sign of b is flipped when needed, and near-parallel quaternions use
// normalized linear interpolation to stay numerically stable.
func Slerp(a, b Quat, u float64) Quat {
	dot := a.Dot(b)
	if dot < 0 {
		b = b.Neg()
		dot = -dot
	}
	if dot > 0.9995 {
		// Very close: nlerp is numerically safer and the angular gap is
		// tiny, so the constant-speed error is negligible.
		q, _ := nlerpRaw(a, b, u).Normalized()
		return q
	}
	theta0 := math.Acos(clamp(dot, -1, 1))
	sinTheta0 := math.Sin(theta0)
	theta := theta0 * u
	sinTheta := math.Sin(theta)
	s0 := math.Cos(theta) - dot*sinTheta/sinTheta0
	s1 := sinTheta / sinTheta0
	return Quat{
		W: s0*a.W + s1*b.W,
		X: s0*a.X + s1*b.X,
		Y: s0*a.Y + s1*b.Y,
		Z: s0*a.Z + s1*b.Z,
	}
}

// NlerpRaw interpolates without any shortest-arc sign correction. It
// deliberately mirrors a naive renderer that trusts stored quaternion
// signs: antipodal endpoints produce a zero-length blend at u=0.5.
func NlerpRaw(a, b Quat, u float64) (Quat, bool) {
	blended := nlerpRaw(a, b, u)
	_, ok := blended.Normalized()
	return blended, ok
}

func nlerpRaw(a, b Quat, u float64) Quat {
	return Quat{
		W: (1-u)*a.W + u*b.W,
		X: (1-u)*a.X + u*b.X,
		Y: (1-u)*a.Y + u*b.Y,
		Z: (1-u)*a.Z + u*b.Z,
	}
}
