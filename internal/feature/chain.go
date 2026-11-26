package feature

import (
	"fmt"

	"featurelines/internal/mesh"
	"featurelines/internal/topo"
)

// Gap records a bridged break between two chain endpoints.
type Gap struct {
	// From and To are the connected endpoint vertex indices.
	From, To int
	// Via lists intermediate mesh vertices (failed/threshold-rejected).
	Via []int
	// Points is the rendered bridge polyline (From, Via..., To positions).
	Points []mesh.Vec3
	Length float64
	Kind   string // "single-vertex", "multi-gap", "closure"
}

// Chain is an ordered feature polyline over mesh vertices.
type Chain struct {
	Vertices []int
	Edges    []int // mesh edge ids along the chain; -1 for bridged links
	Closed   bool
	Contour  bool
	Gaps     []Gap
	Score    float64
}

// PointCount returns the number of polyline points including bridge samples.
func (c *Chain) PointCount() int { return len(c.Vertices) }

// Endpoints returns the two endpoint vertex ids of an open chain.
func (c *Chain) Endpoints() (int, int, bool) {
	if c.Closed || len(c.Vertices) < 2 {
		return -1, -1, false
	}
	return c.Vertices[0], c.Vertices[len(c.Vertices)-1], true
}

// Length sums polyline length in world units.
func (c *Chain) Length(m *mesh.Mesh) float64 {
	total := 0.0
	for i := 1; i < len(c.Vertices); i++ {
		total += m.V[c.Vertices[i-1]].Dist(m.V[c.Vertices[i]])
	}
	return total
}

func (c *Chain) String() string {
	state := "open"
	if c.Closed {
		state = "closed"
	}
	return fmt.Sprintf("Chain(%s verts=%d gaps=%d)", state, len(c.Vertices), len(c.Gaps))
}

// BuildChains orders the given edge set into maximal polylines. Vertices with
// feature degree != 2 act as chain nodes; pure degree-2 cycles become closed
// chains. Junctions (degree >=3) produce incident arm chains that the tracer
// can later pair by tangent continuity.
func BuildChains(tp *topo.Topology, edgeSet map[int]bool, contour bool) []*Chain {
	adj := make([][]int, len(tp.Mesh.V)) // values are edge indices
	for ei := range edgeSet {
		e := tp.Edges[ei]
		adj[e.A] = append(adj[e.A], ei)
		adj[e.B] = append(adj[e.B], ei)
	}

	usedEdge := make(map[int]bool, len(edgeSet))
	var chains []*Chain

	makeChain := func(startV, firstE int) *Chain {
		ch := &Chain{Contour: contour}
		v := startV
		e := firstE
		ch.Vertices = append(ch.Vertices, v)
		for {
			usedEdge[e] = true
			ch.Edges = append(ch.Edges, e)
			v = tp.Other(e, v)
			ch.Vertices = append(ch.Vertices, v)

			next := -1
			for _, cand := range adj[v] {
				if cand == e || usedEdge[cand] {
					continue
				}
				if len(adj[v]) == 2 {
					next = cand
					break
				}
				// At a junction/endpoint stop this arm.
				if len(adj[v]) > 2 {
					next = -1
					break
				}
			}
			if next == -1 {
				break
			}
			e = next
		}
		// Closed cycle: returned to start and every edge consumed.
		if ch.Vertices[len(ch.Vertices)-1] == startV && len(adj[startV]) == 2 {
			ch.Closed = true
			ch.Vertices = ch.Vertices[:len(ch.Vertices)-1]
		}
		return ch
	}

	// First pass: chains rooted at degree != 2 vertices.
	for v := 0; v < len(tp.Mesh.V); v++ {
		if len(adj[v]) == 0 || len(adj[v]) == 2 {
			continue
		}
		for _, ei := range adj[v] {
			if usedEdge[ei] {
				continue
			}
			chains = append(chains, makeChain(v, ei))
		}
	}

	// Second pass: degree-2-only cycles.
	for ei := range edgeSet {
		if usedEdge[ei] {
			continue
		}
		e := tp.Edges[ei]
		ch := makeChain(e.A, ei)
		if !ch.Closed {
			// Should not happen for an unused degree-2 component, but keep it.
			ch.Closed = false
		}
		chains = append(chains, ch)
	}

	return chains
}
