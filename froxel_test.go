package main

import (
	"image"
	"math"
	"testing"
)

func testCamera(width, height int) *Camera {
	return NewCamera(V(0, 3, -4.8), V(0, 2.15, 9), V(0, 1, 0), 67,
		float64(width)/float64(height), 0.16, 30)
}

func TestReferenceAndFroxelMatchIncludingVolume(t *testing.T) {
	scene := NewGalleryScene()
	camera := testCamera(192, 108)
	grid := BuildFroxelGrid(scene, camera, DefaultGridConfig())
	cfg := DefaultRenderConfig()
	cfg.Width, cfg.Height, cfg.Samples, cfg.FogSteps = 192, 108, 1, 10
	cfg.Mode = ReferenceMode
	ref := RenderScene(scene, camera, grid, cfg)
	cfg.Mode = FroxelMode
	frx := RenderScene(scene, camera, grid, cfg)
	cmp := CompareImages(ref.Image, frx.Image)
	if !cmp.Exact {
		t.Fatalf("froxel pass differs from all-light reference: %+v", cmp)
	}
	if frx.Stats.AverageLights <= 0 || frx.Stats.AverageLights > float64(len(scene.Lights)) {
		t.Fatalf("invalid average light count: %v", frx.Stats.AverageLights)
	}
	if frx.Stats.AverageLights >= ref.Stats.AverageLights {
		t.Fatalf("froxel pass did not reduce light visits: froxel=%v reference=%v",
			frx.Stats.AverageLights, ref.Stats.AverageLights)
	}
}

func TestLightSpanningDepthSlicesAppearsInEveryIntersectedCell(t *testing.T) {
	scene := NewGalleryScene()
	camera := testCamera(320, 180)
	grid := BuildFroxelGrid(scene, camera, DefaultGridConfig())
	lightIndex := 2
	foundSlices := 0
	for tz := 0; tz < grid.Config.Slices; tz++ {
		cell, ok := grid.InspectCell(grid.Config.TilesX/2, grid.Config.TilesY/2, tz)
		if !ok {
			t.Fatal("expected valid central cell")
		}
		if containsInt(cell.Lights, lightIndex) {
			foundSlices++
		}
	}
	if foundSlices < 2 {
		t.Fatalf("light crossing depth slices was only present in %d cell(s)", foundSlices)
	}
}

func TestNoAdjacentCellLightingCopy(t *testing.T) {
	scene := NewGalleryScene()
	camera := testCamera(320, 180)
	grid := BuildFroxelGrid(scene, camera, DefaultGridConfig())
	tx, ty := grid.Config.TilesX/2, grid.Config.TilesY/2
	differentPair := false
	for tz := 0; tz+1 < grid.Config.Slices; tz++ {
		a, _ := grid.InspectCell(tx, ty, tz)
		b, _ := grid.InspectCell(tx, ty, tz+1)
		if !sameInts(a.Lights, b.Lights) {
			differentPair = true
		}
		z0, z1 := grid.depthSliceBounds(tz)
		zMid0 := math.Sqrt(z0 * z1)
		z2, z3 := grid.depthSliceBounds(tz + 1)
		zMid1 := math.Sqrt(z2 * z3)
		q0, cell0, _ := grid.Query(160.5, 90.5, 320, 180, zMid0)
		q1, cell1, _ := grid.Query(160.5, 90.5, 320, 180, zMid1)
		if !sameInts(q0, a.Lights) || !sameInts(q1, b.Lights) {
			t.Fatalf("query returned another cell's list at slice %d", tz)
		}
		if cell0 == cell1 {
			t.Fatalf("adjacent depth queries resolved to the same cell %d", cell0)
		}
	}
	if !differentPair {
		t.Fatal("all central depth slices have identical light lists; suspicious neighbor copy")
	}
}

func TestConservativeCoverageAtArbitrarySurfaces(t *testing.T) {
	scene := NewGalleryScene()
	camera := testCamera(320, 180)
	grid := BuildFroxelGrid(scene, camera, DefaultGridConfig())
	tanHalf := math.Tan(camera.FovY * 0.5)
	tested := 0
	for x := -4.0; x <= 4.0; x += 1.0 {
		for z := 3.0; z <= 16.0; z += 1.0 {
			for _, y := range []float64{0.02, 1.0, 3.0} {
				p := V(x, y, z)
				view := camera.View(p)
				if view.Z < camera.Near || view.Z > camera.Far {
					continue
				}
				px := (view.X/(view.Z*tanHalf*camera.Aspect) + 1) * 0.5 * 320
				py := (1 - view.Y/(view.Z*tanHalf)) * 0.5 * 180
				lights, _, fallback := grid.Query(px, py, 320, 180, view.Z)
				if fallback != NoFallback {
					continue
				}
				for i, light := range scene.Lights {
					if light.Attenuation(p) > 0 && !containsInt(lights, i) {
						t.Fatalf("visible light %d missing at point %+v (pixel %.2f,%.2f)", i, p, px, py)
					}
				}
				tested++
			}
		}
	}
	if tested < 100 {
		t.Fatalf("coverage test only reached %d points", tested)
	}
}

