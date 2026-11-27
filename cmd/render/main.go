package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"time"

	fp "flakepaint"
)

func main() {
	width := flag.Int("w", 1200, "output width")
	height := flag.Int("h", 760, "output height")
	samples := flag.Int("samples", 4, "samples per pixel (1, 2, 4, 8)")
	seed := flag.Uint64("seed", 0x20261127, "deterministic flake seed")
	outDir := flag.String("out", "out", "output directory")
	flag.Parse()

	if *width < 64 || *height < 64 {
		fmt.Fprintln(os.Stderr, "width and height must be at least 64")
		os.Exit(2)
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "create output directory:", err)
		os.Exit(1)
	}

	start := time.Now()
	nearBuf := fp.RenderNearPanel(*width, *height, *samples, *seed)
	near := nearBuf.ToImage(0.92, 0.075)
	fp.DecorateNear(near, "NEAR PANEL / 0.62 M")
	mustWritePNG(filepath.Join(*outDir, "near_panel.png"), near)

	farBuf := fp.RenderFarSphere(*width, *height, *samples, *seed)
	far := farBuf.ToImage(0.95, 0.12)
	fp.DecorateFar(far, "FAR SPHERE / 20 M")
	mustWritePNG(filepath.Join(*outDir, "far_sphere.png"), far)

	density := fp.RenderDensityMap(*width, *height, fp.DefaultFlakeField(*seed))
	mustWritePNG(filepath.Join(*outDir, "flake_density.png"), density)

	showcase := fp.ComposeShowcase(near, far, density)
	mustWritePNG(filepath.Join(*outDir, "showcase.png"), showcase)

	fmt.Printf("rendered 4 images in %s\n", time.Since(start).Round(time.Millisecond))
}

func mustWritePNG(path string, img image.Image) {
	f, err := os.Create(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, "create", path+":", err)
		os.Exit(1)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		fmt.Fprintln(os.Stderr, "encode", path+":", err)
		os.Exit(1)
	}
	fmt.Println(path)
}
