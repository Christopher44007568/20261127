package main

import (
	"math"
)

type SparseEntry struct {
	Col int
	V   float64
}

type SparseRow struct {
	Entries []SparseEntry
	B       float64
}

type LSCMResult struct {
	UV         []Vec2
	Iterations int
	Residual   float64
}

func BuildWedges(tp *Topology, seam []bool) ([][3]int, []int) {
	cornerCount := len(tp.Mesh.Tri) * 3
	dsu := NewDSU(cornerCount)
	for eid, incs := range tp.EdgeInc {
		if seam != nil && eid < len(seam) && seam[eid] {
			continue
		}
		if len(incs) != 2 {
			continue
		}
		a0 := incs[0].Face*3 + incs[0].Corners[0]
		b0 := incs[1].Face*3 + incs[1].Corners[0]
		a1 := incs[0].Face*3 + incs[0].Corners[1]
		b1 := incs[1].Face*3 + incs[1].Corners[1]
		dsu.Union(a0, b0)
		dsu.Union(a1, b1)
	}

	rootWedge := map[int]int{}
	faceWedge := make([][3]int, len(tp.Mesh.Tri))
	var wedgeVertex []int
	for f, t := range tp.Mesh.Tri {
		verts := [3]int{t.A, t.B, t.C}
		for k := 0; k < 3; k++ {
			corner := f*3 + k
			root := dsu.Find(corner)
			w, ok := rootWedge[root]
			if !ok {
				w = len(wedgeVertex)
				rootWedge[root] = w
				wedgeVertex = append(wedgeVertex, verts[k])
			}
			faceWedge[f][k] = w
		}
	}
	return faceWedge, wedgeVertex
}

func SolveIslandLSCM(m *Mesh, faceWedge [][3]int, wedgeVertex []int, faces []int) (LSCMResult, error) {
	used := map[int]bool{}
	for _, f := range faces {
		for _, w := range faceWedge[f] {
			used[w] = true
		}
	}
	wedges := make([]int, 0, len(used))
	for w := range used {
		wedges = append(wedges, w)
	}
	if len(wedges) < 3 {
		return LSCMResult{}, mathError("island has too few wedges")
	}

	pin0, pin1 := farthestWedges(m, wedgeVertex, wedges)
	pinDistance := m.Pos[wedgeVertex[pin1]].Sub(m.Pos[wedgeVertex[pin0]]).Len()
	if pinDistance < 1e-8 {
		pinDistance = 1
	}
	pinUV := map[int]Vec2{
		pin0: {0, 0},
		pin1: {pinDistance, 0},
	}

	freeIndex := make([]int, len(wedgeVertex))
	for i := range freeIndex {
		freeIndex[i] = -1
	}
	nFree := 0
	for _, w := range wedges {
		if _, pinned := pinUV[w]; !pinned {
			freeIndex[w] = nFree
			nFree++
		}
	}
	nVar := 2 * nFree

	var rows []SparseRow
	addCoeff := func(row *SparseRow, wedge, component int, coeff float64) {
		if math.Abs(coeff) < 1e-14 {
			return
		}
		if p, ok := pinUV[wedge]; ok {
			val := p.X
			if component == 1 {
				val = p.Y
			}
			row.B -= coeff * val
			return
		}
		idx := freeIndex[wedge]
		if idx < 0 {
			return
		}
		row.Entries = append(row.Entries, SparseEntry{Col: 2*idx + component, V: coeff})
	}

	for _, f := range faces {
		t := m.Tri[f]
		p0 := m.Pos[t.A]
		p1 := m.Pos[t.B]
		p2 := m.Pos[t.C]
		e1 := p1.Sub(p0)
		e2 := p2.Sub(p0)
		x2 := e1.Len()
		if x2 < 1e-10 {
			continue
		}
		e1n := e1.Mul(1 / x2)
		x3 := e2.Dot(e1n)
		y3sq := e2.Dot(e2) - x3*x3
		if y3sq < 1e-14 {
			y3sq = 1e-14
		}
		y3 := math.Sqrt(y3sq)
		den := x2 * y3
		weight := math.Sqrt(math.Max(den*0.5, 1e-12))

		ux := [3]float64{-1 / x2, 1 / x2, 0}
		uy := [3]float64{(x3 - x2) / den, -x3 / den, x2 / den}
		ws := faceWedge[f]

		var r1 SparseRow
		var r2 SparseRow
		for k := 0; k < 3; k++ {
			addCoeff(&r1, ws[k], 0, weight*ux[k])
			addCoeff(&r1, ws[k], 1, -weight*uy[k])
			addCoeff(&r2, ws[k], 0, weight*uy[k])
			addCoeff(&r2, ws[k], 1, weight*ux[k])
		}
		rows = append(rows, r1, r2)
	}

	x, iters, res := solveNormalCG(rows, nVar)
	uv := make([]Vec2, len(wedgeVertex))
	for w := range wedgeVertex {
		if p, ok := pinUV[w]; ok {
			uv[w] = p
		} else if idx := freeIndex[w]; idx >= 0 {
			uv[w] = Vec2{x[2*idx], x[2*idx+1]}
		}
	}
	return LSCMResult{UV: uv, Iterations: iters, Residual: res}, nil
}

