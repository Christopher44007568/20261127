package main

import (
	"flag"
	"fmt"
	"image"
	"log"
	"math"
	"path/filepath"
	"time"
)

func main() {
	outDir := flag.String("out", "artifacts", "output directory")
	quick := flag.Bool("quick", false, "lower resolution preview render")
	flag.Parse()

	if err := Generate(*outDir, *quick); err != nil {
		log.Fatal(err)
	}
}

func Generate(outDir string, quick bool) error {
	scene := NewGalleryScene()
	width, height := 960, 540
	samples, fogSteps := 4, 32
	if quick {
		width, height = 480, 270
		samples, fogSteps = 1, 16
	}
	camera := heroCamera(width, height)
	grid := BuildFroxelGrid(scene, camera, DefaultGridConfig())
	base := DefaultRenderConfig()
	base.Width, base.Height = width, height
	base.Samples = samples
	base.FogSteps = fogSteps

	fmt.Printf("reference: %dx%d, %d samples, %d froxels\n", width, height, samples, len(grid.Cells))
	refCfg := base
	refCfg.Mode = ReferenceMode
	ref := RenderScene(scene, camera, grid, refCfg)
	if err := SavePNG(filepath.Join(outDir, "reference.png"), ref.Image); err != nil {
		return err
	}

	fmt.Println("froxel pass")
	frxCfg := base
	frxCfg.Mode = FroxelMode
	frx := RenderScene(scene, camera, grid, frxCfg)
	if err := SavePNG(filepath.Join(outDir, "froxel.png"), frx.Image); err != nil {
		return err
	}

	cmp := CompareImages(ref.Image, frx.Image)
	fmt.Printf("parity: exact=%v different=%d max=%d psnr=%s\n",
		cmp.Exact, cmp.DifferentPixels, cmp.MaxChannelDiff, formatFloat(cmp.PSNR, 2))
	diff := DiffImage(ref.Image, frx.Image, 12)
	if err := SavePNG(filepath.Join(outDir, "diff.png"), diff); err != nil {
		return err
	}
	heatmap := BuildHeatmap(grid, 16)
	if err := SavePNG(filepath.Join(outDir, "heatmap.png"), heatmap); err != nil {
		return err
	}

	wide, err := generateWideFOV(outDir, scene, quick)
	if err != nil {
		return err
	}
	frames, gifImage, err := generateFrames(outDir, scene, quick)
	if err != nil {
		return err
	}

	manifest := ArtifactManifest{
		Title:        "Conservative Froxel Forward Lighting",
		GeneratedUTC: time.Now().UTC().Format(time.RFC3339),
		Renderer:     "pure Go CPU ray tracer; standard library only",
		Scene: map[string]any{
			"point_and_spot_lights": len(scene.Lights),
			"boxes":                 len(scene.Boxes),
			"spheres":               len(scene.Spheres),
			"transparent_volume":    scene.Volume.Bounds,
			"volume_density":        scene.Volume.Density,
			"volume_anisotropy":     scene.Volume.Anisotropy,
		},
		Camera:          manifestCamera(camera),
		Froxel:          grid.Stats,
		Reference:       manifestImage(filepath.ToSlash("reference.png"), ref.Image),
		FroxelImage:     manifestImage(filepath.ToSlash("froxel.png"), frx.Image),
		Heatmap:         manifestImage(filepath.ToSlash("heatmap.png"), heatmap),
		Diff:            manifestImage(filepath.ToSlash("diff.png"), diff),
		Comparison:      manifestComparison(cmp),
		ReferenceRender: ref.Stats,
		FroxelRender:    frx.Stats,
		WideFOV:         wide,
		Frames:          frames,
		GIF:             gifImage,
		FallbackPolicies: []string{
			"A light spanning depth slices is inserted into every cell whose view-space AABB intersects its finite influence sphere.",
			"Vertical FOV above 120 degrees or horizontal FOV above 150 degrees marks every cell unsafe and queries the all-light list.",
			"Transparent-volume samples resolve their own (pixel, viewZ) cell; no opaque surface list or adjacent-cell list is reused.",
			"Volume samples outside the froxel depth range fall back to all lights, and fragments never copy another cell's shaded result.",
		},
	}
	return WriteManifest(filepath.Join(outDir, "manifest.json"), manifest)
}

func heroCamera(width, height int) *Camera {
	return NewCamera(
		V(0, 3.0, -4.8),
		V(0, 2.15, 9.0),
		V(0, 1, 0),
		67,
		float64(width)/float64(height),
		0.16,
		30,
	)
}

func formatFloat(x float64, precision int) string {
	return fmt.Sprintf("%.*f", precision, x)
}

