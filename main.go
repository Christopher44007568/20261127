// Command dirtycanvas demonstrates pixel-perfect dirty-rectangle repainting
// while dragging objects on a large canvas. It writes a dirty-region map,
// the incremental result and the full reference render as PPM files, and
// verifies every simulated frame against a full repaint.
//
// Only the Go standard library is used.
package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	cfg := DefaultConfig()
	flag.IntVar(&cfg.Width, "width", cfg.Width, "canvas width in pixels")
	flag.IntVar(&cfg.Height, "height", cfg.Height, "canvas height in pixels")
	flag.IntVar(&cfg.Updates, "frames", cfg.Updates, "number of drag frames to simulate")
	flag.StringVar(&cfg.OutDir, "out", cfg.OutDir, "directory for PPM/PNG/report artifacts")
	flag.Parse()

	rep, err := Run(cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "dirtycanvas: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("canvas %dx%d, %d drag frames\n", rep.Width, rep.Height, rep.Updates)
	fmt.Printf("average dirty area: %.2f%% of canvas (last frame %.2f%%)\n", rep.AvgDirtyRatioPct, rep.LastDirtyRatioPct)
	fmt.Printf("PIXEL_PERFECT=%t total_mismatch_pixels=%d\n", rep.PixelPerfect, rep.TotalMismatchPixels)
	fmt.Printf("artifacts written to %s/\n", cfg.OutDir)
	for _, f := range rep.Files {
		fmt.Println("  " + f)
	}
	if !rep.PixelPerfect {
		os.Exit(2)
	}
}