func farthestWedges(m *Mesh, wedgeVertex []int, wedges []int) (int, int) {
	best := -1.0
	a := wedges[0]
	b := wedges[len(wedges)-1]
	for i := 0; i < len(wedges); i++ {
		pi := m.Pos[wedgeVertex[wedges[i]]]
		for j := i + 1; j < len(wedges); j++ {
			pj := m.Pos[wedgeVertex[wedges[j]]]
			d := pi.Sub(pj).Len()
			if d > best {
				best = d
				a = wedges[i]
				b = wedges[j]
			}
		}
	}
	return a, b
}

type simpleError string

func (e simpleError) Error() string { return string(e) }

func mathError(s string) error { return simpleError(s) }

func solveNormalCG(rows []SparseRow, n int) ([]float64, int, float64) {
	diag := make([]float64, n)
	rhs := make([]float64, n)
	for _, row := range rows {
		for _, e := range row.Entries {
			diag[e.Col] += e.V * e.V
			rhs[e.Col] += e.V * row.B
		}
	}
	for i := range diag {
		if diag[i] < 1e-12 {
			diag[i] = 1
		}
	}

	ax := func(x []float64) []float64 {
		out := make([]float64, len(rows))
		for r, row := range rows {
			s := 0.0
			for _, e := range row.Entries {
				s += e.V * x[e.Col]
			}
			out[r] = s
		}
		return out
	}
	at := func(y []float64) []float64 {
		out := make([]float64, n)
		for r, row := range rows {
			for _, e := range row.Entries {
				out[e.Col] += e.V * y[r]
			}
		}
		return out
	}
	ata := func(x []float64) []float64 {
		return at(ax(x))
	}

	x := make([]float64, n)
	r := append([]float64(nil), rhs...)
	z := make([]float64, n)
	for i := range r {
		z[i] = r[i] / diag[i]
	}
	p := append([]float64(nil), z...)
	rz := dot(r, z)
	rhsNorm := math.Sqrt(dot(rhs, rhs))
	if rhsNorm < 1e-12 {
		rhsNorm = 1
	}
	iters := 0
	for it := 0; it < 4000; it++ {
		iters = it + 1
		ap := ata(p)
		pap := dot(p, ap)
		if pap <= 1e-30 {
			break
		}
		alpha := rz / pap
		for i := range x {
			x[i] += alpha * p[i]
			r[i] -= alpha * ap[i]
		}
		rnorm := math.Sqrt(dot(r, r))
		if rnorm/rhsNorm < 1e-9 || rnorm < 1e-10 {
			return x, iters, rnorm
		}
		for i := range z {
			z[i] = r[i] / diag[i]
		}
		rzNew := dot(r, z)
		beta := rzNew / rz
		for i := range p {
			p[i] = z[i] + beta*p[i]
		}
		rz = rzNew
	}
	res := math.Sqrt(dot(r, r))
	return x, iters, res
}

func dot(a, b []float64) float64 {
	s := 0.0
	for i := range a {
		s += a[i] * b[i]
	}
	return s
}
