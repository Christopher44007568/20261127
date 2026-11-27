package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for PNG renders")
	texSize := flag.Int("tex-size", 1024, "power-of-two procedural roughness texture size")
	lutSamples := flag.Int("lut-samples", 96, "deterministic energy LUT samples")
	flag.Parse()

	started := time.Now()
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		exitf("create output directory: %v", err)
	}

	texture := BuildRoughnessTexture(*texSize)
	lut := BuildEnergyLUT(24, 17, *lutSamples)
	scene := &Scene{
		Texture: texture,
		LUT:     lut,
		Lights:  DefaultLights(),
		F0:      V(0.74, 0.80, 0.88),
	}

	views := []ComparisonView{NearView(), FarView(), PreviewView()}
	for _, view := range views {
		frame := RenderComparison(view, scene)
		img := frame.ToRGBA(1.0)
		path := filepath.Join(*outDir, view.Name+".png")
		if err := WritePNG(path, img); err != nil {
			exitf("write %s: %v", path, err)
		}
		fmt.Printf("wrote %s (%dx%d, r=%.0f)\n", path, view.Width, view.Height, view.Radius)
	}

	sweep := RenderSweep(scene)
	sweepPath := filepath.Join(*outDir, "continuity.png")
	if err := WritePNG(sweepPath, sweep.ToRGBA(1.0)); err != nil {
		exitf("write %s: %v", sweepPath, err)
	}
	fmt.Printf("wrote %s\n", sweepPath)

	report := BuildReport(scene)
	reportPath := filepath.Join(*outDir, "roughfoot-report.json")
	if err := WriteReport(reportPath, report); err != nil {
		exitf("write %s: %v", reportPath, err)
	}
	fmt.Printf("wrote %s (max continuity jump %.3f)\n", reportPath, report.MaxLuminanceJump)
	fmt.Printf("done in %s\n", time.Since(started).Round(time.Millisecond))
}

func exitf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "roughfoot: "+format+"\n", args...)
	os.Exit(1)
}
