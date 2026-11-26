package main

import "math"

var cubeCornerOffset = [8][3]int{
	{0, 0, 0},
	{1, 0, 0},
	{0, 1, 0},
	{1, 1, 0},
	{0, 0, 1},
	{1, 0, 1},
	{0, 1, 1},
	{1, 1, 1},
}

var cubeTetrahedra = [6][4]int{
	{0, 1, 3, 7},
	{0, 1, 5, 7},
	{0, 2, 3, 7},
	{0, 2, 6, 7},
	{0, 4, 5, 7},
	{0, 4, 6, 7},
}

type crossingEdge struct {
	ga, gb int
	vid    int
}

func (g *FieldGrid) gridPosition(id int) V3 {
	plane := g.nx * g.ny
	k := id / plane
	rem := id % plane
	j := rem / g.nx
	i := rem % g.nx
	return g.corner(i, j, k)
}

func ExtractSurface(g *FieldGrid) *Mesh {
	m := &Mesh{}
	edgeVerts := map[uint64]int{}

	vertexFor := func(ga, gb int, pa, pb V3, va, vb float64) int {
		key := meshEdgeKey(ga, gb)
		if id, ok := edgeVerts[key]; ok {
			return id
		}
		t := 0.5
		if math.Abs(vb-va) > 1e-14 {
			t = (0 - va) / (vb - va)
			t = clampFloat(t, 0, 1)
		}
		id := len(m.P)
		m.P = append(m.P, LerpV3(pa, pb, t))
		edgeVerts[key] = id
		return id
	}

	emit := func(i0, i1, i2 int) {
		if i0 == i1 || i1 == i2 || i0 == i2 {
			return
		}
		a := m.P[i0]
		b := m.P[i1]
		c := m.P[i2]
		centroid := a.Add(b).Add(c).Mul(1.0 / 3.0)
		near, ok := g.tree.Nearest(centroid)
		if !ok {
			return
		}
		trim2 := g.opts.TrimRadius * g.opts.TrimRadius
		if near.Dist2 > trim2 {
			return
		}
		face := [3]int{i0, i1, i2}
		fn := Cross(b.Sub(a), c.Sub(a))
		if near.ID < len(g.normals) && Norm(fn) > 1e-18 {
			if Dot(fn, g.normals[near.ID]) < 0 {
				face[1], face[2] = face[2], face[1]
			}
		}
		m.Tri = append(m.Tri, face)
	}

	for k := 0; k < g.nz-1; k++ {
		for j := 0; j < g.ny-1; j++ {
			for i := 0; i < g.nx-1; i++ {
				var ids [8]int
				var pos [8]V3
				allValid := true
				for c := 0; c < 8; c++ {
					off := cubeCornerOffset[c]
					ii, jj, kk := i+off[0], j+off[1], k+off[2]
					id := g.index(ii, jj, kk)
					ids[c] = id
					pos[c] = g.corner(ii, jj, kk)
					if !g.valid[id] {
						allValid = false
					}
				}
				if !allValid {
					continue
				}

				for _, tet := range cubeTetrahedra {
					var crosses []crossingEdge
					for e := 0; e < 6; e++ {
						ea := [6][2]int{{0, 1}, {0, 2}, {0, 3}, {1, 2}, {1, 3}, {2, 3}}[e]
						la, lb := ea[0], ea[1]
						ga, gb := ids[tet[la]], ids[tet[lb]]
						va, vb := g.value[ga], g.value[gb]
						if (va < 0) == (vb < 0) {
							continue
						}
						vid := vertexFor(ga, gb, pos[tet[la]], pos[tet[lb]], va, vb)
						crosses = append(crosses, crossingEdge{ga: ga, gb: gb, vid: vid})
					}
					if len(crosses) < 3 {
						continue
					}

					order := orderCrossingEdges(crosses)
					if len(order) != len(crosses) {
						order = make([]int, len(crosses))
						for i := range order {
							order[i] = i
						}
					}
					vids := make([]int, len(order))
					for r, ci := range order {
						vids[r] = crosses[ci].vid
					}
					if len(vids) == 3 {
						emit(vids[0], vids[1], vids[2])
					} else if len(vids) >= 4 {
						emit(vids[0], vids[1], vids[2])
						emit(vids[0], vids[2], vids[3])
					}
				}
			}
		}
	}

	m.RemoveSmallComponents(64)
	m.ComputeNormals()
	return m
}

func shareCorner(a, b crossingEdge) bool {
	return a.ga == b.ga || a.ga == b.gb || a.gb == b.ga || a.gb == b.gb
}

func orderCrossingEdges(edges []crossingEdge) []int {
	n := len(edges)
	adj := make([][]int, n)
	for i := 0; i < n; i++ {
		for j := i + 1; j < n; j++ {
			if shareCorner(edges[i], edges[j]) {
				adj[i] = append(adj[i], j)
				adj[j] = append(adj[j], i)
			}
		}
	}
	order := []int{0}
	used := make([]bool, n)
	used[0] = true
	for len(order) < n {
		cur := order[len(order)-1]
		next := -1
		for _, cand := range adj[cur] {
			if !used[cand] {
				next = cand
				break
			}
		}
		if next < 0 {
			return nil
		}
		used[next] = true
		order = append(order, next)
	}
	return order
}
