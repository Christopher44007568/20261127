package cover_test

import (
	"math"
	"strconv"
	"testing"

	"coverlab/cover"
	"coverlab/scenes"
	"coverlab/viz"
)

func trackedCoverage(t *testing.T, dy float64, method string) float64 {
	t.Helper()
	scene := scenes.SweepScene(dy)
	var cov []float64
	switch method {
	case "analytic":
		cov = cover.AnalyticLayer(scene.Layers[0], scene.W, scene.H)
	default:
		n, err := strconv.Atoi(method[2:])
		if err != nil {
			t.Fatal(err)
		}
		cov = cover.SampledLayer(scene.Layers[0], scene.W, scene.H, n)
	}
	return cov[scenes.SweepPixelY*scene.W+scenes.SweepPixelX]
}

func TestAnalyticThinLineContinuity(t *testing.T) {
	// Shifting a 0.52px line in 0.05px steps must change any fixed pixel's
	// analytic coverage by at most the shift (Lipschitz 1).
	maxJump, prev := 0.0, 0.0
	seenMin, seenMax := 1.0, 0.0
	first := true
	for i := -40; i <= 40; i++ {
		dy := float64(i) * 0.05
		c := trackedCoverage(t, dy, "analytic")
		if !first {
			if d := math.Abs(c - prev); d > maxJump {
				maxJump = d
			}
		}
		first = false
		prev = c
		seenMin = math.Min(seenMin, c)
		seenMax = math.Max(seenMax, c)
	}
	if maxJump > 0.05+1e-9 {
		t.Fatalf("analytic brightness not continuous: max 0.05-step jump %g", maxJump)
	}
	if seenMax-seenMin < 0.4 {
		t.Fatalf("sweep did not traverse coverage range: [%g,%g]", seenMin, seenMax)
	}

	// Headline 0.1px steps used by the report.
	headline := 0.0
	for i := -10; i < 10; i++ {
		a := trackedCoverage(t, float64(i)/10, "analytic")
		b := trackedCoverage(t, float64(i+1)/10, "analytic")
		headline = math.Max(headline, math.Abs(b-a))
	}
	if headline > 0.1+1e-9 {
		t.Fatalf("0.1px jump %g exceeds bound", headline)
	}
}

func TestSampledCoverageIsQuantized(t *testing.T) {
	for _, tc := range []struct {
		method  string
		n       int
		minJump float64
	}{
		{"ss2", 2, 0.5}, // horizontal line: a sample row flips N hits -> 1/N
		{"ss3", 3, 1.0 / 3.0},
	} {
		maxJump, prev := 0.0, 0.0
		for i := -10; i <= 10; i++ {
			c := trackedCoverage(t, float64(i)/10, tc.method)
			k := math.Round(c * float64(tc.n*tc.n))
			if math.Abs(c-float64(k)/float64(tc.n*tc.n)) > 1e-12 {
				t.Fatalf("%s coverage %g not a 1/%d grid step", tc.method, c, tc.n*tc.n)
			}
			if i > -10 {
				maxJump = math.Max(maxJump, math.Abs(c-prev))
			}
			prev = c
		}
		if maxJump < tc.minJump-1e-9 {
			t.Fatalf("%s max jump %g, want quantization jump >= %g", tc.method, maxJump, tc.minJump)
		}
	}
}

func TestDisjointContoursDoNotStain(t *testing.T) {
	scene := scenes.Segregate()
	redCov := cover.AnalyticLayer(scene.Layers[0], scene.W, scene.H)
	gidx := scenes.GuardPixelY*scene.W + scenes.GuardPixelX
	if redCov[gidx] != 0 {
		t.Fatalf("guard pixel has red coverage %g", redCov[gidx])
	}

	noRed := scene
	noRed.Layers = append([]cover.Layer{}, scene.Layers[1:]...)

	for _, m := range []string{"analytic", "ss2", "ss3"} {
		full, _, _ := cover.RenderMethod(scene, m, viz.White)
		without, _, _ := cover.RenderMethod(noRed, m, viz.White)
		rep := cover.MeasureStain("red", redCov, full, without)
		if rep.MaxStain255 != 0 {
			t.Fatalf("%s stained disjoint pixels by %g/255", m, rep.MaxStain255)
		}
		if g := full[gidx]; g.R != 255 || g.G != 255 || g.B != 255 {
			t.Fatalf("%s guard pixel not pure white: %+v", m, g)
		}
	}

	fb, _ := cover.RenderBlur(scene, 1.15, viz.White, -1)
	nb, _ := cover.RenderBlur(scene, 1.15, viz.White, 0)
	blurRep := cover.MeasureStain("red", redCov, fb, nb)
	if blurRep.MaxStain255 <= 1 {
		t.Fatalf("blur baseline unexpectedly clean: max stain %g", blurRep.MaxStain255)
	}
	if g := fb[gidx]; g.R > 250 || g.G > 250 || g.B > 250 {
		t.Fatalf("blur guard pixel expected tint, got %+v", g)
	}
}

func TestSamplingConvergesToAnalytic(t *testing.T) {
	for _, sc := range []cover.Scene{scenes.Diagonal(), scenes.Cross()} {
		_, ref, _ := cover.RenderMethod(sc, "analytic", viz.White)
		_, coarse, _ := cover.RenderMethod(sc, "ss2", viz.White)
		_, fine, _ := cover.RenderMethod(sc, "ss8", viz.White)
		e2 := cover.DiffStats(ref, coarse)
		e8 := cover.DiffStats(ref, fine)
		if !(e8.RMSE < e2.RMSE) {
			t.Fatalf("%s: ss8 rmse %g not below ss2 rmse %g", sc.Name, e8.RMSE, e2.RMSE)
		}
		if e8.RMSE > 0.02 {
			t.Fatalf("%s: ss8 rmse %g still too large", sc.Name, e8.RMSE)
		}
	}
}

func TestAllSceneCoveragesInRange(t *testing.T) {
	for _, sc := range scenes.All() {
		for _, m := range []string{"analytic", "ss2", "ss3"} {
			_, alpha, _ := cover.RenderMethod(sc, m, viz.White)
			for i, c := range alpha {
				if c < 0 || c > 1+1e-9 || math.IsNaN(c) {
					t.Fatalf("%s %s bad coverage %g at %d", sc.Name, m, c, i)
				}
			}
		}
	}
}

func TestNearDiagContoursDoNotStain(t *testing.T) {
	scene := scenes.NearDiag()
	for layerIdx := range scene.Layers {
		refCov := cover.AnalyticLayer(scene.Layers[layerIdx], scene.W, scene.H)
		without := scene
		without.Layers = append([]cover.Layer{}, scene.Layers...)
		without.Layers = append(without.Layers[:layerIdx], without.Layers[layerIdx+1:]...)
		for _, m := range []string{"analytic", "ss2", "ss3"} {
			full, _, _ := cover.RenderMethod(scene, m, viz.White)
			removed, _, _ := cover.RenderMethod(without, m, viz.White)
			rep := cover.MeasureStain(scene.Layers[layerIdx].Name, refCov, full, removed)
			if rep.MaxStain255 != 0 {
				t.Fatalf("layer %d %s stains by %g/255", layerIdx, m, rep.MaxStain255)
			}
		}
	}
}