func TestWideFOVFallsBackAndStillMatches(t *testing.T) {
	scene := NewGalleryScene()
	camera := testCamera(160, 90)
	camera.FovY = 150 * math.Pi / 180
	grid := BuildFroxelGrid(scene, camera, DefaultGridConfig())
	if !grid.Stats.WideFOV || grid.Stats.FallbackCells != grid.Stats.Cells {
		t.Fatalf("wide FOV did not mark all cells unsafe: %+v", grid.Stats)
	}
	lights, _, reason := grid.Query(80, 45, 160, 90, 5)
	if reason != WideFOVFallback || len(lights) != len(scene.Lights) {
		t.Fatalf("wide FOV query did not return all lights: reason=%v lights=%d", reason, len(lights))
	}
	cfg := DefaultRenderConfig()
	cfg.Width, cfg.Height, cfg.Samples, cfg.FogSteps = 160, 90, 1, 4
	cfg.Mode = ReferenceMode
	ref := RenderScene(scene, camera, grid, cfg)
	cfg.Mode = FroxelMode
	frx := RenderScene(scene, camera, grid, cfg)
	if cmp := CompareImages(ref.Image, frx.Image); !cmp.Exact {
		t.Fatalf("wide FOV fallback differs from reference: %+v", cmp)
	}
	if frx.Stats.VolumeFallback == 0 {
		t.Fatal("wide FOV transparent volume did not exercise full-light fallback")
	}
}

func TestTransparentVolumeUsesOwnDepthCellAndFallsBackOutsideGrid(t *testing.T) {
	scene := NewGalleryScene()
	camera := testCamera(320, 180)
	grid := BuildFroxelGrid(scene, camera, DefaultGridConfig())
	lightsA, cellA, reasonA := grid.Query(160.5, 90.5, 320, 180, 4.5)
	lightsB, cellB, reasonB := grid.Query(160.5, 90.5, 320, 180, 12.0)
	if reasonA != NoFallback || reasonB != NoFallback || cellA == cellB {
		t.Fatalf("volume samples shared a depth cell: %d/%d reasons %v/%v",
			cellA, cellB, reasonA, reasonB)
	}
	if sameInts(lightsA, lightsB) {
		t.Log("nearby volume cells happen to have equal light lists")
	}
	all, cell, reason := grid.Query(160.5, 90.5, 320, 180, camera.Far+1)
	if reason != DepthOutsideFallback || cell != -1 || len(all) != len(scene.Lights) {
		t.Fatalf("outside-depth transparent sample did not fall back: reason=%v cell=%d", reason, cell)
	}
	info := pixelInfo{X: 160.5, Y: 90.5, Width: 320, Height: 180, Grid: grid, Mode: FroxelMode}
	ray := camera.RayAt(160.5, 90.5, 320, 180)
	stats := RenderStats{}
	color, trans := integrateVolume(scene, camera, ray, 100, info, DefaultRenderConfig(), &stats)
	if stats.VolumeSamples == 0 || color.LengthSquared() == 0 || trans >= 1 {
		t.Fatalf("transparent volume did not march: color=%+v trans=%v stats=%+v", color, trans, stats)
	}
}

func TestHeatmapCoversEveryGridCell(t *testing.T) {
	scene := NewGalleryScene()
	camera := testCamera(320, 180)
	grid := BuildFroxelGrid(scene, camera, GridConfig{TilesX: 4, TilesY: 3, Slices: 5})
	img := BuildHeatmap(grid, 12)
	if img.Bounds().Dx() < grid.Config.TilesX*12 || img.Bounds().Dy() < grid.Config.TilesY*12 {
		t.Fatalf("heatmap too small: %v", img.Bounds())
	}
	if img.Bounds() == image.Rect(0, 0, 0, 0) {
		t.Fatal("empty heatmap")
	}
}

func containsInt(values []int, target int) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func sameInts(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
