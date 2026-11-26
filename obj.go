package holefill

import (
	"bufio"
	"fmt"
	"os"
)

func WriteOBJ(path string, m *Mesh) error {
	if len(m.Norm) != len(m.V) {
		m.ComputeNormals()
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	for _, v := range m.V {
		fmt.Fprintf(w, "v %.6f %.6f %.6f\n", v.X, v.Y, v.Z)
	}
	for _, n := range m.Norm {
		fmt.Fprintf(w, "vn %.6f %.6f %.6f\n", n.X, n.Y, n.Z)
	}
	lastMaterial := ""
	for ti, t := range m.T {
		mat := "scan"
		if ti < len(m.Tags) {
			tag := m.Tags[ti]
			switch tag.Kind {
			case KindPatch:
				mat = "patch_" + tag.Class.String()
			case KindBottom:
				mat = "underside"
			case KindWall:
				mat = "wall"
			}
		}
		if mat != lastMaterial {
			fmt.Fprintf(w, "usemtl %s\n", mat)
			lastMaterial = mat
		}
		fmt.Fprintf(w, "f %d//%d %d//%d %d//%d\n",
			t[0]+1, t[0]+1, t[1]+1, t[1]+1, t[2]+1, t[2]+1)
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return f.Close()
}
