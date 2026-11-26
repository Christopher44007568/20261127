package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
)

func ensureDir(path string) error {
	return os.MkdirAll(filepath.Dir(path), 0o755)
}

func WriteMeshPLY(path string, mesh *Mesh) error {
	if err := ensureDir(path); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriterSize(f, 1<<20)
	fmt.Fprintln(w, "ply")
	fmt.Fprintln(w, "format ascii 1.0")
	fmt.Fprintf(w, "element vertex %d\n", len(mesh.Vertices))
	fmt.Fprintln(w, "property float x")
	fmt.Fprintln(w, "property float y")
	fmt.Fprintln(w, "property float z")
	fmt.Fprintln(w, "property float nx")
	fmt.Fprintln(w, "property float ny")
	fmt.Fprintln(w, "property float nz")
	fmt.Fprintln(w, "property uchar red")
	fmt.Fprintln(w, "property uchar green")
	fmt.Fprintln(w, "property uchar blue")
	fmt.Fprintln(w, "property float error_m")
	fmt.Fprintf(w, "element face %d\n", len(mesh.Triangles))
	fmt.Fprintln(w, "property list uchar int vertex_indices")
	fmt.Fprintln(w, "end_header")
	for _, v := range mesh.Vertices {
		col := HeatColor(v.Error / 0.03).Clamp()
		fmt.Fprintf(w, "%.6f %.6f %.6f %.5f %.5f %.5f %d %d %d %.6f\n",
			v.Pos.X, v.Pos.Y, v.Pos.Z, v.Normal.X, v.Normal.Y, v.Normal.Z,
			int(col.R*255+0.5), int(col.G*255+0.5), int(col.B*255+0.5), v.Error)
	}
	for _, t := range mesh.Triangles {
		fmt.Fprintf(w, "3 %d %d %d\n", t[0], t[1], t[2])
	}
	return w.Flush()
}

func WritePointsPLY(path string, points []Vec3, colors []RGB) error {
	if err := ensureDir(path); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := bufio.NewWriterSize(f, 1<<20)
	fmt.Fprintln(w, "ply")
	fmt.Fprintln(w, "format ascii 1.0")
	fmt.Fprintf(w, "element vertex %d\n", len(points))
	fmt.Fprintln(w, "property float x")
	fmt.Fprintln(w, "property float y")
	fmt.Fprintln(w, "property float z")
	if colors != nil {
		fmt.Fprintln(w, "property uchar red")
		fmt.Fprintln(w, "property uchar green")
		fmt.Fprintln(w, "property uchar blue")
	}
	fmt.Fprintln(w, "end_header")
	for i, p := range points {
		if colors != nil && i < len(colors) {
			c := colors[i].Clamp()
			fmt.Fprintf(w, "%.6f %.6f %.6f %d %d %d\n", p.X, p.Y, p.Z,
				int(c.R*255+0.5), int(c.G*255+0.5), int(c.B*255+0.5))
		} else {
			fmt.Fprintf(w, "%.6f %.6f %.6f\n", p.X, p.Y, p.Z)
		}
	}
	return w.Flush()
}

func WriteThresholdPLY(path string, cells []ThresholdCell) error {
	points := make([]Vec3, len(cells))
	colors := make([]RGB, len(cells))
	for i, c := range cells {
		points[i] = c.Pos
		colors[i] = Viridis(0.25 + 0.6*float64(c.Weight)/10)
	}
	return WritePointsPLY(path, points, colors)
}
