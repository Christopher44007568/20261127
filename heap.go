package qem

import "container/heap"

type heapItem struct {
	edge    Edge
	cost    float64
	version uint64
	index   int
}

type candidateHeap []*heapItem

func (h candidateHeap) Len() int { return len(h) }

func (h candidateHeap) Less(i, j int) bool {
	if h[i].cost != h[j].cost {
		return h[i].cost < h[j].cost
	}
	if h[i].edge.A != h[j].edge.A {
		return h[i].edge.A < h[j].edge.A
	}
	return h[i].edge.B < h[j].edge.B
}

func (h candidateHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}

func (h *candidateHeap) Push(x any) {
	item := x.(*heapItem)
	item.index = len(*h)
	*h = append(*h, item)
}

func (h *candidateHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*h = old[:n-1]
	return item
}

func (s *Simplifier) bumpEdge(e Edge) uint64 {
	s.edgeVer[e]++
	return s.edgeVer[e]
}

func (s *Simplifier) enqueueEdge(e Edge) {
	if _, ok := s.Topo.EdgeFaces[e]; !ok {
		return
	}
	rec, ok := s.prepareCollapse(e)
	if !ok {
		return
	}
	ver := s.edgeVer[e]
	if ver == 0 {
		ver = 1
		s.edgeVer[e] = ver
	}
	heap.Push(&s.heap, &heapItem{edge: e, cost: rec.Cost, version: ver})
}

func (s *Simplifier) rebuildHeap() error {
	s.heap = make(candidateHeap, 0, len(s.Topo.EdgeFaces))
	s.edgeVer = make(map[Edge]uint64, len(s.Topo.EdgeFaces))
	for e := range s.Topo.EdgeFaces {
		s.edgeVer[e] = 1
		rec, ok := s.prepareCollapse(e)
		if ok {
			heap.Push(&s.heap, &heapItem{edge: e, cost: rec.Cost, version: 1})
		}
	}
	s.heapReady = true
	return nil
}
