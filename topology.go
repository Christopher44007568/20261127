package main

import "sort"

type Edge struct {
	A, B int
}

type EdgeIncidence struct {
	Face    int
	Corners [2]int
}

type Topology struct {
	Mesh      *Mesh
	Edge      []Edge
	EdgeIndex map[[2]int]int
	FaceEdge  [][3]int
	EdgeFaces [][]int
	EdgeInc   [][]EdgeIncidence
	VtxEdges  [][]int
	VtxFaces  [][]int
}

func edgeKey(a, b int) [2]int {
	if a > b {
		a, b = b, a
	}
	return [2]int{a, b}
}

func BuildTopology(m *Mesh) *Topology {
	tp := &Topology{
		Mesh:      m,
		EdgeIndex: map[[2]int]int{},
		FaceEdge:  make([][3]int, len(m.Tri)),
		VtxEdges:  make([][]int, len(m.Pos)),
		VtxFaces:  make([][]int, len(m.Pos)),
	}

	getEdge := func(a, b int) int {
		key := edgeKey(a, b)
		if id, ok := tp.EdgeIndex[key]; ok {
			return id
		}
		id := len(tp.Edge)
		tp.EdgeIndex[key] = id
		tp.Edge = append(tp.Edge, Edge{A: key[0], B: key[1]})
		tp.EdgeFaces = append(tp.EdgeFaces, nil)
		tp.EdgeInc = append(tp.EdgeInc, nil)
		tp.VtxEdges[a] = append(tp.VtxEdges[a], id)
		tp.VtxEdges[b] = append(tp.VtxEdges[b], id)
		return id
	}

	for f, t := range m.Tri {
		verts := [3]int{t.A, t.B, t.C}
		tp.VtxFaces[t.A] = append(tp.VtxFaces[t.A], f)
		tp.VtxFaces[t.B] = append(tp.VtxFaces[t.B], f)
		tp.VtxFaces[t.C] = append(tp.VtxFaces[t.C], f)
		for k := 0; k < 3; k++ {
			a := verts[k]
			b := verts[(k+1)%3]
			eid := getEdge(a, b)
			tp.FaceEdge[f][k] = eid
			tp.EdgeFaces[eid] = append(tp.EdgeFaces[eid], f)
			inc := EdgeIncidence{Face: f}
			if a == tp.Edge[eid].A {
				inc.Corners = [2]int{k, (k + 1) % 3}
			} else {
				inc.Corners = [2]int{(k + 1) % 3, k}
			}
			tp.EdgeInc[eid] = append(tp.EdgeInc[eid], inc)
		}
	}

	for i := range tp.VtxEdges {
		sort.Ints(tp.VtxEdges[i])
	}
	return tp
}

func (tp *Topology) EdgeID(a, b int) (int, bool) {
	id, ok := tp.EdgeIndex[edgeKey(a, b)]
	return id, ok
}

// Islands flood-fills faces while treating seam edges as chart borders.
func (tp *Topology) Islands(seam []bool) [][]int {
	comp := make([]int, len(tp.Mesh.Tri))
	for i := range comp {
		comp[i] = -1
	}
	var islands [][]int
	for seed := range tp.Mesh.Tri {
		if comp[seed] != -1 {
			continue
		}
		id := len(islands)
		var faces []int
		queue := []int{seed}
		comp[seed] = id
		for len(queue) > 0 {
			f := queue[0]
			queue = queue[1:]
			faces = append(faces, f)
			for _, eid := range tp.FaceEdge[f] {
				if seam != nil && eid < len(seam) && seam[eid] {
					continue
				}
				for _, nf := range tp.EdgeFaces[eid] {
					if nf == f || comp[nf] != -1 {
						continue
					}
					comp[nf] = id
					queue = append(queue, nf)
				}
			}
		}
		islands = append(islands, faces)
	}
	return islands
}

type DSU struct {
	parent []int
	size   []int
}

func NewDSU(n int) *DSU {
	d := &DSU{parent: make([]int, n), size: make([]int, n)}
	for i := range d.parent {
		d.parent[i] = i
		d.size[i] = 1
	}
	return d
}

func (d *DSU) Find(x int) int {
	for d.parent[x] != x {
		d.parent[x] = d.parent[d.parent[x]]
		x = d.parent[x]
	}
	return x
}

func (d *DSU) Union(a, b int) {
	a = d.Find(a)
	b = d.Find(b)
	if a == b {
		return
	}
	if d.size[a] < d.size[b] {
		a, b = b, a
	}
	d.parent[b] = a
	d.size[a] += d.size[b]
}
