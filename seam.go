package main

import (
	"container/heap"
	"math"
)

type SeamPlan struct {
	EdgeSeam []bool
	Paths    [][]int
}

func PlanSeams(m *Mesh, tp *Topology, sig *Signals) *SeamPlan {
	plan := &SeamPlan{EdgeSeam: make([]bool, len(tp.Edge))}

	ring := m.Hints.NeckRing
	for i := range ring {
		a := ring[i]
		b := ring[(i+1)%len(ring)]
		if eid, ok := tp.EdgeID(a, b); ok {
			plan.EdgeSeam[eid] = true
		}
	}

	backWeight := func(eid int) float64 {
		if sig.Protected[eid] {
			return math.Inf(1)
		}
		e := tp.Edge[eid]
		mid := m.Pos[e.A].Add(m.Pos[e.B]).Mul(0.5)
		zPenalty := 3.2 * smoothStep(-0.18, 0.20, mid.Z)
		frontLengthPenalty := 0.35 * smoothStep(-0.55, 0.75, mid.Y)
		return sig.EdgeCost[eid] + zPenalty + frontLengthPenalty
	}

	if path, edges, ok := ShortestVertexPath(tp, m.Hints.Crown, m.Hints.Bottom, backWeight); ok {
		plan.Paths = append(plan.Paths, path)
		for _, eid := range edges {
			plan.EdgeSeam[eid] = true
		}
	}
	return plan
}

func smoothStep(a, b, x float64) float64 {
	t := clampFloat((x-a)/(b-a), 0, 1)
	return t * t * (3 - 2*t)
}

type pqItem struct {
	v    int
	dist float64
}

type priorityQueue []pqItem

func (q priorityQueue) Len() int            { return len(q) }
func (q priorityQueue) Less(i, j int) bool  { return q[i].dist < q[j].dist }
func (q priorityQueue) Swap(i, j int)       { q[i], q[j] = q[j], q[i] }
func (q *priorityQueue) Push(x interface{}) { *q = append(*q, x.(pqItem)) }
func (q *priorityQueue) Pop() interface{} {
	old := *q
	n := len(old)
	it := old[n-1]
	*q = old[:n-1]
	return it
}

func ShortestVertexPath(tp *Topology, src, dst int, weight func(eid int) float64) ([]int, []int, bool) {
	n := len(tp.Mesh.Pos)
	dist := make([]float64, n)
	parent := make([]int, n)
	parentEdge := make([]int, n)
	for i := range dist {
		dist[i] = math.Inf(1)
		parent[i] = -1
		parentEdge[i] = -1
	}
	dist[src] = 0
	pq := &priorityQueue{{v: src, dist: 0}}
	heap.Init(pq)
	for pq.Len() > 0 {
		it := heap.Pop(pq).(pqItem)
		if it.dist != dist[it.v] {
			continue
		}
		if it.v == dst {
			break
		}
		for _, eid := range tp.VtxEdges[it.v] {
			w := weight(eid)
			if math.IsInf(w, 1) || math.IsNaN(w) {
				continue
			}
			e := tp.Edge[eid]
			to := e.A
			if to == it.v {
				to = e.B
			}
			nd := it.dist + w
			if nd < dist[to] {
				dist[to] = nd
				parent[to] = it.v
				parentEdge[to] = eid
				heap.Push(pq, pqItem{v: to, dist: nd})
			}
		}
	}
	if math.IsInf(dist[dst], 1) {
		return nil, nil, false
	}
	var verts, edges []int
	cur := dst
	for cur != src && cur != -1 {
		verts = append(verts, cur)
		edges = append(edges, parentEdge[cur])
		cur = parent[cur]
	}
	if cur == -1 {
		return nil, nil, false
	}
	verts = append(verts, src)
	for i, j := 0, len(verts)-1; i < j; i, j = i+1, j-1 {
		verts[i], verts[j] = verts[j], verts[i]
	}
	for i, j := 0, len(edges)-1; i < j; i, j = i+1, j-1 {
		edges[i], edges[j] = edges[j], edges[i]
	}
	return verts, edges, true
}
