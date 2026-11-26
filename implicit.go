package main

import (
	"math"
)

type ImplicitOptions struct {
	Cell       float64
	FitRadius  float64
	FitK       int
	MinFit     int
	Padding    float64
	TrimRadius float64
}

type FieldGrid struct {
	Origin Box
	origin V3
	cell   float64
	nx     int
	ny     int
	nz     int
	value  []float64
	valid  []bool

	points  []V3
	normals []V3
	tree    *KDTree
	opts    ImplicitOptions
}

func DefaultImplicitOptions(spacing float64) ImplicitOptions {
	cell := spacing * 1.9
	if cell < 0.016 {
		cell = 0.016
	}
	if cell > 0.032 {
		cell = 0.032
	}
	return ImplicitOptions{
		Cell:       cell,
		FitRadius:  cell * 2.9,
		FitK:       48,
		MinFit:     12,
		Padding:    cell * 2.2,
		TrimRadius: cell * 1.25,
	}
}

func (g *FieldGrid) index(i, j, k int) int {
	return i + g.nx*(j+g.ny*k)
}

func (g *FieldGrid) corner(i, j, k int) V3 {
	return V3{
		g.origin.X + float64(i)*g.cell,
		g.origin.Y + float64(j)*g.cell,
		g.origin.Z + float64(k)*g.cell,
	}
}

func BuildImplicitField(points []V3, normals []V3, spacing float64, opts ImplicitOptions) *FieldGrid {
	if opts.Cell <= 0 {
		opts = DefaultImplicitOptions(spacing)
	}
	if opts.FitRadius <= 0 {
		opts.FitRadius = opts.Cell * 2.7
	}
	if opts.MinFit <= 0 {
		opts.MinFit = 12
	}
	if opts.TrimRadius <= 0 {
		opts.TrimRadius = opts.Cell * 1.35
	}

	box := NewBox()
	for _, p := range points {
		box.Add(p)
	}
	origin := box.Min.Sub(V3{opts.Padding, opts.Padding, opts.Padding})
	size := box.Max.Sub(box.Min).Add(V3{opts.Padding * 2, opts.Padding * 2, opts.Padding * 2})
	nx := int(math.Ceil(size.X/opts.Cell)) + 1
	ny := int(math.Ceil(size.Y/opts.Cell)) + 1
	nz := int(math.Ceil(size.Z/opts.Cell)) + 1
	if nx < 3 {
		nx = 3
	}
	if ny < 3 {
		ny = 3
	}
	if nz < 3 {
		nz = 3
	}

	g := &FieldGrid{
		Origin:  box,
		origin:  origin,
		cell:    opts.Cell,
		nx:      nx,
		ny:      ny,
		nz:      nz,
		value:   make([]float64, nx*ny*nz),
		valid:   make([]bool, nx*ny*nz),
		points:  points,
		normals: normals,
		tree:    NewKDTree(points),
		opts:    opts,
	}

	for k := 0; k < nz; k++ {
		for j := 0; j < ny; j++ {
			for i := 0; i < nx; i++ {
				id := g.index(i, j, k)
				q := g.corner(i, j, k)
				v, ok := g.fitValue(q)
				if ok {
					g.value[id] = v
					g.valid[id] = true
				} else {
					g.value[id] = 0
					g.valid[id] = false
				}
			}
		}
	}
	return g
}

func wendlandWeight(d, radius float64) float64 {
	if d >= radius {
		return 0
	}
	t := d / radius
	u := 1 - t
	return u * u * u * u * (4*t + 1)
}

func (g *FieldGrid) fitValue(q V3) (float64, bool) {
	nearest, ok := g.tree.Nearest(q)
	if !ok || nearest.Dist2 > g.opts.FitRadius*g.opts.FitRadius {
		return 0, false
	}
	ids := g.tree.RadiusSearch(q, g.opts.FitRadius)
	if len(ids) < g.opts.MinFit {
		return 0, false
	}

	totalW := 0.0
	centroid := V3{}
	frameN := V3{}
	type sample struct {
		x, y, z, w float64
	}
	samples := make([]sample, 0, len(ids))
	for _, id := range ids {
		p := g.points[id]
		d := Norm(q.Sub(p))
		w := wendlandWeight(d, g.opts.FitRadius)
		if w <= 1e-8 {
			continue
		}
		totalW += w
		centroid = centroid.Add(p.Mul(w))
		if id < len(g.normals) {
			frameN = frameN.Add(g.normals[id].Mul(w))
		}
		samples = append(samples, sample{w: w})
		// Store point temporarily through closure-unfriendly pattern below.
		samples[len(samples)-1].x = p.X
		samples[len(samples)-1].y = p.Y
		samples[len(samples)-1].z = p.Z
	}
	if totalW <= 1e-12 || len(samples) < g.opts.MinFit {
		return 0, false
	}
	centroid = centroid.Div(totalW)
	n := Normalize(frameN)
	if Norm(n) < 0.5 {
		n, _ = g.localPCANormal(ids)
	}
	if Norm(n) < 0.5 {
		return 0, false
	}

	ref := V3{0, 0, 1}
	if math.Abs(n.Z) > 0.9 {
		ref = V3{1, 0, 0}
	}
	u := Normalize(Cross(ref, n))
	if Norm(u) < 0.5 {
		return 0, false
	}
	v := Cross(n, u)

	var ata [6][6]float64
	var atb [6]float64
	for _, s := range samples {
		p := V3{s.x, s.y, s.z}
		r := p.Sub(centroid)
		x := Dot(r, u)
		y := Dot(r, v)
		z := Dot(r, n)
		basis := [6]float64{1, x, y, x * x, x * y, y * y}
		w := s.w
		for row := 0; row < 6; row++ {
			atb[row] += w * basis[row] * z
			for col := 0; col < 6; col++ {
				ata[row][col] += w * basis[row] * basis[col]
			}
		}
	}
	var rhs [6]float64
	for i := range atb {
		rhs[i] = atb[i]
	}
	if !solveLinear6(&ata, &rhs) {
		return 0, false
	}

	rq := q.Sub(centroid)
	xq := Dot(rq, u)
	yq := Dot(rq, v)
	zq := Dot(rq, n)
	fitZ := rhs[0] + rhs[1]*xq + rhs[2]*yq + rhs[3]*xq*xq + rhs[4]*xq*yq + rhs[5]*yq*yq
	return zq - fitZ, true
}

func (g *FieldGrid) localPCANormal(ids []int) (V3, bool) {
	centroid := V3{}
	for _, id := range ids {
		centroid = centroid.Add(g.points[id])
	}
	centroid = centroid.Div(float64(len(ids)))
	var cov [3][3]float64
	for _, id := range ids {
		r := g.points[id].Sub(centroid)
		cov[0][0] += r.X * r.X
		cov[0][1] += r.X * r.Y
		cov[0][2] += r.X * r.Z
		cov[1][1] += r.Y * r.Y
		cov[1][2] += r.Y * r.Z
		cov[2][2] += r.Z * r.Z
	}
	cov[1][0] = cov[0][1]
	cov[2][0] = cov[0][2]
	cov[2][1] = cov[1][2]
	_, vecs := eigenSym3(cov)
	n := Normalize(V3{vecs[0][0], vecs[1][0], vecs[2][0]})
	return n, Norm(n) > 0.5
}
