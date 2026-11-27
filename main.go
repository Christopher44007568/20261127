package main

import (
	"flag"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

func defaultCamera() Camera {
	return Camera{
		Eye:    Vec3{0.12, 2.48, 3.20},
		Target: Vec3{0.02, -0.16, 0.03},
		Up:     Vec3{0, 1, 0},
		FovY:   37 * 3.141592653589793 / 180,
	}
}

func savePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(f, img); err != nil {
		return err
	}
	fmt.Printf("wrote %s\n", path)
	return nil
}

func main() {
	outDir := flag.String("out", "out", "output directory")
	width := flag.Int("width", 1600, "render width")
	height := flag.Int("height", 1000, "render height")
	flag.Parse()

	cam := defaultCamera()
	g := renderGBuffer(cam, *width, *height, 820, 760)
	heightBlend := shadeImage(cam, g, ShadeHeight)
	naive := shadeImage(cam, g, ShadeNaive)
	mud := shadeImage(cam, g, ShadeMaterialMud)
	moss := shadeImage(cam, g, ShadeMaterialMoss)
	stone := shadeImage(cam, g, ShadeMaterialStone)
	wetReflection := shadeImage(cam, g, ShadeReflectionWet)
	reflectionDiff := shadeImage(cam, g, ShadeReflectionDiff)

	outputs := []struct {
		name string
		img  image.Image
	}{
		{"height_blend.png", composeHero(heightBlend)},
		{"material_decomposition.png", composeDecomposition(mud, moss, stone, g)},
		{"boundary_zoom.png", composeBoundary(naive, heightBlend)},
		{"reflection_difference.png", composeReflection(wetReflection, reflectionDiff)},
	}
	for _, item := range outputs {
		if err := savePNG(filepath.Join(*outDir, item.name), item.img); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}
