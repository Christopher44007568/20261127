package main

import (
	"fmt"
	"math"
	"os"
)

func main() {
	outDir := "out"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fatal(err)
	}

	// Four equal-radius decals: one on a broad slope, one on a sharp bump,
	// one in a narrow dimple, and one deliberately overlapping the first.
	decals := []Decal{
		newDecal("broad-rings", -1.1, 0.1, 0.85, patRings, 1.0, v3(0.95, 0.45, 0.25)),
		newDecal("bump-rib", -2.8, 1.2, 0.85, patRib, 1.0, v3(0.35, 0.75, 0.95)),
		newDecal("dimple-bump", 2.9, -1.3, 0.85, patBump, 1.0, v3(0.55, 0.9, 0.45)),
		newDecal("overlap-rib", -0.2, -0.7, 0.85, patRib, 1.0, v3(0.8, 0.5, 0.95)),
	}

	const (
		width  = 900
		height = 675
		viewW  = 8.4
		viewH  = viewW * height / width
	)
	cam := newCamera(v3(0.9, -7.5, 4.6), v3(0, 0, 0), viewW, viewH, width, height)

	scene := newScene(cam, decals)
	scene.render()
	jobs := []struct {
		name string
		img  *FloatImage
	}{
		{"01_original_surface.png", scene.Base},
		{"02_decal_normals.png", scene.DecalLayer},
		{"03_final_lighting_rnm.png", scene.Final},
		{"04_naive_linear_average.png", scene.Naive},
	}
	for _, job := range jobs {
		if err := job.img.writePNG(outDir + "/" + job.name); err != nil {
			fatal(err)
		}
		fmt.Println("wrote", job.name)
	}
	if err := scene.writeComparison(outDir + "/05_comparison_split.png"); err != nil {
		fatal(err)
	}
	fmt.Println("wrote 05_comparison_split.png (left: linear average, right: reoriented)")

	printScaleStability(cam, decals)
	footprints := scene.footprintWidths()
	fmt.Println("\nrendered mask width (screen pixels):")
	for i, d := range decals {
		fmt.Printf("  %-14s %8.0f px\n", d.Name, footprints[i])
	}
	assertScaleStable(footprints)
	maxAll, maxOverlap, overlapPixels := scene.MaxDevAll, scene.MaxDevOver, scene.OverlapCount
	fmt.Printf("\nunit-normal audit (RNM):\n")
	fmt.Printf("  overlap pixels:        %d\n", overlapPixels)
	fmt.Printf("  max |len(n)-1| all:    %.3e\n", maxAll)
	fmt.Printf("  max |len(n)-1| overlap:%.3e\n", maxOverlap)
	if maxAll > 1e-12 {
		fmt.Println("  FAIL: final normal left unit length")
		os.Exit(1)
	}
	fmt.Println("  OK: every final normal stays unit length")
}

// assertScaleStable fails the run if the rendered footprint of a decal
// changes materially when it is moved to a different curvature region.
func assertScaleStable(widths []float64) {
	mean := 0.0
	for _, w := range widths {
		mean += w
	}
	mean /= float64(len(widths))
	for _, w := range widths {
		if pct := 100 * abs(w-mean) / mean; pct > 5 {
			fmt.Printf("  FAIL: decal scale drift %.2f%% exceeds 5%%\n", pct)
			os.Exit(1)
		}
	}
	fmt.Printf("  OK: rendered footprint within 5%% of mean %.1f px\n", mean)
}

// printScaleStability measures each projector's footprint in screen pixels
// along two tangent directions. A stable world-space scale means decals on
// high-curvature features show the same pixel footprint as decals on flats.
func printScaleStability(cam Camera, decals []Decal) {
	fmt.Println("\nscale stability (projector footprint, pixels):")
	fmt.Printf("  %-14s %8s %8s\n", "decal", "along-T1", "along-T2")
	widths := []float64{}
	for _, d := range decals {
		span := func(t Vec3) float64 {
			a := d.Frame.Origin.Add(t.Scale(-d.Radius))
			b := d.Frame.Origin.Add(t.Scale(d.Radius))
			pa, pb := worldToScreen(cam, a), worldToScreen(cam, b)
			return pa.sub(pb).Length()
		}
		s1, s2 := span(d.Frame.T1), span(d.Frame.T2)
		widths = append(widths, s1, s2)
		fmt.Printf("  %-14s %8.1f %8.1f\n", d.Name, s1, s2)
	}
	report := func(label string, values []float64) {
		mean := 0.0
		for _, v := range values {
			mean += v
		}
		mean /= float64(len(values))
		spread := 0.0
		for _, v := range values {
			if d := abs(v - mean); d > spread {
				spread = d
			}
		}
		fmt.Printf("  %s mean %.1f px, max deviation %.1f px (%.2f%%)\n",
			label, mean, spread, 100*spread/mean)
	}
	// Compare the same tangent direction across decals so camera tilt
	// (foreshortening) does not contaminate the cross-curvature comparison.
	t1, t2 := []float64{}, []float64{}
	for i, w := range widths {
		if i%2 == 0 {
			t1 = append(t1, w)
		} else {
			t2 = append(t2, w)
		}
	}
	report("T1 across decals", t1)
	report("T2 across decals", t2)
	fmt.Println("  note: T2 screen spans vary because the tilted camera foreshortens")
	fmt.Println("  that axis by depth; the world-space footprint below is identical")
	fmt.Println("  world-space footprint is constant: diameter =",
		fmt.Sprintf("%.2f", 2*decals[0].Radius), "units on every decal")
}

type screenPt struct{ X, Y float64 }

func (p screenPt) sub(q screenPt) screenPt { return screenPt{p.X - q.X, p.Y - q.Y} }
func (p screenPt) Length() float64         { return math.Hypot(p.X, p.Y) }

func worldToScreen(cam Camera, p Vec3) screenPt {
	rel := p.Sub(cam.Origin)
	sx := rel.Dot(cam.Right) / cam.ViewW
	sy := rel.Dot(cam.Up) / cam.ViewH
	return screenPt{sx * float64(cam.WidthPx), sy * float64(cam.HeightPx)}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
