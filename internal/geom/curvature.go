// Package geom estimates discrete differential quantities (curvature) on
// triangle meshes using only the standard library.
package geom

import (
	"math"

	"featurelines/internal/linalg"
	"featurelines/internal/mesh"
	"featurelines/internal/topo"
)

// Vec3 is the local vector alias.
type Vec3 = linalg.Vec3

// Curvature computes a cotangent-weight Laplace-Beltrami mean-curvature
// magnitude per vertex. The value has units 1/length; it is a stable scalar
// used for *relative* curvature differences, not an exact H estimate.
//
// Formula:  L_i = sum_j w_ij (p_j - p_i),  w_ij = cot alpha + cot beta,
// and kappa_i = |L_i| / (2 A_i), with A_i one third of incident areas.
func Curvature(m *mesh.Mesh, tp *topo.Topology) []float64 {
	n := len(m.V)
	lap := make([]Vec3, n)
	weight := make([]map[int]float64, n)
	area := make([]float64, n)

	for i := range weight {
		weight[i] = map[int]float64{}
	}

	addCotan := func(opposite int, u, v int) {
		// Angle at "opposite" between (u-opposite) and (v-opposite).
		e1 := m.V[u].Sub(m.V[opposite])
		e2 := m.V[v].Sub(m.V[opposite])
		cot := cotAngle(e1, e2)
		weight[u][v] += cot
		weight[v][u] += cot
	}

	for _, t := range m.T {
		a, b, c := t[0], t[1], t[2]
		// Triangle area (half cross product length).
		ar := m.V[b].Sub(m.V[a]).Cross(m.V[c].Sub(m.V[a])).Len() * 0.5
		if ar < 1e-12 {
			continue
		}
		third := ar / 3
		area[a] += third
		area[b] += third
		area[c] += third

		addCotan(a, b, c) // angle at a, opposite edge b-c
		addCotan(b, c, a)
		addCotan(c, a, b)
	}

	for i := 0; i < n; i++ {
		var l Vec3
		for j, w := range weight[i] {
			// Clamp pathological negative cotangent weights from obtuse
			// triangles to a small floor; keeps estimates bounded.
			if w < 0 {
				w = 0
			}
			l = l.Add(m.V[j].Sub(m.V[i]).Scale(w))
		}
		lap[i] = l
	}

	kappa := make([]float64, n)
	for i := 0; i < n; i++ {
		if area[i] > 1e-12 {
			kappa[i] = lap[i].Len() / (2 * area[i])
		}
	}
	return kappa
}

// cotAngle returns cot(theta) for the angle between e1 and e2.
func cotAngle(e1, e2 linalg.Vec3) float64 {
	cross := e1.Cross(e2).Len()
	dot := e1.Dot(e2)
	if cross < 1e-15 {
		if dot > 0 {
			return 1e6
		}
		return -1e6
	}
	return dot / cross
}

// RobustScale returns a normalization scale for curvature values: a high
// percentile over *all* vertices (zeros included), which represents the
// typical strongly-bent region and ignores isolated estimation spikes.
func RobustScale(kappa []float64) float64 {
	if len(kappa) == 0 {
		return 1
	}
	all := append([]float64{}, kappa...)
	sortFloat64s(all)
	q95 := all[int(0.95*float64(len(all)-1))]
	q80 := all[int(0.80*float64(len(all)-1))]
	scale := math.Max(q95, q80*1.5)
	if scale < 1e-9 {
		return 1
	}
	return scale
}

// FeatureScale is a topology-aware normalization: a high percentile of
// curvature at non-boundary vertices, so boundary-estimation spikes cannot
// compress real curvature contrasts.
func FeatureScale(kappa []float64, tp *topo.Topology) float64 {
	bnd := map[int]bool{}
	if tp != nil {
		for _, e := range tp.Edges {
			if e.Boundary() {
				bnd[e.A] = true
				bnd[e.B] = true
			}
		}
	}
	vals := make([]float64, 0, len(kappa))
	for i, k := range kappa {
		if k < 0 || bnd[i] {
			continue
		}
		vals = append(vals, k)
	}
	if len(vals) < 16 {
		return RobustScale(kappa)
	}
	sortFloat64s(vals)
	q98 := vals[int(0.98*float64(len(vals)-1))]
	if q98 < 1e-9 {
		return RobustScale(kappa)
	}
	return q98
}

func sortFloat64s(a []float64) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}
