package main

import (
	"math"
	"sort"
)

type GridPoint [3]int

type Node struct {
	Index       int
	Depth       int
	IX, IY, IZ  int
	Units       int
	Min         Vec3
	Center      Vec3
	Size        float64
	H           float64
	Children    [8]*Node
	Leaf        bool
	VertexID    int
	CornerVals  [8]float64
	Spread      float64
	Feature     bool
	ConstraintN int
	Mass        Vec3
	Sharp       Vec3
	Baseline    Vec3
}

type EdgeKey struct {
	A GridPoint
	B GridPoint
}

type CrossInfo struct {
	Point  Vec3
	Normal Vec3
}

type Octree struct {
	Scene    Scene
	Root     *Node
	MaxDepth int
	N        int
	HFinest  float64
	Leaves   []*Node
	Corner   map[GridPoint]float64
	Edges    map[EdgeKey]struct{}
	Crosses  map[EdgeKey]CrossInfo
}

var cornerBits = [8][3]int{
	{0, 0, 0}, {1, 0, 0}, {0, 1, 0}, {1, 1, 0},
	{0, 0, 1}, {1, 0, 1}, {0, 1, 1}, {1, 1, 1},
}

var cubeEdges = [12][2]int{
	{0, 1}, {2, 3}, {4, 5}, {6, 7},
	{0, 2}, {1, 3}, {4, 6}, {5, 7},
	{0, 4}, {1, 5}, {2, 6}, {3, 7},
}

func NewOctree(scene Scene) *Octree {
	depth := scene.MaxDepth
	if depth <= 0 {
		depth = 7
	}
	n := 1 << depth
	o := &Octree{
		Scene:    scene,
		MaxDepth: depth,
		N:        n,
		HFinest:  2 / float64(n),
		Corner:   make(map[GridPoint]float64),
		Edges:    make(map[EdgeKey]struct{}),
		Crosses:  make(map[EdgeKey]CrossInfo),
	}
	root := &Node{
		Depth: 0,
		IX:    0, IY: 0, IZ: 0,
		Units:  n,
		Min:    V(-1, -1, -1),
		Center: V(0, 0, 0),
		Size:   2,
		H:      2,
	}
	o.Root = root
	o.build(root)
	return o
}

func (o *Octree) gridWorld(i int) float64 {
	return -1 + 2*float64(i)/float64(o.N)
}

func (o *Octree) pointWorld(p GridPoint) Vec3 {
	return V(o.gridWorld(p[0]), o.gridWorld(p[1]), o.gridWorld(p[2]))
}

func (o *Octree) cornerValue(p GridPoint) float64 {
	if v, ok := o.Corner[p]; ok {
		return v
	}
	v := o.Scene.SDF(o.pointWorld(p))
	o.Corner[p] = v
	return v
}

func (o *Octree) nodeCornerGrid(n *Node, ci int) GridPoint {
	b := cornerBits[ci]
	return GridPoint{
		n.IX + b[0]*n.Units,
		n.IY + b[1]*n.Units,
		n.IZ + b[2]*n.Units,
	}
}

func (o *Octree) readCorners(n *Node) [8]float64 {
	var vals [8]float64
	for i := 0; i < 8; i++ {
		vals[i] = o.cornerValue(o.nodeCornerGrid(n, i))
	}
	n.CornerVals = vals
	return vals
}

func (o *Octree) build(n *Node) {
	n.Min = V(o.gridWorld(n.IX), o.gridWorld(n.IY), o.gridWorld(n.IZ))
	n.Size = 2 * float64(n.Units) / float64(o.N)
	n.H = n.Size
	n.Center = n.Min.Add(V(n.Size/2, n.Size/2, n.Size/2))
	vals := o.readCorners(n)

	if o.shouldRefine(n, vals) {
		n.Leaf = false
		half := n.Units / 2
		for i := 0; i < 8; i++ {
			b := cornerBits[i]
			child := &Node{
				Depth: n.Depth + 1,
				IX:    n.IX + b[0]*half,
				IY:    n.IY + b[1]*half,
				IZ:    n.IZ + b[2]*half,
				Units: half,
			}
			n.Children[i] = child
			o.build(child)
		}
		return
	}

	n.Leaf = true
	n.Index = len(o.Leaves)
	n.VertexID = n.Index
	o.Leaves = append(o.Leaves, n)
}

func (o *Octree) shouldRefine(n *Node, vals [8]float64) bool {
	if n.Depth >= o.Scene.MaxDepth {
		return false
	}

	minV := vals[0]
	maxV := vals[0]
	for _, v := range vals[1:] {
		minV = math.Min(minV, v)
		maxV = math.Max(maxV, v)
	}
	crosses := minV <= 0 && maxV > 0
	centerD := o.Scene.SDF(n.Center)
	radius := 0.8660254037844386 * n.Size
	surfaceNear := crosses || math.Abs(centerD) <= radius+n.H*0.25
	if !surfaceNear {
		return false
	}
	if n.Depth < o.Scene.MinDepth {
		return true
	}

	normals := make([]Vec3, 0, 9)
	normals = append(normals, o.Scene.Normal(n.Center))
	for ci := 0; ci < 8; ci++ {
		p := o.pointWorld(o.nodeCornerGrid(n, ci))
		normals = append(normals, o.Scene.Normal(p))
	}
	minDot := 1.0
	for i := 0; i < len(normals); i++ {
		for j := i + 1; j < len(normals); j++ {
			minDot = math.Min(minDot, normals[i].Dot(normals[j]))
		}
	}
	spread := 1 - clampFloat(minDot, -1, 1)
	n.Spread = spread

	switch o.Scene.Kind {
	case KindBox:
		n.Feature = spread > 0.10
		if n.Feature {
			return true
		}
	case KindShell:
		n.Feature = spread > 0.08
		if n.Size > o.Scene.Thickness*0.24 {
			return true
		}
		if n.Feature {
			return true
		}
	case KindSphere:
		n.Feature = spread > 0.010
		if n.Feature {
			return true
		}
	}
	return false
}

func edgeKey(a, b GridPoint) EdgeKey {
	if gridLess(b, a) {
		a, b = b, a
	}
	return EdgeKey{A: a, B: b}
}

func gridLess(a, b GridPoint) bool {
	for i := 0; i < 3; i++ {
		if a[i] != b[i] {
			return a[i] < b[i]
		}
	}
	return false
}

func (o *Octree) DepthHistogram() map[int]int {
	out := map[int]int{}
	for _, n := range o.Leaves {
		out[n.Depth]++
	}
	return out
}

func sortedDepthCounts(h map[int]int) []struct {
	Depth int
	Count int
} {
	var out []struct {
		Depth int
		Count int
	}
	for d, c := range h {
		out = append(out, struct {
			Depth int
			Count int
		}{d, c})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Depth < out[j].Depth })
	return out
}
