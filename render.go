package main

import (
	"image"
	"image/color"
	"math"
)

type Camera struct {
	Eye, Target, Up Vec3
	FovY            float64
}

func (c Camera) basis() (Vec3, Vec3, Vec3) {
	fwd := c.Target.Sub(c.Eye).Normalize()
	right := fwd.Cross(c.Up).Normalize()
	up := right.Cross(fwd).Normalize()
	return right, up, fwd
}

type projVert struct {
	x, y, z float64
	ok      bool
}

func (c Camera) project(p Vec3, w, h int) projVert {
	right, up, fwd := c.basis()
	v := p.Sub(c.Eye)
	cx, cy, cz := v.Dot(right), v.Dot(up), v.Dot(fwd)
	if cz <= 0.04 {
		return projVert{}
	}
	aspect := float64(w) / float64(h)
	t := math.Tan(c.FovY * 0.5)
	ndcX := cx / (cz * t * aspect)
	ndcY := cy / (cz * t)
	return projVert{
		x: (ndcX*0.5 + 0.5) * float64(w),
		y: (0.5 - ndcY*0.5) * float64(h),
		z: cz, ok: true,
	}
}

type Mat3 [numMaterials]float64

type GBuffer struct {
	W, H  int
	Depth []float64
	Pos   []Vec3
	Nrm   []Vec3
	Cov   []Mat3
	GeoH  []float64
	AO    []float64
	Valid []bool
}

func newGBuffer(w, h int) *GBuffer {
	g := &GBuffer{
		W: w, H: h,
		Depth: make([]float64, w*h),
		Pos:   make([]Vec3, w*h),
		Nrm:   make([]Vec3, w*h),
		Cov:   make([]Mat3, w*h),
		GeoH:  make([]float64, w*h),
		AO:    make([]float64, w*h),
		Valid: make([]bool, w*h),
	}
	for i := range g.Depth {
		g.Depth[i] = math.Inf(1)
	}
	return g
}

type mesh struct {
	nx, nz int
	pos    []Vec3
	nrm    []Vec3
	cov    []Mat3
	h      []float64
	ao     []float64
}

func buildMesh(nx, nz int) *mesh {
	m := &mesh{
		nx: nx, nz: nz,
		pos: make([]Vec3, nx*nz),
		nrm: make([]Vec3, nx*nz),
		cov: make([]Mat3, nx*nz),
		h:   make([]float64, nx*nz),
		ao:  make([]float64, nx*nz),
	}
	for iz := 0; iz < nz; iz++ {
		z := mix(worldMinZ, worldMaxZ, float64(iz)/float64(nz-1))
		for ix := 0; ix < nx; ix++ {
			x := mix(worldMinX, worldMaxX, float64(ix)/float64(nx-1))
			i := iz*nx + ix
			h := terrainHeight(x, z)
			m.h[i] = h
			m.pos[i] = Vec3{x, h, z}
			m.nrm[i] = terrainNormal(x, z, 0.004)
			m.cov[i] = Mat3(coverageAt(x, z))
		}
	}
	m.computeAO()
	return m
}

func (m *mesh) computeAO() {
	type tap struct {
		dx, dz int
		weight float64
	}
	taps := []tap{
		{2, 0, 1.0}, {-2, 0, 1.0}, {0, 2, 1.0}, {0, -2, 1.0},
		{2, 2, 0.8}, {-2, 2, 0.8}, {2, -2, 0.8}, {-2, -2, 0.8},
		{5, 0, 0.7}, {-5, 0, 0.7}, {0, 5, 0.7}, {0, -5, 0.7},
		{5, 5, 0.5}, {-5, 5, 0.5}, {5, -5, 0.5}, {-5, -5, 0.5},
	}
	for iz := 0; iz < m.nz; iz++ {
		for ix := 0; ix < m.nx; ix++ {
			i := iz*m.nx + ix
			occ, norm := 0.0, 0.0
			for _, t := range taps {
				jx, jz := ix+t.dx, iz+t.dz
				if jx < 0 || jx >= m.nx || jz < 0 || jz >= m.nz {
					continue
				}
				diff := m.h[jz*m.nx+jx] - m.h[i]
				occ += smoothstep(0.004, 0.16, diff) * t.weight
				norm += t.weight
			}
			m.ao[i] = clamp(1.0-0.72*occ/math.Max(norm, 1), 0.28, 1)
		}
	}
}

func edge(ax, ay, bx, by, px, py float64) float64 {
	return (px-ax)*(by-ay) - (py-ay)*(bx-ax)
}

