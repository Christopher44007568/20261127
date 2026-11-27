package main

import (
	"math"
	"sort"
)

type Signals struct {
	Curvature   []float64
	AO          []float64
	AORaw       []float64
	EdgeCurv    []float64
	EdgeAO      []float64
	EdgeAngle   []float64
	EdgeCost    []float64
	Protected   []bool
	FrontFacing []bool
}

func ComputeSignals(m *Mesh, tp *Topology, aoSamples int) *Signals {
	if aoSamples <= 0 {
		aoSamples = 16
	}
	s := &Signals{
		Curvature:   make([]float64, len(m.Pos)),
		AO:          make([]float64, len(m.Pos)),
		AORaw:       make([]float64, len(m.Pos)),
		EdgeCurv:    make([]float64, len(tp.Edge)),
		EdgeAO:      make([]float64, len(tp.Edge)),
		EdgeAngle:   make([]float64, len(tp.Edge)),
		EdgeCost:    make([]float64, len(tp.Edge)),
		Protected:   make([]bool, len(tp.Edge)),
		FrontFacing: make([]bool, len(tp.Edge)),
	}

	weights := make([]float64, len(tp.Edge))
	lap := make([]Vec3, len(m.Pos))
	for f, t := range m.Tri {
		ids := [3]int{t.A, t.B, t.C}
		for k := 0; k < 3; k++ {
			o := ids[k]
			a := ids[(k+1)%3]
			b := ids[(k+2)%3]
			va := m.Pos[a].Sub(m.Pos[o])
			vb := m.Pos[b].Sub(m.Pos[o])
			crossLen := va.Cross(vb).Len()
			cot := 0.0
			if crossLen > 1e-12 {
				cot = va.Dot(vb) / crossLen
			}
			cot = clampFloat(cot, -2.0, 8.0)
			eid, ok := tp.EdgeID(a, b)
			if ok {
				weights[eid] += cot * 0.5
			}
		}
		_ = f
	}
	for eid, e := range tp.Edge {
		w := math.Max(weights[eid], 0.025)
		delta := m.Pos[e.B].Sub(m.Pos[e.A])
		lap[e.A] = lap[e.A].Add(delta.Mul(w))
		lap[e.B] = lap[e.B].Add(delta.Mul(-w))
	}
	edgeScale := medianEdgeLength(m, tp)
	for i := range s.Curvature {
		deg := float64(len(tp.VtxEdges[i]))
		if deg > 0 {
			s.Curvature[i] = lap[i].Len() / deg / math.Max(edgeScale, 1e-6)
		}
	}
	normalizeRobust(s.Curvature, 0.90)

	for eid, e := range tp.Edge {
		s.EdgeCurv[eid] = clampFloat(0.5*(s.Curvature[e.A]+s.Curvature[e.B]), 0, 1)
		if len(tp.EdgeFaces[eid]) == 2 {
			n1 := m.FaceNormal(tp.EdgeFaces[eid][0])
			n2 := m.FaceNormal(tp.EdgeFaces[eid][1])
			s.EdgeAngle[eid] = math.Acos(clampFloat(n1.Dot(n2), -1, 1)) / math.Pi
		}
	}

	bvh := BuildBVH(m)
	diag := bboxDiagonal(m)
	maxDist := diag * 0.24
	for i := range m.Pos {
		s.AORaw[i] = vertexOcclusion(m, bvh, i, aoSamples, maxDist)
	}
	smoothed := smoothVertexScalar(tp, s.AORaw, 3)
	copy(s.AO, smoothed)
	normalizeRobust(s.AO, 0.92)
	for eid, e := range tp.Edge {
		s.EdgeAO[eid] = 0.5 * (s.AO[e.A] + s.AO[e.B])
		mid := m.Pos[e.A].Add(m.Pos[e.B]).Mul(0.5)
		s.Protected[eid] = IsProtectedFacePoint(mid)
		s.FrontFacing[eid] = IsVisibleFrontPoint(mid)
		curv := clampFloat(0.65*s.EdgeCurv[eid]+0.35*s.EdgeAngle[eid], 0, 1)
		cost := 0.30 + 1.15*(1-curv) + 1.05*(1-s.EdgeAO[eid])
		if s.Protected[eid] {
			cost = math.Inf(1)
		}
		s.EdgeCost[eid] = cost
	}
	return s
}

