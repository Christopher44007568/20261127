package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"

	skinsss "skinsss"
)

func main() {
	outDir := flag.String("out", "artifacts", "output directory for PNG files")
	size := flag.Int("size", 512, "render width/height in pixels")
	samples := flag.Int("samples", 4, "supersamples per pixel (1 or 4)")
	flag.Parse()

	scene := skinsss.DefaultScene()

	sssCfg := skinsss.RenderConfig{
		Width: *size, Height: *size, Samples: *samples,
		SSSEnabled: true,
		CamPos:     skinsss.Vec3{X: 0, Y: 0.12, Z: 3.4},
		CamTarget:  skinsss.Vec3{},
		Focal:      1.1,
	}
	diffCfg := sssCfg
	diffCfg.SSSEnabled = false

	modes := []struct {
		mode skinsss.Mode
		name string
	}{
		{skinsss.ModeFront, "front"},
		{skinsss.ModeSide, "side"},
		{skinsss.ModeBack, "back"},
		{skinsss.ModeHeatmap, "heatmap"},
	}

	for _, m := range modes {
		img := skinsss.Render(scene, m.mode, sssCfg)
		path := filepath.Join(*outDir, "sss_"+m.name+".png")
		if err := skinsss.WritePNG(path, img); err != nil {
			fatal(err)
		}
		fmt.Println("wrote", path)

		if m.mode != skinsss.ModeHeatmap {
			diff := skinsss.Render(scene, m.mode, diffCfg)
			diffPath := filepath.Join(*outDir, "diffuse_"+m.name+".png")
			if err := skinsss.WritePNG(diffPath, diff); err != nil {
				fatal(err)
			}
			fmt.Println("wrote", diffPath)
		}
	}

	// Contact sheet: front / side / back / heatmap in a 2x2 grid.
	tileCfg := sssCfg
	tileCfg.Width, tileCfg.Height = 256, 256
	tileCfg.Samples = 1
	tiles := make([]*image.RGBA, 0, 4)
	for _, m := range modes {
		tiles = append(tiles, skinsss.Render(scene, m.mode, tileCfg))
	}
	sheet := skinsss.ComposeGrid(tiles, 2, 2, 8)
	sheetPath := filepath.Join(*outDir, "contact_sheet.png")
	if err := skinsss.WritePNG(sheetPath, sheet); err != nil {
		fatal(err)
	}
	fmt.Println("wrote", sheetPath)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "skinsss:", err)
	os.Exit(1)
}