func rasterTriangle(g *GBuffer, m *mesh, ids [3]int, pv [3]projVert) {
	a, b, c := pv[0], pv[1], pv[2]
	if !a.ok || !b.ok || !c.ok {
		return
	}
	area := edge(a.x, a.y, b.x, b.y, c.x, c.y)
	if math.Abs(area) < 1e-8 {
		return
	}
	minX := int(math.Floor(math.Min(a.x, math.Min(b.x, c.x))))
	maxX := int(math.Ceil(math.Max(a.x, math.Max(b.x, c.x))))
	minY := int(math.Floor(math.Min(a.y, math.Min(b.y, c.y))))
	maxY := int(math.Ceil(math.Max(a.y, math.Max(b.y, c.y))))
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= g.W {
		maxX = g.W - 1
	}
	if maxY >= g.H {
		maxY = g.H - 1
	}
	invArea := 1.0 / area
	p0, p1, p2 := m.pos[ids[0]], m.pos[ids[1]], m.pos[ids[2]]
	n0, n1, n2 := m.nrm[ids[0]], m.nrm[ids[1]], m.nrm[ids[2]]
	c0, c1, c2 := m.cov[ids[0]], m.cov[ids[1]], m.cov[ids[2]]
	h0, h1, h2 := m.h[ids[0]], m.h[ids[1]], m.h[ids[2]]
	a0, a1, a2 := m.ao[ids[0]], m.ao[ids[1]], m.ao[ids[2]]
	for y := minY; y <= maxY; y++ {
		py := float64(y) + 0.5
		for x := minX; x <= maxX; x++ {
			px := float64(x) + 0.5
			w0 := edge(b.x, b.y, c.x, c.y, px, py) * invArea
			w1 := edge(c.x, c.y, a.x, a.y, px, py) * invArea
			w2 := 1 - w0 - w1
			if w0 < -1e-7 || w1 < -1e-7 || w2 < -1e-7 {
				continue
			}
			iz := 1.0/a.z*w0 + 1.0/b.z*w1 + 1.0/c.z*w2
			if iz <= 0 {
				continue
			}
			depth := 1.0 / iz
			i := y*g.W + x
			if depth >= g.Depth[i] {
				continue
			}
			g.Depth[i] = depth
			g.Valid[i] = true
			g.Pos[i] = Vec3{
				X: (p0.X*w0/a.z + p1.X*w1/b.z + p2.X*w2/c.z) * depth,
				Y: (p0.Y*w0/a.z + p1.Y*w1/b.z + p2.Y*w2/c.z) * depth,
				Z: (p0.Z*w0/a.z + p1.Z*w1/b.z + p2.Z*w2/c.z) * depth,
			}
			g.Nrm[i] = Vec3{
				X: (n0.X*w0/a.z + n1.X*w1/b.z + n2.X*w2/c.z) * depth,
				Y: (n0.Y*w0/a.z + n1.Y*w1/b.z + n2.Y*w2/c.z) * depth,
				Z: (n0.Z*w0/a.z + n1.Z*w1/b.z + n2.Z*w2/c.z) * depth,
			}.Normalize()
			for k := 0; k < numMaterials; k++ {
				g.Cov[i][k] = (c0[k]*w0/a.z + c1[k]*w1/b.z + c2[k]*w2/c.z) * depth
			}
			g.GeoH[i] = (h0*w0/a.z + h1*w1/b.z + h2*w2/c.z) * depth
			g.AO[i] = (a0*w0/a.z + a1*w1/b.z + a2*w2/c.z) * depth
		}
	}
}

func renderGBuffer(c Camera, w, h, nx, nz int) *GBuffer {
	g := newGBuffer(w, h)
	m := buildMesh(nx, nz)
	projected := make([]projVert, len(m.pos))
	for i, p := range m.pos {
		projected[i] = c.project(p, w, h)
	}
	for iz := 0; iz < nz-1; iz++ {
		for ix := 0; ix < nx-1; ix++ {
			i00 := iz*nx + ix
			i10 := i00 + 1
			i01 := i00 + nx
			i11 := i01 + 1
			rasterTriangle(g, m, [3]int{i00, i01, i10}, [3]projVert{projected[i00], projected[i01], projected[i10]})
			rasterTriangle(g, m, [3]int{i10, i01, i11}, [3]projVert{projected[i10], projected[i01], projected[i11]})
		}
	}
	return g
}

func gbufferToImage(g *GBuffer, shade func(x, y, i int) Vec3) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, g.W, g.H))
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			i := y*g.W + x
			var c Vec3
			if g.Valid[i] {
				c = shade(x, y, i)
			} else {
				t := float64(y) / float64(g.H)
				c = Vec3{0.015, 0.022, 0.030}.Lerp(Vec3{0.048, 0.060, 0.066}, 1-t)
			}
			r, g, b, a := rgbaOf(c)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: a})
		}
	}
	return img
}
