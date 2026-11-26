package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	stroke "strokegeom"
)

func main() {
	mesh := demoMesh()
	bounds := mesh.Bounds()
	const width, height = 1200, 800
	transform := stroke.FitTransform(bounds, width, height, 64)
	opts := stroke.RasterOptions{
		Width:      width,
		Height:     height,
		Transform:  transform,
		Samples:    4,
		Background: color.NRGBA{R: 250, G: 248, B: 243, A: 255},
		Foreground: color.NRGBA{R: 23, G: 42, B: 58, A: 255},
		Overlap:    color.NRGBA{R: 220, G: 38, B: 38, A: 255},
	}
	diag := stroke.RenderDiagnostics(mesh, opts)

	outDir := filepath.Join("artifacts")
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatal(err)
	}
	writePNG(filepath.Join(outDir, "stroke-fill.png"), diag.Fill)
	writePNG(filepath.Join(outDir, "stroke-aa.png"), diag.AntiAliased)
	writePNG(filepath.Join(outDir, "stroke-overlap.png"), diag.Overlaps)

	fmt.Printf("triangles: %d\n", len(mesh.Triangles))
	fmt.Printf("overlap pairs: %d\n", len(diag.OverlapPairs))
	fmt.Println("wrote artifacts/stroke-fill.png")
	fmt.Println("wrote artifacts/stroke-aa.png")
	fmt.Println("wrote artifacts/stroke-overlap.png")
}

func demoMesh() stroke.Mesh {
	var mesh stroke.Mesh

	sharp := stroke.NewPath()
	sharp.MoveTo(stroke.Point{X: 42, Y: 62})
	sharp.LineTo(stroke.Point{X: 112, Y: 208})
	sharp.LineTo(stroke.Point{X: 188, Y: 54})
	sharp.LineTo(stroke.Point{X: 264, Y: 208})
	sharp.LineTo(stroke.Point{X: 336, Y: 54})
	sharp.LineTo(stroke.Point{X: 398, Y: 176})
	appendMesh(&mesh, stroke.Stroke(*sharp, stroke.Style{
		Width:      15,
		Join:       stroke.JoinMiter,
		Cap:        stroke.CapRound,
		MiterLimit: 4,
		Dashes:     []float64{48, 32},
		DashOffset: 6,
	}))

	wave := stroke.NewPath()
	wave.MoveTo(stroke.Point{X: 44, Y: 286})
	for i := 0; i < 8; i++ {
		x := 44.0 + float64(i)*62
		y := 218.0
		if i%2 == 1 {
			y = 286
		}
		wave.LineTo(stroke.Point{X: x, Y: y})
	}
	appendMesh(&mesh, stroke.Stroke(*wave, stroke.Style{
		Width:      12,
		Join:       stroke.JoinRound,
		Cap:        stroke.CapRound,
		Dashes:     []float64{18, 20},
		DashOffset: 6,
	}))

	dots := stroke.NewPath()
	for _, p := range []stroke.Point{
		{X: 64, Y: 338},
		{X: 112, Y: 338},
		{X: 160, Y: 338},
	} {
		dots.MoveTo(p)
	}
	appendMesh(&mesh, stroke.Stroke(*dots, stroke.Style{
		Width: 11, Cap: stroke.CapRound, Join: stroke.JoinRound,
	}))

	diamond := stroke.NewPath()
	diamond.MoveTo(stroke.Point{X: 0, Y: -34})
	diamond.LineTo(stroke.Point{X: 28, Y: 0})
	diamond.LineTo(stroke.Point{X: 0, Y: 34})
	diamond.LineTo(stroke.Point{X: -28, Y: 0})
	diamond.Close()
	scaled := stroke.Affine{A: 1.45, D: 0.55, E: 438, F: 300}
	appendMesh(&mesh, stroke.StrokeTransformed(*diamond, stroke.Style{
		Width: 8, Join: stroke.JoinRound, Cap: stroke.CapRound,
	}, scaled, stroke.WidthFollowTransform))

	fault := stroke.NewPath()
	fault.MoveTo(stroke.Point{X: 214, Y: 326})
	fault.LineTo(stroke.Point{X: 282, Y: 288})
	fault.LineTo(stroke.Point{X: 238, Y: 360})
	fault.LineTo(stroke.Point{X: 302, Y: 324})
	appendMesh(&mesh, stroke.Stroke(*fault, stroke.Style{
		Width: 10, Join: stroke.JoinBevel, Cap: stroke.CapRound,
	}))

	return mesh
}

func appendMesh(dst *stroke.Mesh, src stroke.Mesh) {
	dst.Triangles = append(dst.Triangles, src.Triangles...)
}

func writePNG(path string, img image.Image) {
	if img == nil {
		fatal(fmt.Errorf("nil image for %s", path))
	}
	f, err := os.Create(path)
	if err != nil {
		fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fatal(err)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