func generateWideFOV(outDir string, scene *Scene, quick bool) (*WideFOVArtifact, error) {
	width, height := 720, 405
	samples, fogSteps := 1, 20
	if quick {
		width, height = 360, 203
		fogSteps = 12
	}
	camera := NewCamera(
		V(0, 2.9, -1.0),
		V(0, 2.15, 9.0),
		V(0, 1, 0),
		135,
		float64(width)/float64(height),
		0.16,
		30,
	)
	grid := BuildFroxelGrid(scene, camera, DefaultGridConfig())
	cfg := DefaultRenderConfig()
	cfg.Width, cfg.Height = width, height
	cfg.Samples, cfg.FogSteps = samples, fogSteps
	fmt.Printf("wide fov: hfov %.1f deg, fallback cells %d/%d\n",
		grid.Stats.HorizontalFOV, grid.Stats.FallbackCells, grid.Stats.Cells)

	cfg.Mode = ReferenceMode
	ref := RenderScene(scene, camera, grid, cfg)
	if err := SavePNG(filepath.Join(outDir, "wide_fov_reference.png"), ref.Image); err != nil {
		return nil, err
	}
	cfg.Mode = FroxelMode
	frx := RenderScene(scene, camera, grid, cfg)
	if err := SavePNG(filepath.Join(outDir, "wide_fov_froxel.png"), frx.Image); err != nil {
		return nil, err
	}
	cmp := CompareImages(ref.Image, frx.Image)
	fmt.Printf("wide fov parity: exact=%v different=%d\n", cmp.Exact, cmp.DifferentPixels)
	return &WideFOVArtifact{
		Camera:          manifestCamera(camera),
		Froxel:          grid.Stats,
		ReferenceImage:  manifestImage(filepath.ToSlash("wide_fov_reference.png"), ref.Image),
		FroxelImage:     manifestImage(filepath.ToSlash("wide_fov_froxel.png"), frx.Image),
		ReferenceRender: ref.Stats,
		FroxelRender:    frx.Stats,
		Comparison:      manifestComparison(cmp),
	}, nil
}

func generateFrames(outDir string, scene *Scene, quick bool) ([]ManifestFrame, ManifestImage, error) {
	frameCount := 12
	width, height := 640, 360
	samples, fogSteps := 2, 20
	if quick {
		frameCount = 6
		width, height = 320, 180
		samples, fogSteps = 1, 12
	}
	var gifFrames []*image.RGBA
	frames := make([]ManifestFrame, 0, frameCount)
	for i := 0; i < frameCount; i++ {
		t := float64(i) / float64(frameCount-1)
		camera := movingCamera(width, height, t)
		grid := BuildFroxelGrid(scene, camera, DefaultGridConfig())
		cfg := DefaultRenderConfig()
		cfg.Width, cfg.Height = width, height
		cfg.Samples, cfg.FogSteps = samples, fogSteps
		cfg.Mode = ReferenceMode
		ref := RenderScene(scene, camera, grid, cfg)
		cfg.Mode = FroxelMode
		frx := RenderScene(scene, camera, grid, cfg)
		cmp := CompareImages(ref.Image, frx.Image)
		path := filepath.ToSlash(filepath.Join("frames", fmt.Sprintf("frame_%03d.png", i)))
		if err := SavePNG(filepath.Join(outDir, filepath.FromSlash(path)), frx.Image); err != nil {
			return nil, ManifestImage{}, err
		}
		gifFrames = append(gifFrames, frx.Image)
		frames = append(frames, ManifestFrame{
			Index:      i,
			Path:       path,
			Camera:     manifestCamera(camera),
			Froxel:     grid.Stats,
			Render:     frx.Stats,
			Comparison: manifestComparison(cmp),
		})
		fmt.Printf("frame %02d parity: exact=%v different=%d pos=(%.2f,%.2f,%.2f)\n",
			i, cmp.Exact, cmp.DifferentPixels, camera.Position.X, camera.Position.Y, camera.Position.Z)
	}
	gifPath := filepath.Join(outDir, "camera_orbit.gif")
	if err := SaveGIF(gifPath, gifFrames, 8); err != nil {
		return nil, ManifestImage{}, err
	}
	gifImage := manifestImage(filepath.ToSlash("camera_orbit.gif"), gifFrames[0])
	return frames, gifImage, nil
}

func movingCamera(width, height int, t float64) *Camera {
	angle := -0.48 + 0.96*t
	target := V(0.25*math.Sin(angle*1.7), 2.18, 9.1)
	position := V(
		math.Sin(angle)*8.5,
		3.02+0.28*math.Sin(angle*2.4),
		9.0-math.Cos(angle)*13.4,
	)
	fov := 66 + 3*math.Sin(t*math.Pi)
	return NewCamera(position, target, V(0, 1, 0), fov, float64(width)/float64(height), 0.16, 30)
}
