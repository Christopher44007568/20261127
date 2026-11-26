package qem

import "math"

// Quadric is a symmetric 4x4 error matrix. For homogeneous point p=(x,y,z,1),
// error is p^T Q p.
type Quadric struct {
	M [4][4]float64 `json:"m"`
}

func (q *Quadric) Add(b Quadric) {
	for i := 0; i < 4; i++ {
		for j := i; j < 4; j++ {
			v := q.M[i][j] + b.M[i][j]
			q.M[i][j] = v
			q.M[j][i] = v
		}
	}
}

func (q Quadric) Added(b Quadric) Quadric {
	out := q
	out.Add(b)
	return out
}

func (q *Quadric) Scale(s float64) {
	for i := 0; i < 4; i++ {
		for j := 0; j < 4; j++ {
			q.M[i][j] *= s
		}
	}
}

// PlaneQuadric builds weight * p p^T for plane n.X*x+n.Y*y+n.Z*z+d=0.
// The plane normal should be unit length.
func PlaneQuadric(n Vec3, d, weight float64) Quadric {
	p := [4]float64{n.X, n.Y, n.Z, d}
	var q Quadric
	for i := 0; i < 4; i++ {
		for j := i; j < 4; j++ {
			v := weight * p[i] * p[j]
			q.M[i][j] = v
			q.M[j][i] = v
		}
	}
	return q
}

func (q Quadric) Eval(p Vec3) float64 {
	h := [4]float64{p.X, p.Y, p.Z, 1}
	sum := 0.0
	for i := 0; i < 4; i++ {
		row := 0.0
		for j := 0; j < 4; j++ {
			row += q.M[i][j] * h[j]
		}
		sum += h[i] * row
	}
	if math.IsNaN(sum) || sum < 0 && sum > -1e-8 {
		return 0
	}
	return sum
}

// Optimal solves the 3x3 linear system for minimum quadric error. Singular
// quadrics return false and callers try endpoints and midpoint.
func (q Quadric) Optimal() (Vec3, bool) {
	a := [3][4]float64{
		{q.M[0][0], q.M[0][1], q.M[0][2], -q.M[0][3]},
		{q.M[1][0], q.M[1][1], q.M[1][2], -q.M[1][3]},
		{q.M[2][0], q.M[2][1], q.M[2][2], -q.M[2][3]},
	}
	for col := 0; col < 3; col++ {
		pivot := col
		for r := col + 1; r < 3; r++ {
			if math.Abs(a[r][col]) > math.Abs(a[pivot][col]) {
				pivot = r
			}
		}
		if math.Abs(a[pivot][col]) < 1e-12 {
			return Vec3{}, false
		}
		a[col], a[pivot] = a[pivot], a[col]
		for r := 0; r < 3; r++ {
			if r == col {
				continue
			}
			f := a[r][col] / a[col][col]
			for c := col; c < 4; c++ {
				a[r][c] -= f * a[col][c]
			}
		}
	}
	p := V(a[0][3]/a[0][0], a[1][3]/a[1][1], a[2][3]/a[2][2])
	if !p.finite() {
		return Vec3{}, false
	}
	return p, true
}

func canonicalTriKey(v [3]int) [3]int {
	if v[0] > v[1] {
		v[0], v[1] = v[1], v[0]
	}
	if v[1] > v[2] {
		v[1], v[2] = v[2], v[1]
	}
	if v[0] > v[1] {
		v[0], v[1] = v[1], v[0]
	}
	return v
}
