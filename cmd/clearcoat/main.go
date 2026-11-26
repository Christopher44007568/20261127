package main

import (
	"flag"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"time"

	"clearcoat/render"
)

func main() {
	outDir := flag.String("out", "out", "directory for rendered PNG files")
	flag.Parse()

	start := time.Now()
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}

	camera := render.NewCamera(
		render.V(0, 1.52, 4.90),
		render.V(0, 1.02, 0),
		34,
	)
	base := render.Material{
		Albedo:           render.V(0.58, 0.155, 0.042),
		CoatingRoughness: 0.075,
		CoatingF0:        render.DefaultCoatingF0,
		EnergyCap:        render.DefaultEnergyCap,
	}

	dry := sceneWith(base, 0, false)
	wet := sceneWith(base, 1, false)
	dryImage := render.Render(dry, camera, render.DefaultOptions(760, 760, render.RenderComposite))
	wetImage := render.Render(wet, camera, render.DefaultOptions(760, 760, render.RenderComposite))

	dryWet := render.ComposePanels([]render.Panel{
		{Image: dryImage, Label: "DRY / COAT 0.00"},
		{Image: wetImage, Label: "WET / COAT 1.00"},
	}, 8)
	mustSave(filepath.Join(*outDir, "dry_wet.png"), dryWet)

	splitOpts := render.DefaultOptions(590, 720, render.RenderDiffuse)
	splitOpts.Isolated = true
	diffuseImage := render.Render(wet, camera, splitOpts)
	splitOpts.Mode = render.RenderClearcoat
	coatImage := render.Render(wet, camera, splitOpts)
	compositeImage := render.Render(wet, camera, render.DefaultOptions(590, 720, render.RenderComposite))

	separation := render.ComposePanels([]render.Panel{
		{Image: diffuseImage, Label: "BASE DIFFUSE"},
		{Image: compositeImage, Label: "WET COMPOSITE"},
		{Image: coatImage, Label: "CLEARCOAT REFLECTION"},
	}, 8)
	mustSave(filepath.Join(*outDir, "reflection_separation.png"), separation)

	gradient := base
	gradient.ClearcoatWeight = 1
	gradientScene := sceneWith(gradient, 1, true)
	gradientImage := render.Render(gradientScene, camera, render.DefaultOptions(960, 720, render.RenderComposite))
	roughnessSheet := render.ComposePanels([]render.Panel{
		{Image: gradientImage, Label: "CLEARCOAT ROUGHNESS 0.02 TO 0.92"},
	}, 0)
	mustSave(filepath.Join(*outDir, "roughness_gradient.png"), roughnessSheet)

	fmt.Printf("rendered 3 sheets in %s\n", time.Since(start).Round(time.Millisecond))
}

func sceneWith(material render.Material, weight float64, gradient bool) render.Scene {
	material.ClearcoatWeight = weight
	sphere := render.Sphere{
		Center:            render.V(0, 1.06, 0),
		Radius:            1.05,
		Material:          material,
		RoughnessGradient: gradient,
		GradientMin:       0.02,
		GradientMax:       0.92,
	}
	return render.Scene{Sphere: sphere, FloorY: 0}
}

func mustSave(path string, img image.Image) {
	if err := render.SavePNG(path, img); err != nil {
		fatal(err)
	}
	fmt.Println("wrote", path)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "clearcoat:", err)
	os.Exit(1)
}
