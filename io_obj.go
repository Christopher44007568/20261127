package main

import (
	"bufio"
	"fmt"
	"os"
)

func writeOBJFile(path string, m *Mesh, wedgeUV []Vec2, faceWedge [][3]int) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriter(f)
	defer w.Flush()

	fmt.Fprintln(w, "# Organic curvature/occlusion UV unwrap")
	fmt.Fprintf(w, "o %s\n", m.Name)
	for _, p := range m.Pos {
		fmt.Fprintf(w, "v %.6f %.6f %.6f\n", p.X, p.Y, p.Z)
	}
	for _, uv := range wedgeUV {
		fmt.Fprintf(w, "vt %.6f %.6f\n", uv.X, uv.Y)
	}
	for fi, t := range m.Tri {
		ws := faceWedge[fi]
		fmt.Fprintf(w, "f %d/%d %d/%d %d/%d\n",
			t.A+1, ws[0]+1,
			t.B+1, ws[1]+1,
			t.C+1, ws[2]+1)
	}
	if err := w.Flush(); err != nil {
		_ = f.Close()
		return err
	}
	return f.Close()
}
