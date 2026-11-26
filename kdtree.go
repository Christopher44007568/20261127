package main

import (
	"container/heap"
	"math"
	"sort"
)

type Neighbor struct {
	ID    int
	Dist2 float64
}

type neighborHeap []Neighbor

func (h neighborHeap) Len() int            { return len(h) }
func (h neighborHeap) Less(i, j int) bool  { return h[i].Dist2 > h[j].Dist2 }
func (h neighborHeap) Swap(i, j int)       { h[i], h[j] = h[j], h[i] }
func (h *neighborHeap) Push(x interface{}) { *h = append(*h, x.(Neighbor)) }

func (h *neighborHeap) Pop() interface{} {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

type kdNode struct {
	id    int
	axis  int
	left  int
	right int
}

type KDTree struct {
	points []V3
	nodes  []kdNode
	root   int
}

func NewKDTree(points []V3) *KDTree {
	t := &KDTree{points: points}
	if len(points) == 0 {
		t.root = -1
		return t
	}
	ids := make([]int, len(points))
	for i := range ids {
		ids[i] = i
	}
	t.root = t.build(ids)
	return t
}

func (t *KDTree) coordinate(id, axis int) float64 {
	switch axis {
	case 0:
		return t.points[id].X
	case 1:
		return t.points[id].Y
	default:
		return t.points[id].Z
	}
}

func (t *KDTree) chooseAxis(ids []int) int {
	if len(ids) <= 1 {
		return 0
	}
	min := [3]float64{math.Inf(1), math.Inf(1), math.Inf(1)}
	max := [3]float64{math.Inf(-1), math.Inf(-1), math.Inf(-1)}
	for _, id := range ids {
		p := t.points[id]
		v := [3]float64{p.X, p.Y, p.Z}
		for a := 0; a < 3; a++ {
			min[a] = math.Min(min[a], v[a])
			max[a] = math.Max(max[a], v[a])
		}
	}
	best := 0
	bestSpread := -1.0
	for a := 0; a < 3; a++ {
		if s := max[a] - min[a]; s > bestSpread {
			bestSpread = s
			best = a
		}
	}
	return best
}

func (t *KDTree) build(ids []int) int {
	if len(ids) == 0 {
		return -1
	}
	axis := t.chooseAxis(ids)
	sort.Slice(ids, func(i, j int) bool {
		ai := t.coordinate(ids[i], axis)
		aj := t.coordinate(ids[j], axis)
		if ai != aj {
			return ai < aj
		}
		return ids[i] < ids[j]
	})
	mid := len(ids) / 2
	id := ids[mid]
	nodeID := len(t.nodes)
	t.nodes = append(t.nodes, kdNode{id: id, axis: axis, left: -1, right: -1})
	left := t.build(ids[:mid])
	right := t.build(ids[mid+1:])
	t.nodes[nodeID].left = left
	t.nodes[nodeID].right = right
	return nodeID
}

func (t *KDTree) searchKNN(node int, q V3, k int, h *neighborHeap) {
	if node < 0 {
		return
	}
	n := t.nodes[node]
	p := t.points[n.id]
	d2 := Norm2(q.Sub(p))
	if h.Len() < k {
		heap.Push(h, Neighbor{ID: n.id, Dist2: d2})
	} else if d2 < (*h)[0].Dist2 {
		heap.Pop(h)
		heap.Push(h, Neighbor{ID: n.id, Dist2: d2})
	}

	delta := t.coordinate(n.id, n.axis) - coordinateV3(q, n.axis)
	near, far := n.left, n.right
	if delta < 0 {
		near, far = n.right, n.left
	}
	t.searchKNN(near, q, k, h)
	if h.Len() < k || delta*delta < (*h)[0].Dist2 {
		t.searchKNN(far, q, k, h)
	}
}

func coordinateV3(p V3, axis int) float64 {
	switch axis {
	case 0:
		return p.X
	case 1:
		return p.Y
	default:
		return p.Z
	}
}

func (t *KDTree) KNN(q V3, k int) []Neighbor {
	if t == nil || t.root < 0 || k <= 0 {
		return nil
	}
	h := &neighborHeap{}
	heap.Init(h)
	t.searchKNN(t.root, q, k, h)
	out := append([]Neighbor(nil), (*h)...)
	sort.Slice(out, func(i, j int) bool { return out[i].Dist2 < out[j].Dist2 })
	return out
}

func (t *KDTree) Nearest(q V3) (Neighbor, bool) {
	ns := t.KNN(q, 1)
	if len(ns) == 0 {
		return Neighbor{}, false
	}
	return ns[0], true
}

func (t *KDTree) searchRadius(node int, q V3, r2 float64, out *[]int) {
	if node < 0 {
		return
	}
	n := t.nodes[node]
	p := t.points[n.id]
	d2 := Norm2(q.Sub(p))
	if d2 <= r2 {
		*out = append(*out, n.id)
	}
	delta := coordinateV3(q, n.axis) - t.coordinate(n.id, n.axis)
	near, far := n.left, n.right
	if delta > 0 {
		near, far = n.right, n.left
	}
	t.searchRadius(near, q, r2, out)
	if delta*delta <= r2 {
		t.searchRadius(far, q, r2, out)
	}
}

func (t *KDTree) RadiusSearch(q V3, radius float64) []int {
	if t == nil || t.root < 0 || radius < 0 {
		return nil
	}
	var out []int
	t.searchRadius(t.root, q, radius*radius, &out)
	return out
}
