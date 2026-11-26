package main

import (
	"fmt"
	"math"
	"os"
	"strings"
)

// Mesh is a triangle soup with per-vertex normals and a per-face
// component id (tube of branch k, or a joint sphere).
type Mesh struct {
	V    []Vec3
	N    []Vec3
	F    [][3]int
	Comp []int
}

func (m *Mesh) addVertex(p Vec3) int {
	m.V = append(m.V, p)
	return len(m.V) - 1
}

func (m *Mesh) addFace(a, b, c, comp int) {
	if a == b || b == c || a == c {
		return
	}
	m.F = append(m.F, [3]int{a, b, c})
	m.Comp = append(m.Comp, comp)
}

// SphereRef marks a joint sphere used to weld a trimmed tube end onto.
type SphereRef struct {
	C Vec3
	R float64
}

// AddTube sweeps a circular cross-section along the polyline using the
// given parallel-transport frames. If project is set, the first ring is
// projected onto the joint sphere so the seam closes without a gap.
func (m *Mesh) AddTube(P []Vec3, R []float64, nrm, bin []Vec3, sides int,
	capStart, capEnd bool, comp int, project *SphereRef) {
	n := len(P)
	if n < 2 {
		return
	}
	ring := make([][]int, n)
	for i := 0; i < n; i++ {
		ring[i] = make([]int, sides)
		for j := 0; j < sides; j++ {
			th := 2 * math.Pi * float64(j) / float64(sides)
			off := nrm[i].Scale(math.Cos(th)).Add(bin[i].Scale(math.Sin(th)))
			v := P[i].Add(off.Scale(R[i]))
			if project != nil && i == 0 {
				v = project.C.Add(v.Sub(project.C).Norm().Scale(project.R))
			}
			ring[i][j] = m.addVertex(v)
		}
	}
	for i := 0; i < n-1; i++ {
		for j := 0; j < sides; j++ {
			j1 := (j + 1) % sides
			m.addFace(ring[i][j], ring[i][j1], ring[i+1][j], comp)
			m.addFace(ring[i][j1], ring[i+1][j1], ring[i+1][j], comp)
		}
	}
	t := Tangents(P)
	if capStart {
		tip := m.addVertex(P[0].Sub(t[0].Scale(R[0] * 0.7)))
		for j := 0; j < sides; j++ {
			j1 := (j + 1) % sides
			m.addFace(tip, ring[0][j1], ring[0][j], comp)
		}
	}
	if capEnd {
		tip := m.addVertex(P[n-1].Add(t[n-1].Scale(R[n-1] * 0.7)))
		for j := 0; j < sides; j++ {
			j1 := (j + 1) % sides
			m.addFace(tip, ring[n-1][j], ring[n-1][j1], comp)
		}
	}
}

// AddSphere appends a UV sphere (joint filler).
func (m *Mesh) AddSphere(c Vec3, r float64, nu, nv, comp int) {
	grid := make([][]int, nv+1)
	for iv := 0; iv <= nv; iv++ {
		phi := math.Pi * float64(iv) / float64(nv)
		grid[iv] = make([]int, nu)
		for iu := 0; iu < nu; iu++ {
			th := 2 * math.Pi * float64(iu) / float64(nu)
			p := c.Add(Vec3{
				r * math.Sin(phi) * math.Cos(th),
				r * math.Sin(phi) * math.Sin(th),
				r * math.Cos(phi),
			})
			grid[iv][iu] = m.addVertex(p)
		}
	}
	for iv := 0; iv < nv; iv++ {
		for iu := 0; iu < nu; iu++ {
			iu1 := (iu + 1) % nu
			a, b := grid[iv][iu], grid[iv][iu1]
			c2, d := grid[iv+1][iu], grid[iv+1][iu1]
			if iv > 0 {
				m.addFace(a, c2, b, comp)
			}
			if iv < nv-1 {
				m.addFace(b, c2, d, comp)
			}
		}
	}
}

// ComputeNormals builds area-weighted vertex normals.
func (m *Mesh) ComputeNormals() {
	m.N = make([]Vec3, len(m.V))
	for _, f := range m.F {
		n := m.V[f[1]].Sub(m.V[f[0]]).Cross(m.V[f[2]].Sub(m.V[f[0]]))
		for k := 0; k < 3; k++ {
			m.N[f[k]] = m.N[f[k]].Add(n)
		}
	}
	for i := range m.N {
		m.N[i] = m.N[i].Norm()
	}
}

func (m *Mesh) SaveOBJ(path string) error {
	var sb strings.Builder
	sb.WriteString("# tubegen output\n")
	for _, v := range m.V {
		fmt.Fprintf(&sb, "v %.6f %.6f %.6f\n", v.X, v.Y, v.Z)
	}
	lastComp := -1
	for i, f := range m.F {
		if m.Comp[i] != lastComp {
			lastComp = m.Comp[i]
			fmt.Fprintf(&sb, "o comp_%d\n", lastComp)
		}
		fmt.Fprintf(&sb, "f %d %d %d\n", f[0]+1, f[1]+1, f[2]+1)
	}
	return os.WriteFile(path, []byte(sb.String()), 0644)
}
