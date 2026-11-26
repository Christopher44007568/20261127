// Package topo builds edge/vertex adjacency tables for a triangle mesh.
package topo

import (
	"sort"

	"featurelines/internal/mesh"
)

// Edge is an undirected mesh edge with its incident triangles.
type Edge struct {
	A, B  int
	Faces []int
}

// Boundary reports whether the edge has exactly one incident face.
func (e *Edge) Boundary() bool { return len(e.Faces) == 1 }

type edgeKey struct{ lo, hi int }

// Topology holds deduplicated edges and per-vertex incident edges/faces.
type Topology struct {
	Mesh      *mesh.Mesh
	Edges     []*Edge
	EdgeIndex map[[2]int]int
	// VEdges lists edge indices incident to each vertex.
	VEdges [][]int
	// VFaces lists triangle indices incident to each vertex.
	VFaces [][]int
}

// Key returns the canonical sorted array key for a vertex pair.
func Key(a, b int) [2]int {
	if a < b {
		return [2]int{a, b}
	}
	return [2]int{b, a}
}

// Build constructs the topology for m.
func Build(m *mesh.Mesh) *Topology {
	tp := &Topology{
		Mesh:      m,
		EdgeIndex: map[[2]int]int{},
		VEdges:    make([][]int, len(m.V)),
		VFaces:    make([][]int, len(m.V)),
	}

	getEdge := func(a, b int) int {
		k := Key(a, b)
		if idx, ok := tp.EdgeIndex[k]; ok {
			return idx
		}
		e := &Edge{A: k[0], B: k[1]}
		idx := len(tp.Edges)
		tp.Edges = append(tp.Edges, e)
		tp.EdgeIndex[k] = idx
		tp.VEdges[k[0]] = append(tp.VEdges[k[0]], idx)
		tp.VEdges[k[1]] = append(tp.VEdges[k[1]], idx)
		return idx
	}

	for fi, t := range m.T {
		e0 := getEdge(t[0], t[1])
		e1 := getEdge(t[1], t[2])
		e2 := getEdge(t[2], t[0])
		tp.Edges[e0].Faces = append(tp.Edges[e0].Faces, fi)
		tp.Edges[e1].Faces = append(tp.Edges[e1].Faces, fi)
		tp.Edges[e2].Faces = append(tp.Edges[e2].Faces, fi)
		for _, vi := range t {
			tp.VFaces[vi] = append(tp.VFaces[vi], fi)
		}
	}

	for vi := range tp.VEdges {
		sort.Ints(tp.VEdges[vi])
	}
	return tp
}

// Other returns the opposite endpoint of edge ei from vertex v.
func (tp *Topology) Other(ei, v int) int {
	e := tp.Edges[ei]
	if e.A == v {
		return e.B
	}
	return e.A
}
