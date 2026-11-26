package main

import "math"

type QuadricInfo struct {
	MeanCurvature float64
	Normal        V3
}

func (g *FieldGrid) QuadricAt(q V3, radius float64) (QuadricInfo, bool) {
	if radius <= 0 {
		radius = g.opts.FitRadius * 1.2
	}
	ids := g.tree.RadiusSearch(q, radius)
	if len(ids) < 16 {
		return QuadricInfo{}, false
	}

	totalW := 0.0
	centroid := V3{}
	frameN := V3{}
	type sample struct {
		r V3
		w float64
	}
	samples := make([]sample, 0, len(ids))
	for _, id := range ids {
		p := g.points[id]
		d := Norm(q.Sub(p))
		w := wendlandWeight(d, radius)
		if w <= 1e-8 {
			continue
		}
		totalW += w
		centroid = centroid.Add(p.Mul(w))
		if id < len(g.normals) {
			frameN = frameN.Add(g.normals[id].Mul(w))
		}
		samples = append(samples, sample{r: p, w: w})
	}
	if totalW <= 1e-12 || len(samples) < 16 {
		return QuadricInfo{}, false
	}
	centroid = centroid.Div(totalW)
	n := Normalize(frameN)
	if Norm(n) < 0.5 {
		n, _ = g.localPCANormal(ids)
	}
	if Norm(n) < 0.5 {
		return QuadricInfo{}, false
	}
	ref := V3{0, 0, 1}
	if math.Abs(n.Z) > 0.92 {
		ref = V3{1, 0, 0}
	}
	u := Normalize(Cross(ref, n))
	if Norm(u) < 0.5 {
		return QuadricInfo{}, false
	}
	v := Cross(n, u)

	var ata [6][6]float64
	var atb [6]float64
	for _, s := range samples {
		r := s.r.Sub(centroid)
		x := Dot(r, u)
		y := Dot(r, v)
		z := Dot(r, n)
		basis := [6]float64{1, x, y, x * x, x * y, y * y}
		for row := 0; row < 6; row++ {
			atb[row] += s.w * basis[row] * z
			for col := 0; col < 6; col++ {
				ata[row][col] += s.w * basis[row] * basis[col]
			}
		}
	}
	if !solveLinear6(&ata, &atb) {
		return QuadricInfo{}, false
	}

	rq := q.Sub(centroid)
	xq := Dot(rq, u)
	yq := Dot(rq, v)
	fx := atb[1] + 2*atb[3]*xq + atb[4]*yq
	fy := atb[2] + atb[4]*xq + 2*atb[5]*yq
	fxx := 2 * atb[3]
	fxy := atb[4]
	fyy := 2 * atb[5]
	den := 1 + fx*fx + fy*fy
	h := ((1+fy*fy)*fxx - 2*fx*fy*fxy + (1+fx*fx)*fyy) /
		(2 * math.Pow(den, 1.5))
	if math.IsNaN(h) || math.IsInf(h, 0) || math.Abs(h) > 20 {
		return QuadricInfo{}, false
	}
	surfaceNormal := Normalize(n.Sub(u.Mul(fx)).Sub(v.Mul(fy)))
	if Dot(surfaceNormal, n) < 0 {
		surfaceNormal = surfaceNormal.Mul(-1)
	}
	return QuadricInfo{MeanCurvature: h, Normal: surfaceNormal}, true
}

func (g *FieldGrid) CurvatureAt(q V3, radius float64) (float64, bool) {
	info, ok := g.QuadricAt(q, radius)
	return info.MeanCurvature, ok
}

func ApplyFieldNormals(m *Mesh, g *FieldGrid, radius float64) int {
	if radius <= 0 {
		radius = g.opts.FitRadius * 1.05
	}
	if len(m.N) != len(m.P) {
		m.N = make([]V3, len(m.P))
	}
	applied := 0
	for i, p := range m.P {
		info, ok := g.QuadricAt(p, radius)
		if ok && Norm(info.Normal) > 0.5 {
			m.N[i] = info.Normal
			applied++
		} else if Norm(m.N[i]) < 0.5 {
			m.N[i] = V3{0, 0, 1}
		}
	}
	return applied
}

func ScoreDemoFieldCurvature(m *Mesh, field *FieldGrid, scene *DemoScene, mask []bool) CurvatureScore {
	res := CurvatureScore{Error: make([]float64, len(m.P)), Valid: make([]bool, len(m.P))}
	var signed []float64
	for i, p := range m.P {
		if i < len(mask) && mask[i] {
			continue
		}
		if p.X < scene.XMin+0.08 || p.X > scene.XMax-0.08 ||
			p.Y < scene.YMin+0.08 || p.Y > scene.YMax-0.08 ||
			scene.InHole(p.X, p.Y) {
			continue
		}
		info, ok := field.QuadricAt(p, field.opts.FitRadius*3.3)
		if !ok {
			continue
		}
		truth := demoCurvature(p.X, p.Y)
		res.Error[i] = info.MeanCurvature - truth
		res.Valid[i] = true
		signed = append(signed, res.Error[i])
	}
	res.Stats = statsFromValues(signed)
	res.Interior = len(signed)
	return res
}
