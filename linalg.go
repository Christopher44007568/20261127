package main

import "math"

// eigenSym3 returns eigenvalues in ascending order. Eigenvectors are returned
// as columns: vec[i][k] is component i of eigenvector k.
func eigenSym3(matrix [3][3]float64) ([3]float64, [3][3]float64) {
	a := matrix
	var v [3][3]float64
	for i := 0; i < 3; i++ {
		v[i][i] = 1
	}

	for iter := 0; iter < 60; iter++ {
		p, q := 0, 1
		maxOff := math.Abs(a[0][1])
		if math.Abs(a[0][2]) > maxOff {
			p, q = 0, 2
			maxOff = math.Abs(a[0][2])
		}
		if math.Abs(a[1][2]) > maxOff {
			p, q = 1, 2
			maxOff = math.Abs(a[1][2])
		}
		if maxOff < 1e-14 {
			break
		}

		app := a[p][p]
		aqq := a[q][q]
		apq := a[p][q]
		phi := 0.5 * math.Atan2(2*apq, aqq-app)
		c := math.Cos(phi)
		s := math.Sin(phi)

		a[p][p] = c*c*app - 2*s*c*apq + s*s*aqq
		a[q][q] = s*s*app + 2*s*c*apq + c*c*aqq
		a[p][q] = 0
		a[q][p] = 0

		for i := 0; i < 3; i++ {
			if i == p || i == q {
				continue
			}
			aip := a[i][p]
			aiq := a[i][q]
			a[i][p] = c*aip - s*aiq
			a[p][i] = a[i][p]
			a[i][q] = s*aip + c*aiq
			a[q][i] = a[i][q]
		}
		for i := 0; i < 3; i++ {
			vip := v[i][p]
			viq := v[i][q]
			v[i][p] = c*vip - s*viq
			v[i][q] = s*vip + c*viq
		}
	}

	idx := [3]int{0, 1, 2}
	vals := [3]float64{a[0][0], a[1][1], a[2][2]}
	if vals[idx[1]] < vals[idx[0]] {
		idx[0], idx[1] = idx[1], idx[0]
	}
	if vals[idx[2]] < vals[idx[1]] {
		idx[1], idx[2] = idx[2], idx[1]
	}
	if vals[idx[1]] < vals[idx[0]] {
		idx[0], idx[1] = idx[1], idx[0]
	}

	var outVals [3]float64
	var outVecs [3][3]float64
	for k := 0; k < 3; k++ {
		col := idx[k]
		outVals[k] = vals[col]
		for i := 0; i < 3; i++ {
			outVecs[i][k] = v[i][col]
		}
	}
	return outVals, outVecs
}

func solveLinear6(a *[6][6]float64, b *[6]float64) bool {
	for col := 0; col < 6; col++ {
		pivot := col
		best := math.Abs(a[col][col])
		for row := col + 1; row < 6; row++ {
			if v := math.Abs(a[row][col]); v > best {
				best = v
				pivot = row
			}
		}
		if best < 1e-14 {
			return false
		}
		if pivot != col {
			a[col], a[pivot] = a[pivot], a[col]
			b[col], b[pivot] = b[pivot], b[col]
		}
		div := a[col][col]
		for j := col; j < 6; j++ {
			a[col][j] /= div
		}
		b[col] /= div
		for row := 0; row < 6; row++ {
			if row == col {
				continue
			}
			factor := a[row][col]
			if math.Abs(factor) < 1e-15 {
				continue
			}
			for j := col; j < 6; j++ {
				a[row][j] -= factor * a[col][j]
			}
			b[row] -= factor * b[col]
		}
	}
	return true
}

func meanMedianDistance(points []V3, k int) (meanScore, medianScore, spacing float64) {
	if len(points) < 4 {
		return 0, 0, 0
	}
	tree := NewKDTree(points)
	scores := make([]float64, len(points))
	first := make([]float64, len(points))
	kk := k + 1
	for i, p := range points {
		ns := tree.KNN(p, kk)
		sum := 0.0
		count := 0
		for _, n := range ns {
			if n.ID == i || n.Dist2 == 0 {
				continue
			}
			d := math.Sqrt(n.Dist2)
			sum += d
			count++
			if count == 1 {
				first[i] = d
			}
		}
		if count > 0 {
			scores[i] = sum / float64(count)
		}
	}
	return averageFloat(scores), medianFloat(scores), medianFloat(first)
}

func averageFloat(xs []float64) float64 {
	if len(xs) == 0 {
		return 0
	}
	s := 0.0
	for _, x := range xs {
		s += x
	}
	return s / float64(len(xs))
}
