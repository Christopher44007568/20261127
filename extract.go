package main

import "math"

type ThresholdCell struct {
	Index  int
	Pos    Vec3
	Value  float32
	Weight float32
}

type MeshVertex struct {
	Pos    Vec3
	Normal Vec3
	Error  float64
}

type Triangle [3]int

type Mesh struct {
	Vertices  []MeshVertex
	Triangles []Triangle
}

func ExtractThresholdCells(g *VoxelGrid, valueThreshold, minWeight float32) []ThresholdCell {
	cells := make([]ThresholdCell, 0, 4096)
	for id := 0; id < len(g.Value); id++ {
		if g.Weight[id] < minWeight {
			continue
		}
		if math.Abs(float64(g.Value[id])) <= float64(valueThreshold) {
			ix := id % g.NX
			iy := (id / g.NX) % g.NY
			iz := id / (g.NX * g.NY)
			cells = append(cells, ThresholdCell{
				Index:  id,
				Pos:    g.Center(ix, iy, iz),
				Value:  g.Value[id],
				Weight: g.Weight[id],
			})
		}
	}
	return cells
}

func gradientAtVoxel(g *VoxelGrid, id int) Vec3 {
	ix := id % g.NX
	iy := (id / g.NX) % g.NY
	iz := id / (g.NX * g.NY)

	h := g.Voxel
	dx := float32(0)
	if ix >= 2 && ix+2 < g.NX {
		dx = (g.Value[id+2] - g.Value[id-2]) / float32(4*h)
	} else if ix > 0 && ix+1 < g.NX {
		dx = (g.Value[id+1] - g.Value[id-1]) / float32(2*h)
	} else if ix+1 < g.NX {
		dx = (g.Value[id+1] - g.Value[id]) / float32(h)
	} else if ix > 0 {
		dx = (g.Value[id] - g.Value[id-1]) / float32(h)
	}

	dy := float32(0)
	switch {
	case iy >= 2 && iy+2 < g.NY:
		dy = (g.Value[id+2*g.NX] - g.Value[id-2*g.NX]) / float32(4*h)
	case iy > 0 && iy+1 < g.NY:
		dy = (g.Value[id+g.NX] - g.Value[id-g.NX]) / float32(2*h)
	case iy+1 < g.NY:
		dy = (g.Value[id+g.NX] - g.Value[id]) / float32(h)
	case iy > 0:
		dy = (g.Value[id] - g.Value[id-g.NX]) / float32(h)
	}

	strideZ := g.NX * g.NY
	dz := float32(0)
	if iz >= 2 && iz+2 < g.NZ {
		dz = (g.Value[id+2*strideZ] - g.Value[id-2*strideZ]) / float32(4*h)
	} else if iz > 0 && iz+1 < g.NZ {
		dz = (g.Value[id+strideZ] - g.Value[id-strideZ]) / float32(2*h)
	} else if iz+1 < g.NZ {
		dz = (g.Value[id+strideZ] - g.Value[id]) / float32(h)
	} else if iz > 0 {
		dz = (g.Value[id] - g.Value[id-strideZ]) / float32(h)
	}
	n := V3(float64(dx), float64(dy), float64(dz)).Norm()
	if n.Len() < 1e-8 {
		return V3(0, 0, 1)
	}
	return n
}

var cubeCornerOffsets = [8][3]int{
	{0, 0, 0}, {1, 0, 0}, {1, 1, 0}, {0, 1, 0},
	{0, 0, 1}, {1, 0, 1}, {1, 1, 1}, {0, 1, 1},
}

var cubeTetrahedra = [6][4]int{
	{0, 1, 2, 6},
	{0, 2, 3, 6},
	{0, 3, 7, 6},
	{0, 7, 4, 6},
	{0, 4, 5, 6},
	{0, 5, 1, 6},
}

