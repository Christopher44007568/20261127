package main

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"
)

const (
	imageWidth  = 1200
	imageHeight = 800
)

func main() {
	started := time.Now()
	tex := buildStoneTexture(1200, 720)
	outDir := "artifacts"
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		exitf("create output directory: %v", err)
	}

	scenes := []Scene{
		{Name: "front_view", Kind: ViewFront, Budget: 48, MaxDistance: 0.32},
		{Name: "grazing_view", Kind: ViewGrazing, Budget: 64, MaxDistance: 0.85},
		{Name: "error_fallback", Kind: ViewFallback, Budget: 7, MaxDistance: 0.85},
	}
	for _, scene := range scenes {
		img, stats := renderScene(tex, scene)
		path := filepath.Join(outDir, scene.Name+".png")
		if err := writePNG(path, img); err != nil {
			exitf("write %s: %v", path, err)
		}
		fmt.Printf("%-15s %s  layers<=%d  reached=%.1f%%  boundary=%.2f%%  budget=%.2f%%\n",
			scene.Name, path, scene.Budget,
			100*float64(stats.Hit)/float64(stats.Pixels),
			100*float64(stats.BoundaryFallback)/float64(stats.Pixels),
			100*float64(stats.BudgetFallback)/float64(stats.Pixels))
	}
	fmt.Printf("rendered in %s\n", time.Since(started).Round(time.Millisecond))
}

func writePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	return encoder.Encode(file, img)
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "stonewall: "+format+"\n", args...)
	os.Exit(1)
}
