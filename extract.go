package main

import (
	"math"
	"sort"
)

type qef struct {
	a00, a01, a02 float64
	a11, a12, a22 float64
	b0, b1, b2    float64
	mass          Vec3
	count         int
}

func (q *qef) add(p, n Vec3) {
	q.a00 += n.X * n.X
	q.a01 += n.X * n.Y
	q.a02 += n.X * n.Z
	q.a11 += n.Y * n.Y
	q.a12 += n.Y * n.Z
	q.a22 += n.Z * n.Z
	d := n.Dot(p)
	q.b0 += n.X * d
	q.b1 += n.Y * d
	q.b2 += n.Z * d
	q.mass = q.mass.Add(p)
	q.count++
}

func (q *qef) solve(fallback Vec3, cell Node) Vec3 {
	if q.count == 0 {
		return fallback
	}
	mass := q.mass.Div(float64(q.count))
	reg := 1e-3 * float64(q.count)
	m := [3][3]float64{
		{q.a00 + reg, q.a01, q.a02},
		{q.a01, q.a11 + reg, q.a12},
		{q.a02, q.a12, q.a22 + reg},
	}
	rhs := V(q.b0+reg*mass.X, q.b1+reg*mass.Y, q.b2+reg*mass.Z)
	x := solve3(m, rhs)
	if math.IsNaN(x.X) || math.IsInf(x.X, 0) {
		x = mass
	}
	// A small expansion avoids clipping a true corner that lies on a voxel face.
	pad := cell.H * 0.02
	lo := cell.Min.Sub(V(pad, pad, pad))
	hi := cell.Min.Add(V(cell.Size+pad, cell.Size+pad, cell.Size+pad))
	x = x.Max(lo).Min(hi)
	return x
}

func solve3(m [3][3]float64, b Vec3) Vec3 {
	a := [3][4]float64{
		{m[0][0], m[0][1], m[0][2], b.X},
		{m[1][0], m[1][1], m[1][2], b.Y},
		{m[2][0], m[2][1], m[2][2], b.Z},
	}
	for col := 0; col < 3; col++ {
		pivot := col
		for r := col + 1; r < 3; r++ {
			if math.Abs(a[r][col]) > math.Abs(a[pivot][col]) {
				pivot = r
			}
		}
		if math.Abs(a[pivot][col]) < 1e-12 {
			return V(math.NaN(), math.NaN(), math.NaN())
		}
		a[col], a[pivot] = a[pivot], a[col]
		for r := 0; r < 3; r++ {
			if r == col {
				continue
			}
			f := a[r][col] / a[col][col]
			for c := col; c <= 3; c++ {
				a[r][c] -= f * a[col][c]
			}
		}
	}
	return V(a[0][3]/a[0][0], a[1][3]/a[1][1], a[2][3]/a[2][2])
}

func (o *Octree) PrepareDualVertices() {
	o.Edges = make(map[EdgeKey]struct{})
	for _, leaf := range o.Leaves {
		for _, e := range cubeEdges {
			ka := o.nodeCornerGrid(leaf, e[0])
			kb := o.nodeCornerGrid(leaf, e[1])
			o.Edges[edgeKey(ka, kb)] = struct{}{}
		}
	}

	o.Crosses = make(map[EdgeKey]CrossInfo)
	for e := range o.Edges {
		va := o.cornerValue(e.A)
		vb := o.cornerValue(e.B)
		if (va <= 0) == (vb <= 0) {
			continue
		}
		pa := o.pointWorld(e.A)
		pb := o.pointWorld(e.B)
		fa := va
		a, b := pa, pb
		for i := 0; i < 22; i++ {
			m := a.Add(b).Mul(0.5)
			fm := o.Scene.SDF(m)
			if (fa <= 0) == (fm <= 0) {
				a = m
				fa = fm
			} else {
				b = m
			}
		}
		p := a.Add(b).Mul(0.5)
		o.Crosses[e] = CrossInfo{Point: p, Normal: o.Scene.Normal(p)}
	}

	qefs := make([]qef, len(o.Leaves))
	for _, leaf := range o.Leaves {
		q := &qefs[leaf.Index]
		for _, e := range cubeEdges {
			ka := o.nodeCornerGrid(leaf, e[0])
			kb := o.nodeCornerGrid(leaf, e[1])
			key := edgeKey(ka, kb)
			cross, ok := o.Crosses[key]
			if !ok {
				continue
			}
			q.add(cross.Point, cross.Normal)
			leaf.ConstraintN++
			leaf.Mass = leaf.Mass.Add(cross.Point)
		}
		if q.count > 0 {
			leaf.Mass = leaf.Mass.Div(float64(q.count))
			leaf.Baseline = leaf.Mass
			leaf.Sharp = q.solve(leaf.Mass, *leaf)
		} else {
			leaf.Mass = leaf.Center
			leaf.Baseline = leaf.Center
			leaf.Sharp = leaf.Center
		}
		if leaf.Depth >= o.MaxDepth || leaf.Spread == 0 {
			o.evaluateFeature(leaf)
		}
	}
}

func (o *Octree) evaluateFeature(n *Node) {
	normals := []Vec3{o.Scene.Normal(n.Center)}
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
	n.Spread = 1 - clampFloat(minDot, -1, 1)
	threshold := 0.10
	if o.Scene.Kind == KindSphere {
		threshold = 0.010
	} else if o.Scene.Kind == KindShell {
		threshold = 0.08
	}
	n.Feature = n.Spread > threshold
}

func (o *Octree) ActiveEdges() []EdgeKey {
	active := make([]EdgeKey, 0, len(o.Edges))
	for e := range o.Edges {
		axis := edgeAxis(e)
		length := e.B[axis] - e.A[axis]
		superseded := false
		if length > 1 {
			mid := e.A
			mid[axis] += length / 2
			_, ok1 := o.Edges[edgeKey(e.A, mid)]
			_, ok2 := o.Edges[edgeKey(mid, e.B)]
			superseded = ok1 && ok2
		}
		if !superseded {
			active = append(active, e)
		}
	}
	sort.Slice(active, func(i, j int) bool {
		if active[i].A != active[j].A {
			return gridLess(active[i].A, active[j].A)
		}
		return gridLess(active[i].B, active[j].B)
	})
	return active
}

func edgeAxis(e EdgeKey) int {
	for i := 0; i < 3; i++ {
		if e.A[i] != e.B[i] {
			return i
		}
	}
	return -1
}

func edgeLengthUnits(e EdgeKey) int {
	axis := edgeAxis(e)
	return e.B[axis] - e.A[axis]
}

func (o *Octree) leafAtWorld(p Vec3) *Node {
	if p.X < -1 || p.X > 1 || p.Y < -1 || p.Y > 1 || p.Z < -1 || p.Z > 1 {
		return nil
	}
	n := o.Root
	for !n.Leaf {
		mid := []float64{n.Min.X + n.Size/2, n.Min.Y + n.Size/2, n.Min.Z + n.Size/2}
		idx := 0
		if p.X >= mid[0] {
			idx |= 1
		}
		if p.Y >= mid[1] {
			idx |= 2
		}
		if p.Z >= mid[2] {
			idx |= 4
		}
		n = n.Children[idx]
		if n == nil {
			return nil
		}
	}
	return n
}