func ExtractMarchingTetrahedra(g *VoxelGrid, minWeight float32) *Mesh {
	mesh := &Mesh{Vertices: make([]MeshVertex, 0, 16384), Triangles: make([]Triangle, 0, 16384)}
	edgeVertices := make(map[[2]int]int, 32768)

	cornerID := func(x, y, z, c int) int {
		o := cubeCornerOffsets[c]
		return g.Index(x+o[0], y+o[1], z+o[2])
	}

	edgeVertex := func(aID, bID int) int {
		key := [2]int{aID, bID}
		if aID > bID {
			key = [2]int{bID, aID}
		}
		if vi, ok := edgeVertices[key]; ok {
			return vi
		}
		va := g.Value[aID]
		vb := g.Value[bID]
		t := float64(-va / (vb - va))
		if math.IsNaN(t) || math.IsInf(t, 0) {
			t = 0.5
		}
		t = clampFloat(t, 0, 1)

		ax := aID % g.NX
		ay := (aID / g.NX) % g.NY
		az := aID / (g.NX * g.NY)
		bx := bID % g.NX
		by := (bID / g.NX) % g.NY
		bz := bID / (g.NX * g.NY)
		pa := g.Center(ax, ay, az)
		pb := g.Center(bx, by, bz)
		na := gradientAtVoxel(g, aID)
		nb := gradientAtVoxel(g, bID)
		n := na.Lerp(nb, t).Norm()
		if n.Len() < 1e-8 {
			n = V3(0, 0, 1)
		}
		vi := len(mesh.Vertices)
		mesh.Vertices = append(mesh.Vertices, MeshVertex{Pos: pa.Lerp(pb, t), Normal: n})
		edgeVertices[key] = vi
		return vi
	}

	addTriangle := func(a, b, c int) {
		pa := mesh.Vertices[a].Pos
		pb := mesh.Vertices[b].Pos
		pc := mesh.Vertices[c].Pos
		gn := mesh.Vertices[a].Normal.Add(mesh.Vertices[b].Normal).Add(mesh.Vertices[c].Normal).Norm()
		geomN := pb.Sub(pa).Cross(pc.Sub(pa)).Norm()
		if gn.Len() > 0 && geomN.Dot(gn) < 0 {
			b, c = c, b
		}
		mesh.Triangles = append(mesh.Triangles, Triangle{a, b, c})
	}

	for iz := 0; iz < g.NZ-1; iz++ {
		for iy := 0; iy < g.NY-1; iy++ {
			for ix := 0; ix < g.NX-1; ix++ {
				var ids [8]int
				valid := true
				for c := 0; c < 8; c++ {
					ids[c] = cornerID(ix, iy, iz, c)
					if g.Weight[ids[c]] < minWeight {
						valid = false
						break
					}
				}
				if !valid {
					continue
				}

				for _, tet := range cubeTetrahedra {
					var pos, neg []int
					var localID [4]int
					for i, c := range tet {
						localID[i] = ids[c]
						if g.Value[ids[c]] < 0 {
							neg = append(neg, i)
						} else {
							pos = append(pos, i)
						}
					}
					if len(pos) == 0 || len(neg) == 0 {
						continue
					}
					ev := func(i, j int) int {
						return edgeVertex(localID[i], localID[j])
					}

					if len(pos) == 1 {
						p := pos[0]
						a := ev(p, neg[0])
						b := ev(p, neg[1])
						c := ev(p, neg[2])
						addTriangle(a, b, c)
					} else if len(neg) == 1 {
						n := neg[0]
						a := ev(n, pos[0])
						b := ev(n, pos[1])
						c := ev(n, pos[2])
						addTriangle(a, b, c)
					} else {
						p0, p1 := pos[0], pos[1]
						n0, n1 := neg[0], neg[1]
						ac := ev(p0, n0)
						bc := ev(p1, n0)
						bd := ev(p1, n1)
						ad := ev(p0, n1)
						addTriangle(ac, bc, bd)
						addTriangle(ac, bd, ad)
					}
				}
			}
		}
	}
	return mesh
}
