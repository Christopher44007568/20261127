package main

import (
	"math"
	"sort"
)

type Tri struct{ I [3]int }

type Mesh struct {
	Pos  []Vec3
	Nrm  []Vec3
	UV   []Vec2
	Tris []Tri
}

// computeNormals accumulates area-weighted face normals at vertices.
func (m *Mesh) computeNormals() {
	m.Nrm = make([]Vec3, len(m.Pos))
	for _, t := range m.Tris {
		a, b, c := m.Pos[t.I[0]], m.Pos[t.I[1]], m.Pos[t.I[2]]
		fn := b.Sub(a).Cross(c.Sub(a))
		for _, i := range t.I {
			m.Nrm[i] = m.Nrm[i].Add(fn)
		}
	}
	for i := range m.Nrm {
		m.Nrm[i] = m.Nrm[i].Norm()
	}
}

type Hit struct {
	Tri  int
	T    float64
	U, V float64
}

type bvhNode struct {
	min, max     Vec3
	left, right  int
	start, count int
}

type BVH struct {
	nodes []bvhNode
	order []int
	mesh  *Mesh
}

func triBounds(m *Mesh, t Tri) (Vec3, Vec3) {
	mn := m.Pos[t.I[0]]
	mx := mn
	for _, i := range t.I {
		mn = vmin(mn, m.Pos[i])
		mx = vmax(mx, m.Pos[i])
	}
	return mn, mx
}

func NewBVH(m *Mesh) *BVH {
	b := &BVH{mesh: m, order: make([]int, len(m.Tris))}
	cents := make([]Vec3, len(m.Tris))
	for i := range b.order {
		b.order[i] = i
		mn, mx := triBounds(m, m.Tris[i])
		cents[i] = mn.Add(mx).Mul(0.5)
	}
	var build func(lo, hi int) int
	build = func(lo, hi int) int {
		idx := len(b.nodes)
		b.nodes = append(b.nodes, bvhNode{left: -1, right: -1})
		mn, mx := Vec3{1e30, 1e30, 1e30}, Vec3{-1e30, -1e30, -1e30}
		cmn, cmx := mn, mx
		for k := lo; k < hi; k++ {
			id := b.order[k]
			a, c := triBounds(m, m.Tris[id])
			mn = vmin(mn, a)
			mx = vmax(mx, c)
			cmn = vmin(cmn, cents[id])
			cmx = vmax(cmx, cents[id])
		}
		b.nodes[idx].min, b.nodes[idx].max = mn, mx
		if hi-lo <= 4 {
			b.nodes[idx].start, b.nodes[idx].count = lo, hi-lo
			return idx
		}
		d := cmx.Sub(cmn)
		axis := 0
		if d.Y > d.X && d.Y >= d.Z {
			axis = 1
		} else if d.Z > d.X && d.Z >= d.Y {
			axis = 2
		}
		sl := b.order[lo:hi]
		sort.Slice(sl, func(i, j int) bool {
			ca, cb := cents[sl[i]], cents[sl[j]]
			switch axis {
			case 1:
				return ca.Y < cb.Y
			case 2:
				return ca.Z < cb.Z
			}
			return ca.X < cb.X
		})
		mid := (lo + hi) / 2
		l := build(lo, mid)
		r := build(mid, hi)
		b.nodes[idx].left, b.nodes[idx].right = l, r
		return idx
	}
	build(0, len(b.order))
	return b
}

func slabHit(mn, mx, ro, inv Vec3, minT, maxT float64) bool {
	t0x := (mn.X - ro.X) * inv.X
	t1x := (mx.X - ro.X) * inv.X
	t0y := (mn.Y - ro.Y) * inv.Y
	t1y := (mx.Y - ro.Y) * inv.Y
	t0z := (mn.Z - ro.Z) * inv.Z
	t1z := (mx.Z - ro.Z) * inv.Z
	tmin := math.Max(math.Max(math.Min(t0x, t1x), math.Min(t0y, t1y)), math.Min(t0z, t1z))
	tmax := math.Min(math.Min(math.Max(t0x, t1x), math.Max(t0y, t1y)), math.Max(t0z, t1z))
	return tmax >= tmin && tmax >= minT && tmin <= maxT
}

// rayTri is two-sided Moller-Trumbore intersection.
func rayTri(ro, rd Vec3, m *Mesh, tri Tri, minT, maxT float64) (t, u, v float64, ok bool) {
	a, b, c := m.Pos[tri.I[0]], m.Pos[tri.I[1]], m.Pos[tri.I[2]]
	e1 := b.Sub(a)
	e2 := c.Sub(a)
	p := rd.Cross(e2)
	det := e1.Dot(p)
	if math.Abs(det) < 1e-15 {
		return 0, 0, 0, false
	}
	inv := 1 / det
	s := ro.Sub(a)
	u = s.Dot(p) * inv
	if u < -1e-9 || u > 1+1e-9 {
		return 0, 0, 0, false
	}
	q := s.Cross(e1)
	v = rd.Dot(q) * inv
	if v < -1e-9 || u+v > 1+1e-9 {
		return 0, 0, 0, false
	}
	t = e2.Dot(q) * inv
	if t < minT || t > maxT {
		return 0, 0, 0, false
	}
	return t, u, v, true
}

// Intersect finds the nearest triangle hit in (minT, maxT]. ignore skips a triangle index (-1 for none).
func (b *BVH) Intersect(ro, rd Vec3, minT, maxT float64, ignore int) (Hit, bool) {
	inv := Vec3{1 / rd.X, 1 / rd.Y, 1 / rd.Z}
	best := maxT
	var hit Hit
	found := false
	stack := make([]int, 0, 64)
	stack = append(stack, 0)
	for len(stack) > 0 {
		ni := stack[len(stack)-1]
		stack = stack[:len(stack)-1]
		nd := b.nodes[ni]
		if !slabHit(nd.min, nd.max, ro, inv, minT, best) {
			continue
		}
		if nd.left < 0 {
			for k := 0; k < nd.count; k++ {
				ti := b.order[nd.start+k]
				if ti == ignore {
					continue
				}
				if t, u, v, ok := rayTri(ro, rd, b.mesh, b.mesh.Tris[ti], minT, best); ok {
					best = t
					hit = Hit{Tri: ti, T: t, U: u, V: v}
					found = true
				}
			}
		} else {
			stack = append(stack, nd.left, nd.right)
		}
	}
	return hit, found
}
