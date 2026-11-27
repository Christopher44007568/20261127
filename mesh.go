package main

import (
	"sort"
)

type Quad struct {
	V    [4]int
	Diag int
}

type Triangle struct {
	V      [3]int
	Repair bool
}

type Topology struct {
	Quads             []Quad
	NaiveQuads        [][4]int
	Tris              []Triangle
	NaiveTris         []Triangle
	TransitionQuads   int
	RepairTriangles   int
	SkippedDegenerate int
}

type Mesh struct {
	Pos     []Vec3
	H       []float64
	Depth   []int
	Feature []bool
	Tris    []Triangle
	Scene   Scene
}

func ExtractTopology(o *Octree) *Topology {
	o.PrepareDualVertices()
	t := &Topology{}
	seen := map[[4]int]bool{}
	naiveSeen := map[[4]int]bool{}
	triSeen := map[[3]int]bool{}
	var extraTris []Triangle

	for _, e := range o.ActiveEdges() {
		cross, ok := o.Crosses[e]
		if !ok {
			continue
		}
		poly, ok := o.incidentPolygon(cross.Point, e, cross.Normal)
		if !ok {
			t.SkippedDegenerate++
			continue
		}
		length := edgeLengthUnits(e)
		edgeDepth := o.MaxDepth
		for l := length; l > 1; l >>= 1 {
			edgeDepth--
		}
		sameDepth := true
		for _, id := range poly {
			if o.Leaves[id].Depth != edgeDepth {
				sameDepth = false
			}
		}
		if len(poly) == 4 {
			ids := [4]int(poly)
			key := quadSortedKey(ids)
			if sameDepth && !naiveSeen[key] {
				naiveSeen[key] = true
				t.NaiveQuads = append(t.NaiveQuads, ids)
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			diag := chooseDiagonal(o, ids)
			t.Quads = append(t.Quads, Quad{V: ids, Diag: diag})
			if !sameDepth {
				t.TransitionQuads++
			}
		} else if len(poly) == 3 {
			ids := [3]int(poly)
			key := triSortedKey(ids)
			if triSeen[key] {
				continue
			}
			triSeen[key] = true
			t.TransitionQuads++
			t.RepairTriangles++
			extraTris = append(extraTris, Triangle{V: ids, Repair: true})
		}
	}

	for _, q := range t.NaiveQuads {
		t.NaiveTris = append(t.NaiveTris,
			Triangle{V: [3]int{q[0], q[1], q[2]}},
			Triangle{V: [3]int{q[0], q[2], q[3]}},
		)
	}
	for _, q := range t.Quads {
		if q.Diag == 0 {
			t.Tris = append(t.Tris,
				Triangle{V: [3]int{q.V[0], q.V[1], q.V[2]}},
				Triangle{V: [3]int{q.V[0], q.V[2], q.V[3]}},
			)
		} else {
			t.Tris = append(t.Tris,
				Triangle{V: [3]int{q.V[0], q.V[1], q.V[3]}},
				Triangle{V: [3]int{q.V[1], q.V[2], q.V[3]}},
			)
		}
	}
	t.Tris = append(t.Tris, extraTris...)
	return t
}

func triSortedKey(t [3]int) [3]int {
	sort.Ints(t[:])
	return t
}

func (o *Octree) incidentPolygon(at Vec3, e EdgeKey, outward Vec3) ([]int, bool) {
	axis := edgeAxis(e)
	if axis < 0 {
		return nil, false
	}
	u := (axis + 1) % 3
	v := (axis + 2) % 3
	base := at
	eps := o.HFinest * 0.24
	signs := [4][2]float64{{-1, -1}, {1, -1}, {1, 1}, {-1, 1}}
	nodes := [4]*Node{}
	var ids [4]int
	for i, s := range signs {
		p := base
		setCoordPtr(&p, u, coord(base, u)+s[0]*eps)
		setCoordPtr(&p, v, coord(base, v)+s[1]*eps)
		nodes[i] = o.leafAtWorld(p)
		if nodes[i] == nil {
			return nil, false
		}
		ids[i] = nodes[i].Index
	}

	poly := []int{ids[0]}
	for i := 1; i < 4; i++ {
		if ids[i] != ids[i-1] {
			poly = append(poly, ids[i])
		}
	}
	if len(poly) > 1 && poly[0] == poly[len(poly)-1] {
		poly = poly[:len(poly)-1]
	}
	if len(poly) < 3 || len(poly) > 4 {
		return nil, false
	}
	unique := map[int]bool{}
	for _, id := range poly {
		if unique[id] {
			return nil, false
		}
		unique[id] = true
	}

	p0 := o.Leaves[poly[0]].Sharp
	p1 := o.Leaves[poly[1]].Sharp
	p2 := o.Leaves[poly[2]].Sharp
	geoN := p1.Sub(p0).Cross(p2.Sub(p0))
	if geoN.Dot(outward) < 0 {
		if len(poly) == 4 {
			poly = []int{poly[0], poly[3], poly[2], poly[1]}
		} else {
			poly = []int{poly[0], poly[2], poly[1]}
		}
	}
	return poly, true
}

func quadSortedKey(q [4]int) [4]int {
	k := q
	sort.Ints(k[:])
	return k
}

func chooseDiagonal(o *Octree, q [4]int) int {
	err0 := triangulationError(o,
		[3]int{q[0], q[1], q[2]},
		[3]int{q[0], q[2], q[3]},
	)
	err1 := triangulationError(o,
		[3]int{q[0], q[1], q[3]},
		[3]int{q[1], q[2], q[3]},
	)
	if err1 < err0-1e-7 {
		return 1
	}
	return 0
}

func triangulationError(o *Octree, tris ...[3]int) float64 {
	score := 0.0
	for _, tri := range tris {
		c := V(0, 0, 0)
		for _, id := range tri {
			c = c.Add(o.Leaves[id].Sharp)
		}
		c = c.Div(3)
		score += absFloat(o.Scene.SDF(c))
	}
	return score
}

func absFloat(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func NewMesh(o *Octree, top *Topology, sharp bool) *Mesh {
	m := &Mesh{Scene: o.Scene, Tris: append([]Triangle(nil), top.Tris...)}
	m.Pos = make([]Vec3, len(o.Leaves))
	m.H = make([]float64, len(o.Leaves))
	m.Depth = make([]int, len(o.Leaves))
	m.Feature = make([]bool, len(o.Leaves))
	for i, n := range o.Leaves {
		if sharp && n.ConstraintN > 0 {
			m.Pos[i] = n.Sharp
		} else {
			m.Pos[i] = n.Baseline
		}
		m.H[i] = n.H
		m.Depth[i] = n.Depth
		m.Feature[i] = n.Feature
	}
	return m
}

type EdgeUsage struct {
	Key   [2]int
	Count int
}

func BoundaryEdges(tris []Triangle) map[[2]int]int {
	counts := map[[2]int]int{}
	for _, tri := range tris {
		ids := tri.V
		for i := 0; i < 3; i++ {
			a := ids[i]
			b := ids[(i+1)%3]
			key := [2]int{a, b}
			if a > b {
				key = [2]int{b, a}
			}
			counts[key]++
		}
	}
	for k, c := range counts {
		if c == 2 {
			delete(counts, k)
		}
	}
	return counts
}

func BoundaryEdgeCount(tris []Triangle) int {
	return len(BoundaryEdges(tris))
}
