package main

import (
	"fmt"
	"math"
	"os"
	"strings"
)

// detailDisp adds cloth-fold waves and a hard-edged square-wave step (high poly only).
func detailDisp(u, v float64) float64 {
	d := 0.05 * math.Sin(14*2*math.Pi*v) * math.Sin(4*2*math.Pi*u)
	if frac(u*8) < 0.5 {
		d += 0.035
	}
	return d
}

// dimpleDisp is a deep concave dent shared by low and high poly; it creates
// self-occluding ray paths near its rim.
func dimpleDisp(u, v float64) float64 {
	du := math.Abs(u - 0.62)
	du = math.Min(du, 1-du)
	dv := v - 0.15
	return -0.35 * math.Exp(-(du*du+dv*dv)/0.0015)
}

// genTorus builds a torus grid with duplicated seam vertices. When mirrorUV is
// set, a rectangular UV patch is flipped so its triangles get negative winding.
func genTorus(segU, segV int, R, r float64, disp func(u, v float64) float64, mirrorUV bool) *Mesh {
	nx := segU + 1
	m := &Mesh{}
	for j := 0; j <= segV; j++ {
		v := float64(j) / float64(segV)
		for i := 0; i <= segU; i++ {
			u := float64(i) / float64(segU)
			cu, su := math.Cos(2*math.Pi*u), math.Sin(2*math.Pi*u)
			cv, sv := math.Cos(2*math.Pi*v), math.Sin(2*math.Pi*v)
			rad := r + disp(u, v)
			m.Pos = append(m.Pos, Vec3{(R + rad*cv) * cu, (R + rad*cv) * su, rad * sv})
			uu := u
			if mirrorUV && u > 0.3 && u < 0.55 && v > 0.35 && v < 0.6 {
				uu = 0.3 + (0.55 - u)
			}
			m.UV = append(m.UV, Vec2{uu, v})
		}
	}
	idx := func(i, j int) int { return j*nx + i }
	for j := 0; j < segV; j++ {
		for i := 0; i < segU; i++ {
			a, b, c, d := idx(i, j), idx(i+1, j), idx(i+1, j+1), idx(i, j+1)
			m.Tris = append(m.Tris, Tri{[3]int{a, b, c}}, Tri{[3]int{a, c, d}})
		}
	}
	m.computeNormals()
	return m
}

func writeOBJ(path string, m *Mesh) error {
	var b strings.Builder
	for _, p := range m.Pos {
		fmt.Fprintf(&b, "v %g %g %g\n", p.X, p.Y, p.Z)
	}
	for _, t := range m.UV {
		fmt.Fprintf(&b, "vt %g %g\n", t.X, t.Y)
	}
	for _, n := range m.Nrm {
		fmt.Fprintf(&b, "vn %g %g %g\n", n.X, n.Y, n.Z)
	}
	for _, t := range m.Tris {
		a, bb, c := t.I[0]+1, t.I[1]+1, t.I[2]+1
		fmt.Fprintf(&b, "f %d/%d/%d %d/%d/%d %d/%d/%d\n", a, a, a, bb, bb, bb, c, c, c)
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
