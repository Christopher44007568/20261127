package main

import (
	"math"
	"sort"
)

type ErrorStats struct {
	Count int     `json:"count"`
	MAE   float64 `json:"mae"`
	RMSE  float64 `json:"rmse"`
	P50   float64 `json:"p50"`
	P95   float64 `json:"p95"`
	Max   float64 `json:"max"`
	Mean  float64 `json:"mean"`
}

func statsFromValues(values []float64) ErrorStats {
	out := ErrorStats{Count: len(values)}
	if len(values) == 0 {
		return out
	}
	abs := make([]float64, len(values))
	sum := 0.0
	sum2 := 0.0
	max := 0.0
	for i, v := range values {
		av := math.Abs(v)
		abs[i] = av
		sum += av
		sum2 += v * v
		max = math.Max(max, av)
	}
	sorted := append([]float64(nil), abs...)
	sort.Float64s(sorted)
	out.MAE = sum / float64(len(values))
	out.RMSE = math.Sqrt(sum2 / float64(len(values)))
	out.P50 = quantileSorted(sorted, 0.50)
	out.P95 = quantileSorted(sorted, 0.95)
	out.Max = max
	out.Mean = averageFloat(values)
	return out
}

func BoundaryVertexMask(m *Mesh, growRings int) []bool {
	mask := make([]bool, len(m.P))
	counts := m.EdgeCounts()
	for key, count := range counts {
		if count == 1 {
			a, b := edgeKeyToAB(key)
			mask[a] = true
			mask[b] = true
		}
	}
	for ring := 0; ring < growRings; ring++ {
		next := append([]bool(nil), mask...)
		for _, t := range m.Tri {
			if mask[t[0]] || mask[t[1]] || mask[t[2]] {
				next[t[0]] = true
				next[t[1]] = true
				next[t[2]] = true
			}
		}
		mask = next
	}
	return mask
}

func EstimateMeshCurvature(m *Mesh, k int) []float64 {
	out := make([]float64, len(m.P))
	if k <= 0 {
		k = 30
	}
	if len(m.P) < 8 {
		return out
	}
	tree := NewKDTree(m.P)

	for i, p := range m.P {
		n := V3{0, 0, 1}
		if i < len(m.N) && Norm(m.N[i]) > 0.5 {
			n = m.N[i]
		}
		ns := tree.KNN(p, minInt(k+1, len(m.P)))
		if len(ns) < 10 {
			continue
		}
		maxD := 0.0
		for _, nb := range ns {
			maxD = math.Max(maxD, math.Sqrt(nb.Dist2))
		}
		if maxD < 1e-9 {
			continue
		}
		ref := V3{0, 0, 1}
		if math.Abs(n.Z) > 0.9 {
			ref = V3{1, 0, 0}
		}
		u := Normalize(Cross(ref, n))
		v := Cross(n, u)

		var ata [6][6]float64
		var atb [6]float64
		used := 0
		for _, nb := range ns {
			q := m.P[nb.ID]
			r := q.Sub(p)
			d := math.Sqrt(nb.Dist2)
			w := tricubeWeight(d / (maxD * 1.05))
			if nb.ID == i {
				w = 4
			}
			x := Dot(r, u)
			y := Dot(r, v)
			z := Dot(r, n)
			basis := [6]float64{1, x, y, x * x, x * y, y * y}
			for row := 0; row < 6; row++ {
				atb[row] += w * basis[row] * z
				for col := 0; col < 6; col++ {
					ata[row][col] += w * basis[row] * basis[col]
				}
			}
			used++
		}
		if used < 10 || !solveLinear6(&ata, &atb) {
			continue
		}
		fx := atb[1]
		fy := atb[2]
		fxx := 2 * atb[3]
		fxy := atb[4]
		fyy := 2 * atb[5]
		den := 1 + fx*fx + fy*fy
		out[i] = ((1+fy*fy)*fxx - 2*fx*fy*fxy + (1+fx*fx)*fyy) /
			(2 * math.Pow(den, 1.5))
	}
	return out
}

func tricubeWeight(t float64) float64 {
	t = math.Abs(t)
	if t >= 1 {
		return 0
	}
	u := 1 - t*t*t
	return u * u * u
}

type CurvatureScore struct {
	Error    []float64
	Valid    []bool
	Stats    ErrorStats
	Interior int
}

func ScoreDemoCurvature(m *Mesh, scene *DemoScene, boundaryMask []bool) CurvatureScore {
	estimated := EstimateMeshCurvature(m, 32)
	res := CurvatureScore{Error: make([]float64, len(m.P)), Valid: make([]bool, len(m.P))}
	var signed []float64
	for i, p := range m.P {
		if i < len(boundaryMask) && boundaryMask[i] {
			continue
		}
		if p.X < scene.XMin+0.08 || p.X > scene.XMax-0.08 ||
			p.Y < scene.YMin+0.08 || p.Y > scene.YMax-0.08 ||
			scene.InHole(p.X, p.Y) {
			continue
		}
		truth := demoCurvature(p.X, p.Y)
		res.Error[i] = estimated[i] - truth
		res.Valid[i] = true
		signed = append(signed, res.Error[i])
	}
	res.Stats = statsFromValues(signed)
	res.Interior = len(signed)
	return res
}