func smoothVertexScalar(tp *Topology, values []float64, passes int) []float64 {
	cur := append([]float64(nil), values...)
	for pass := 0; pass < passes; pass++ {
		next := make([]float64, len(cur))
		for v := range tp.Mesh.Pos {
			sum := cur[v] * 1.5
			n := 1.5
			for _, eid := range tp.VtxEdges[v] {
				e := tp.Edge[eid]
				to := e.A
				if to == v {
					to = e.B
				}
				sum += cur[to]
				n++
			}
			next[v] = sum / n
		}
		cur = next
	}
	return cur
}

func IsProtectedFacePoint(p Vec3) bool {
	return p.Y > -0.84 && p.Y < 0.94 && p.Z > -0.13 && math.Abs(p.X) < 0.66
}

func IsVisibleFrontPoint(p Vec3) bool {
	return p.Y > -0.82 && p.Y < 0.82 && p.Z > 0.08 && math.Abs(p.X) < 0.60
}

func vertexOcclusion(m *Mesh, bvh *BVH, vid, samples int, maxDist float64) float64 {
	n := m.Norm[vid]
	if n.Len() < 0.1 {
		n = Vec3{0, 0, 1}
	}
	tangent := Vec3{1, 0, 0}
	if math.Abs(n.Dot(tangent)) > 0.9 {
		tangent = Vec3{0, 0, 1}
	}
	t := tangent.Cross(n).Normalized()
	b := n.Cross(t).Normalized()
	origin := m.Pos[vid].Add(n.Mul(0.008))
	occ := 0.0
	for k := 0; k < samples; k++ {
		r1 := hash01(uint64(vid*131 + k*17 + 1))
		r2 := float64(k+1) / float64(samples+1)
		phi := 2 * math.Pi * float64(k) * 0.61803398875
		rad := math.Sqrt(r1)
		hemiX := rad * math.Cos(phi)
		hemiY := rad * math.Sin(phi)
		hemiZ := math.Sqrt(math.Max(0, 1-r1))
		dir := t.Mul(hemiX).Add(b.Mul(hemiY)).Add(n.Mul(hemiZ)).Normalized()
		hit, dist := bvh.Occluded(origin, dir, maxDist)
		if hit {
			near := clampFloat(1-dist/maxDist, 0, 1)
			occ += 0.35 + 0.65*near*near
		}
		_ = r2
	}
	return clampFloat(occ/float64(samples)*1.45, 0, 1)
}

func hash01(n uint64) float64 {
	n ^= n >> 33
	n *= 0xff51afd7ed558ccd
	n ^= n >> 33
	n *= 0xc4ceb9fe1a85ec53
	n ^= n >> 33
	return float64(n%100000) / 100000.0
}

func medianEdgeLength(m *Mesh, tp *Topology) float64 {
	lens := make([]float64, 0, len(tp.Edge))
	for _, e := range tp.Edge {
		lens = append(lens, m.Pos[e.A].Sub(m.Pos[e.B]).Len())
	}
	sort.Float64s(lens)
	if len(lens) == 0 {
		return 1
	}
	return lens[len(lens)/2]
}

func bboxDiagonal(m *Mesh) float64 {
	mn, mx := m.Bounds()
	return mx.Sub(mn).Len()
}

func normalizeRobust(v []float64, q float64) {
	xs := append([]float64(nil), v...)
	sort.Float64s(xs)
	scale := 1.0
	if len(xs) > 0 {
		scale = xs[int(float64(len(xs)-1)*q)]
	}
	if scale < 1e-8 {
		scale = 1
	}
	for i := range v {
		v[i] = clampFloat(v[i]/scale, 0, 1)
	}
}
