package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

const (
	textureSize = 512
	tileSpan    = float64(textureSize)
	gridSize    = 1024
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for generated PNG files")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	texture := MakeTexture(textureSize)
	if err := WritePNG(filepath.Join(*outDir, "texture_tile.png"), texture); err != nil {
		fatal(err)
	}

	// The source corner maps to (22,-18), so the four tile corners meet there.
	// Rotation makes any bad filtering at the four-way seam immediately visible.
	sceneTransform := Translate(22, -18).
		Mul(Rotate(degrees(-14))).
		Mul(Scale(1, 1))

	gridView := View{
		MinX: -gridSize / 2, MinY: -gridSize / 2,
		Width: gridSize, Height: gridSize,
	}
	grid := RenderView(texture, sceneTransform, gridView, gridSize, gridSize, 2)
	if err := WritePNG(filepath.Join(*outDir, "texture_four_grid.png"), grid); err != nil {
		fatal(err)
	}

	// Re-render the exact same scene at 4x around the four-corner seam. This
	// preserves the mapping instead of merely upscaling a crop and hiding errors.
	seamX, seamY := sceneTransform.Apply(0, 0)
	const zoom = 4
	const zoomSpan = 256.0
	edgeView := View{
		MinX: seamX - zoomSpan/2, MinY: seamY - zoomSpan/2,
		Width: zoomSpan, Height: zoomSpan,
	}
	edge := RenderView(
		texture, sceneTransform, edgeView,
		int(zoomSpan)*zoom, int(zoomSpan)*zoom, 3,
	)
	if err := WritePNG(filepath.Join(*outDir, "texture_edge_zoom.png"), edge); err != nil {
		fatal(err)
	}

	fmt.Printf("wrote 3 PNGs to %s\n", *outDir)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
