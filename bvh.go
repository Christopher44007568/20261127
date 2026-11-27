package main

import (
	"math"
	"sort"
)

type AABB struct {
	Min, Max Vec3
}

func emptyAABB() AABB {
	inf := math.Inf(1)
	return AABB{Min: Vec3{inf, inf, inf}, Max: Vec3{-inf, -inf, -inf}}
}

func (b AABB) IncludePoint(p Vec3) AABB {
	b.Min.X = math.Min(b.Min.X, p.X)
	b.Min.Y = math.Min(b.Min.Y, p.Y)
	b.Min.Z = math.Min(b.Min.Z, p.Z)
	b.Max.X = math.Max(b.Max.X, p.X)
	b.Max.Y = math.Max(b.Max.Y, p.Y)
	b.Max.Z = math.Max(b.Max.Z, p.Z)
	return b
}

func (b AABB) IncludeBox(c AABB) AABB {
	b = b.IncludePoint(c.Min)
	b = b.IncludePoint(c.Max)
	return b
}

func (b AABB) SurfaceArea() float64 {
	x := b.Max.X - b.Min.X
	y := b.Max.Y - b.Min.Y
	z := b.Max.Z - b.Min.Z
	return 2 * (x*y + y*z + z*x)
}

func (b AABB) IntersectRay(origin, dir Vec3, maxT float64) bool {
	inv := Vec3{1 / dir.X, 1 / dir.Y, 1 / dir.Z}
	tx1 := (b.Min.X - origin.X) * inv.X
	tx2 := (b.Max.X - origin.X) * inv.X
	tmin := math.Min(tx1, tx2)
	tmax := math.Max(tx1, tx2)
	ty1 := (b.Min.Y - origin.Y) * inv.Y
	ty2 := (b.Max.Y - origin.Y) * inv.Y
	tmin = math.Max(tmin, math.Min(ty1, ty2))
	tmax = math.Min(tmax, math.Max(ty1, ty2))
	tz1 := (b.Min.Z - origin.Z) * inv.Z
	tz2 := (b.Max.Z - origin.Z) * inv.Z
	tmin = math.Max(tmin, math.Min(tz1, tz2))
	tmax = math.Min(tmax, math.Max(tz1, tz2))
	if tmax < math.Max(0.0001, tmin) {
		return false
	}
	return tmin < maxT && tmax > 0.0005
}

type bvhNode struct {
	Bounds      AABB
	Left, Right int
	FirstTri    int
	TriCount    int
}

type BVH struct {
	Mesh  *Mesh
	Tri   []int
	Nodes []bvhNode
}

func BuildBVH(m *Mesh) *BVH {
	b := &BVH{Mesh: m, Tri: make([]int, len(m.Tri))}
	for i := range b.Tri {
		b.Tri[i] = i
	}
	root := b.build(0, len(b.Tri))
	_ = root
	return b
}

func (b *BVH) triBounds(f int) AABB {
	t := b.Mesh.Tri[f]
	box := emptyAABB()
	box = box.IncludePoint(b.Mesh.Pos[t.A])
	box = box.IncludePoint(b.Mesh.Pos[t.B])
	box = box.IncludePoint(b.Mesh.Pos[t.C])
	return box
}

func (b *BVH) build(start, count int) int {
	id := len(b.Nodes)
	b.Nodes = append(b.Nodes, bvhNode{})
	box := emptyAABB()
	centers := make([]Vec3, count)
	for i := 0; i < count; i++ {
		f := b.Tri[start+i]
		tb := b.triBounds(f)
		box = box.IncludeBox(tb)
		a, c := b.Mesh.Pos[b.Mesh.Tri[f].A], b.Mesh.Pos[b.Mesh.Tri[f].C]
		_ = c
		centers[i] = tb.Min.Add(tb.Max).Mul(0.5)
		_ = a
	}
	if count <= 6 {
		b.Nodes[id] = bvhNode{Bounds: box, FirstTri: start, TriCount: count}
		return id
	}
	axis := 0
	ext := []float64{box.Max.X - box.Min.X, box.Max.Y - box.Min.Y, box.Max.Z - box.Min.Z}
	if ext[1] > ext[axis] {
		axis = 1
	}
	if ext[2] > ext[axis] {
		axis = 2
	}
	sort.SliceStable(b.Tri[start:start+count], func(i, j int) bool {
		fi := b.Tri[start+i]
		fj := b.Tri[start+j]
		ci := b.triBounds(fi).Min.Add(b.triBounds(fi).Max).Mul(0.5)
		cj := b.triBounds(fj).Min.Add(b.triBounds(fj).Max).Mul(0.5)
		return coord(ci, axis) < coord(cj, axis)
	})
	mid := count / 2
	left := b.build(start, mid)
	right := b.build(start+mid, count-mid)
	b.Nodes[id] = bvhNode{Bounds: box, Left: left, Right: right}
	return id
}

func coord(p Vec3, axis int) float64 {
	switch axis {
	case 1:
		return p.Y
	case 2:
		return p.Z
	default:
		return p.X
	}
}

// Occluded reports a nearest hit within maxT. Triangles are double-sided so
// low-resolution vertex samples remain useful inside folds.
func (b *BVH) Occluded(origin, dir Vec3, maxT float64) (bool, float64) {
	if len(b.Nodes) == 0 {
		return false, maxT
	}
	return b.hitNode(0, origin, dir, maxT)
}

func (b *BVH) hitNode(id int, origin, dir Vec3, maxT float64) (bool, float64) {
	n := b.Nodes[id]
	if !n.Bounds.IntersectRay(origin, dir, maxT) {
		return false, maxT
	}
	if n.TriCount > 0 {
		hit := false
		best := maxT
		for i := n.FirstTri; i < n.FirstTri+n.TriCount; i++ {
			if t, ok := b.intersectTri(b.Tri[i], origin, dir); ok && t > 0.001 && t < best {
				hit = true
				best = t
			}
		}
		return hit, best
	}
	h1, t1 := b.hitNode(n.Left, origin, dir, maxT)
	h2, t2 := b.hitNode(n.Right, origin, dir, maxT)
	if h2 && (!h1 || t2 < t1) {
		return true, t2
	}
	return h1, t1
}

func (b *BVH) intersectTri(f int, origin, dir Vec3) (float64, bool) {
	t := b.Mesh.Tri[f]
	a := b.Mesh.Pos[t.A]
	e1 := b.Mesh.Pos[t.B].Sub(a)
	e2 := b.Mesh.Pos[t.C].Sub(a)
	p := dir.Cross(e2)
	det := e1.Dot(p)
	if math.Abs(det) < 1e-12 {
		return 0, false
	}
	inv := 1 / det
	tv := origin.Sub(a)
	u := tv.Dot(p) * inv
	if u < 0 || u > 1 {
		return 0, false
	}
	q := tv.Cross(e1)
	v := dir.Dot(q) * inv
	if v < 0 || u+v > 1 {
		return 0, false
	}
	tt := e2.Dot(q) * inv
	return tt, tt > 0.0001
}
