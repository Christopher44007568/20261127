package main

import "math"

// CrackValues computes a per-vertex "normal crack" measure in radians:
// intra-component it is the max dihedral angle across edges touching the
// vertex; near joints it also measures the angle between the vertex normal
// and the nearest face normal of a different component (the seam).
func (m *Mesh) CrackValues(joints []JointInfo) []float64 {
	faceN := make([]Vec3, len(m.F))
	cent := make([]Vec3, len(m.F))
	for i, f := range m.F {
		a, b, c := m.V[f[0]], m.V[f[1]], m.V[f[2]]
		faceN[i] = b.Sub(a).Cross(c.Sub(a)).Norm()
		cent[i] = a.Add(b).Add(c).Scale(1.0 / 3)
	}
	crack := make([]float64, len(m.V))
	edgeFaces := map[[2]int][]int{}
	for i, f := range m.F {
		for k := 0; k < 3; k++ {
			a, b := f[k], f[(k+1)%3]
			if a > b {
				a, b = b, a
			}
			edgeFaces[[2]int{a, b}] = append(edgeFaces[[2]int{a, b}], i)
		}
	}
	for e, fs := range edgeFaces {
		if len(fs) != 2 {
			continue
		}
		ang := angleBetween(faceN[fs[0]], faceN[fs[1]])
		if ang > crack[e[0]] {
			crack[e[0]] = ang
		}
		if ang > crack[e[1]] {
			crack[e[1]] = ang
		}
	}
	if len(joints) == 0 {
		return crack
	}
	vertComp := make([]int, len(m.V))
	for i := range vertComp {
		vertComp[i] = -1
	}
	for i, f := range m.F {
		for k := 0; k < 3; k++ {
			if vertComp[f[k]] < 0 {
				vertComp[f[k]] = m.Comp[i]
			}
		}
	}
	for _, j := range joints {
		zone := 2.6 * j.SphereR
		var verts []int
		for i, v := range m.V {
			if v.Dist(j.Pos) < zone {
				verts = append(verts, i)
			}
		}
		var faces []int
		for i := range m.F {
			if cent[i].Dist(j.Pos) < zone*1.2 {
				faces = append(faces, i)
			}
		}
		for _, vi := range verts {
			best := math.Inf(1)
			bestF := -1
			for _, fi := range faces {
				if m.Comp[fi] == vertComp[vi] {
					continue
				}
				if d := m.V[vi].Dist(cent[fi]); d < best {
					best = d
					bestF = fi
				}
			}
			if bestF >= 0 && best < 0.6*j.SphereR {
				if ang := angleBetween(m.N[vi], faceN[bestF]); ang > crack[vi] {
					crack[vi] = ang
				}
			}
		}
	}
	return crack
}

// HeatColor maps t in [0,1] to a jet-like blue-cyan-green-yellow-red ramp.
func HeatColor(t float64) [3]float64 {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	clamp := func(x float64) float64 {
		return math.Max(0, math.Min(1, x))
	}
	return [3]float64{
		clamp(1.5 - math.Abs(4*t-3)),
		clamp(1.5 - math.Abs(4*t-2)),
		clamp(1.5 - math.Abs(4*t-1)),
	}
}
