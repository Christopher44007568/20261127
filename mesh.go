package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

type Mesh struct {
	P   []V3
	N   []V3
	C   []RGB
	V   []float64
	Tri [][3]int
}

type BoundaryEdge struct {
	A, B int
}

func meshEdgeKey(a, b int) uint64 {
	if a > b {
		a, b = b, a
	}
	return uint64(uint32(a))<<32 | uint64(uint32(b))
}

func edgeKeyToAB(key uint64) (int, int) {
	return int(key >> 32), int(uint32(key))
}

func (m *Mesh) ComputeNormals() {
	m.N = make([]V3, len(m.P))
	for _, t := range m.Tri {
		if t[0] >= len(m.P) || t[1] >= len(m.P) || t[2] >= len(m.P) {
			continue
		}
		a := m.P[t[0]]
		b := m.P[t[1]]
		c := m.P[t[2]]
		fn := Cross(b.Sub(a), c.Sub(a))
		m.N[t[0]] = m.N[t[0]].Add(fn)
		m.N[t[1]] = m.N[t[1]].Add(fn)
		m.N[t[2]] = m.N[t[2]].Add(fn)
	}
	for i := range m.N {
		m.N[i] = Normalize(m.N[i])
		if Norm(m.N[i]) < 0.5 {
			m.N[i] = V3{0, 0, 1}
		}
	}
}

func (m *Mesh) FlipFaces() {
	for i := range m.Tri {
		m.Tri[i][1], m.Tri[i][2] = m.Tri[i][2], m.Tri[i][1]
	}
	for i := range m.N {
		m.N[i] = m.N[i].Mul(-1)
	}
}

func (m *Mesh) EdgeCounts() map[uint64]int {
	counts := map[uint64]int{}
	for _, t := range m.Tri {
		counts[meshEdgeKey(t[0], t[1])]++
		counts[meshEdgeKey(t[1], t[2])]++
		counts[meshEdgeKey(t[2], t[0])]++
	}
	return counts
}

func (m *Mesh) BoundaryEdges() []BoundaryEdge {
	var out []BoundaryEdge
	for key, count := range m.EdgeCounts() {
		if count == 1 {
			a, b := edgeKeyToAB(key)
			out = append(out, BoundaryEdge{A: a, B: b})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].A != out[j].A {
			return out[i].A < out[j].A
		}
		return out[i].B < out[j].B
	})
	return out
}

func (m *Mesh) BoundaryEdgeCount() int {
	n := 0
	for _, c := range m.EdgeCounts() {
		if c == 1 {
			n++
		}
	}
	return n
}

func (m *Mesh) RemoveSmallComponents(minTriangles int) int {
	if minTriangles <= 1 || len(m.Tri) == 0 {
		return 0
	}
	uf := newUnionFind(len(m.P))
	for _, t := range m.Tri {
		uf.union(t[0], t[1])
		uf.union(t[1], t[2])
	}
	counts := map[int]int{}
	for _, t := range m.Tri {
		counts[uf.find(t[0])]++
	}
	kept := m.Tri[:0]
	removed := 0
	for _, t := range m.Tri {
		if counts[uf.find(t[0])] >= minTriangles {
			kept = append(kept, t)
		} else {
			removed++
		}
	}
	m.Tri = kept
	m.CompactVertices()
	return removed
}

func MeshComponentStats(m *Mesh) (int, float64) {
	if len(m.Tri) == 0 {
		return 0, 0
	}
	uf := newUnionFind(len(m.P))
	for _, t := range m.Tri {
		uf.union(t[0], t[1])
		uf.union(t[1], t[2])
	}
	counts := map[int]int{}
	for _, t := range m.Tri {
		counts[uf.find(t[0])]++
	}
	largest := 0
	for _, n := range counts {
		if n > largest {
			largest = n
		}
	}
	return len(counts), float64(largest) / float64(len(m.Tri))
}

func (m *Mesh) CompactVertices() {
	used := make([]bool, len(m.P))
	for _, t := range m.Tri {
		for _, id := range t {
			if id >= 0 && id < len(used) {
				used[id] = true
			}
		}
	}
	remap := make([]int, len(m.P))
	next := 0
	for i, u := range used {
		if u {
			remap[i] = next
			next++
		} else {
			remap[i] = -1
		}
	}
	p := make([]V3, 0, next)
	n := make([]V3, 0, next)
	c := make([]RGB, 0, next)
	v := make([]float64, 0, next)
	hasN := len(m.N) == len(m.P)
	hasC := len(m.C) == len(m.P)
	hasV := len(m.V) == len(m.P)
	for i, u := range used {
		if !u {
			continue
		}
		p = append(p, m.P[i])
		if hasN {
			n = append(n, m.N[i])
		}
		if hasC {
			c = append(c, m.C[i])
		}
		if hasV {
			v = append(v, m.V[i])
		}
	}
	for ti := range m.Tri {
		for k, id := range m.Tri[ti] {
			m.Tri[ti][k] = remap[id]
		}
	}
	m.P = p
	if hasN {
		m.N = n
	} else {
		m.N = nil
	}
	if hasC {
		m.C = c
	} else {
		m.C = nil
	}
	if hasV {
		m.V = v
	} else {
		m.V = nil
	}
}

func WriteMeshPLY(path string, m *Mesh, withColor bool) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	fmt.Fprintf(w, "ply\nformat ascii 1.0\nelement vertex %d\n", len(m.P))
	fmt.Fprint(w, "property float x\nproperty float y\nproperty float z\n")
	fmt.Fprint(w, "property float nx\nproperty float ny\nproperty float nz\n")
	if withColor && len(m.C) == len(m.P) {
		fmt.Fprint(w, "property uchar red\nproperty uchar green\nproperty uchar blue\n")
	}
	fmt.Fprintf(w, "element face %d\nproperty list uchar int vertex_indices\nend_header\n", len(m.Tri))
	for i, p := range m.P {
		n := V3{0, 0, 1}
		if i < len(m.N) {
			n = m.N[i]
		}
		if withColor && i < len(m.C) {
			c := m.C[i]
			fmt.Fprintf(w, "%.7f %.7f %.7f %.6f %.6f %.6f %d %d %d\n",
				p.X, p.Y, p.Z, n.X, n.Y, n.Z, c.R, c.G, c.B)
		} else {
			fmt.Fprintf(w, "%.7f %.7f %.7f %.6f %.6f %.6f\n", p.X, p.Y, p.Z, n.X, n.Y, n.Z)
		}
	}
	for _, t := range m.Tri {
		fmt.Fprintf(w, "3 %d %d %d\n", t[0], t[1], t[2])
	}
	return w.Flush()
}

func WriteMeshOBJ(path string, m *Mesh) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	fmt.Fprintln(w, "# Pure Go reconstructed open surface")
	for _, p := range m.P {
		fmt.Fprintf(w, "v %.7f %.7f %.7f\n", p.X, p.Y, p.Z)
	}
	for _, n := range m.N {
		fmt.Fprintf(w, "vn %.7f %.7f %.7f\n", n.X, n.Y, n.Z)
	}
	for _, t := range m.Tri {
		fmt.Fprintf(w, "f %d//%d %d//%d %d//%d\n",
			t[0]+1, t[0]+1, t[1]+1, t[1]+1, t[2]+1, t[2]+1)
	}
	return w.Flush()
}
