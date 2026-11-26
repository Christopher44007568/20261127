package qem

import "math"

// NewDetailOrb builds a deterministic subdivided icosphere with broad volume,
// high-frequency normal detail, and latitude material seams.
func NewDetailOrb(detail int) *Mesh {
	gr := (1 + math.Sqrt(5)) / 2
	base := []Vec3{
		V(-1, gr, 0), V(1, gr, 0), V(-1, -gr, 0), V(1, -gr, 0),
		V(0, -1, gr), V(0, 1, gr), V(0, -1, -gr), V(0, 1, -gr),
		V(gr, 0, -1), V(gr, 0, 1), V(-gr, 0, -1), V(-gr, 0, 1),
	}
	positions := make([]Vec3, len(base))
	for i, p := range base {
		positions[i] = p.Normalized()
	}
	faces := [][3]int{
		{0, 11, 5}, {0, 5, 1}, {0, 1, 7}, {0, 7, 10}, {0, 10, 11},
		{1, 5, 9}, {5, 11, 4}, {11, 10, 2}, {10, 7, 6}, {7, 1, 8},
		{3, 9, 4}, {3, 4, 2}, {3, 2, 6}, {3, 6, 8}, {3, 8, 9},
		{4, 9, 5}, {2, 4, 11}, {6, 2, 10}, {8, 6, 7}, {9, 8, 1},
	}
	midCache := make(map[Edge]int)
	midpoint := func(a, b int) int {
		e := NewEdge(a, b)
		if id, ok := midCache[e]; ok {
			return id
		}
		p := positions[a].Add(positions[b]).Mul(0.5).Normalized()
		id := len(positions)
		positions = append(positions, p)
		midCache[e] = id
		return id
	}
	for level := 0; level < detail; level++ {
		next := make([][3]int, 0, len(faces)*4)
		for _, f := range faces {
			a := midpoint(f[0], f[1])
			b := midpoint(f[1], f[2])
			c := midpoint(f[2], f[0])
			next = append(next,
				[3]int{f[0], a, c},
				[3]int{f[1], b, a},
				[3]int{f[2], c, b},
				[3]int{a, b, c},
			)
		}
		faces = next
	}

	for i, p := range positions {
		n := p.Normalized()
		theta := math.Atan2(n.Z, n.X)
		phi := math.Asin(math.Max(-1, math.Min(1, n.Y)))
		r := 1.0
		r += 0.095 * math.Sin(5*theta+1.2) * math.Cos(4*phi-0.3)
		r += 0.038 * math.Sin(17*theta+3*math.Sin(3*phi)) * math.Cos(13*phi+0.6)
		r += 0.018 * math.Sin(31*theta-7*phi)
		q := n.Mul(r)
		q.Y *= 1.08
		positions[i] = q
	}

	m := NewMesh()
	m.Name = "detail_orb"
	matCoral := m.AddMaterial("coral_bands")
	matSand := m.AddMaterial("sand_bands")
	matTeal := m.AddMaterial("teal_bands")
	for _, p := range positions {
		m.AddVertex(p)
	}
	for _, f := range faces {
		center := positions[f[0]].Add(positions[f[1]]).Add(positions[f[2]]).Mul(1.0 / 3.0)
		n := center.Normalized()
		phi := math.Asin(math.Max(-1, math.Min(1, n.Y)))
		band := int(math.Floor((phi + math.Pi/2) / (math.Pi / 6)))
		if band > 5 {
			band = 5
		}
		mat := []int{matCoral, matSand, matTeal}[band%3]
		_, _ = m.AddTriangle(f[0], f[1], f[2], mat)
	}
	if m.SignedVolume6() < 0 {
		for i := range m.Triangles {
			t := &m.Triangles[i]
			t.B, t.C = t.C, t.B
		}
	}
	return m
}

// NewWavyPatch builds an open grid. Its outer loop is a boundary and its
// central column is a material seam.
func NewWavyPatch(nx, nz int) *Mesh {
	if nx < 5 {
		nx = 5
	}
	if nz < 5 {
		nz = 5
	}
	m := NewMesh()
	m.Name = "wavy_boundary_patch"
	water := m.AddMaterial("water_side")
	reed := m.AddMaterial("reed_side")
	for j := 0; j <= nz; j++ {
		z := -1.35 + 2.7*float64(j)/float64(nz)
		for i := 0; i <= nx; i++ {
			x := -1.65 + 3.3*float64(i)/float64(nx)
			y := 0.20*math.Sin(2.1*x+0.4)*math.Cos(2.4*z-0.2) +
				0.055*math.Sin(7.5*x+2.0*z) +
				0.025*math.Cos(11*z-3*x)
			m.AddVertex(V(x, y, z))
		}
	}
	id := func(i, j int) int { return j*(nx+1) + i }
	for j := 0; j < nz; j++ {
		for i := 0; i < nx; i++ {
			x := -1.65 + 3.3*(float64(i)+0.5)/float64(nx)
			mat := water
			if x > 0 {
				mat = reed
			}
			v00 := id(i, j)
			v10 := id(i+1, j)
			v01 := id(i, j+1)
			v11 := id(i+1, j+1)
			_, _ = m.AddTriangle(v00, v01, v10, mat)
			_, _ = m.AddTriangle(v10, v01, v11, mat)
		}
	}
	return m
}
