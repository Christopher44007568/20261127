package studio

import (
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func almost(a, b float32) bool {
	return math.Abs(float64(a-b)) < 1e-5
}

func TestBlendModes(t *testing.T) {
	backdrop := Hex("203040")
	source := Hex("e05020")
	cases := []struct {
		mode BlendMode
		want RGBA
	}{
		{Normal, source},
		{Multiply, RGBA{
			R: source.R * backdrop.R,
			G: source.G * backdrop.G,
			B: source.B * backdrop.B,
			A: 1,
		}},
		{Screen, RGBA{
			R: source.R + backdrop.R - source.R*backdrop.R,
			G: source.G + backdrop.G - source.G*backdrop.G,
			B: source.B + backdrop.B - source.B*backdrop.B,
			A: 1,
		}},
	}
	for _, tc := range cases {
		dst := NewImage(1, 1)
		dst.SetStraight(0, 0, backdrop)
		src := NewImage(1, 1)
		src.SetStraight(0, 0, source)
		dst.Blend(src, tc.mode)
		got := dst.At(0, 0)
		if !almost(got.R, tc.want.R) || !almost(got.G, tc.want.G) || !almost(got.B, tc.want.B) {
			t.Fatalf("%s: got %.4f %.4f %.4f, want %.4f %.4f %.4f",
				tc.mode, got.R, got.G, got.B, tc.want.R, tc.want.G, tc.want.B)
		}
	}
}

func TestPremultipliedDarkEdgeHasChroma(t *testing.T) {
	edge := NewImage(1, 1)
	orange := RGBA{R: 0.9, G: 0.12, B: 0.02, A: 0.62}
	edge.SetStraight(0, 0, orange)
	edge.ApplyMask([]float32{0.35})

	p := edge.At(0, 0)
	if p.A <= 0 {
		t.Fatal("edge coverage disappeared")
	}
	if !almost(p.R/p.A, orange.R) || !almost(p.G/p.A, orange.G) || !almost(p.B/p.A, orange.B) {
		t.Fatalf("mask changed hue: premul=%+v", p)
	}

	final := NewImage(1, 1)
	final.Clear(RGBA{})
	final.Blend(edge, Normal)
	q := final.At(0, 0)
	if q.A <= 0 || q.R <= q.G*2 || q.B >= q.G {
		t.Fatalf("dark-background edge turned grey: %+v", q)
	}
}

func TestClipStackIntersectionAndPop(t *testing.T) {
	stack := NewClipStack(4, 4)
	left := RectMask(4, 4, 0, 0, 2, 4)
	top := RectMask(4, 4, 0, 0, 4, 2)
	stack.Push(left)
	stack.Push(top)
	if stack.Depth() != 2 {
		t.Fatalf("depth=%d", stack.Depth())
	}
	cur := stack.Current()
	if cur[0] != 1 || cur[3] != 0 || cur[12] != 0 {
		t.Fatalf("intersection wrong: %v", cur)
	}
	stack.Pop()
	cur = stack.Current()
	if cur[0] != 1 || cur[3] != 0 || cur[12] != 1 || cur[15] != 0 {
		t.Fatalf("pop did not restore left clip: %v", cur)
	}
}

func TestPeriodicTileHasNoSeamStep(t *testing.T) {
	for i := 0; i <= 256; i++ {
		u := float64(i) / 256
		a := PeriodicTileColor(0, u)
		b := PeriodicTileColor(1, u)
		for _, pair := range [][2]float32{{a.R, b.R}, {a.G, b.G}, {a.B, b.B}} {
			if math.Abs(float64(pair[0]-pair[1])) > 1e-6 {
				t.Fatalf("horizontal seam at v=%.3f: %v vs %v", u, a, b)
			}
		}
	}
	if d := tileBoundaryDelta(320, 100); d > 1e-6 {
		t.Fatalf("tile boundary delta=%g", d)
	}
	tile := TileImage(320, 100, 3, 2)
	if residual := seamResidual(ToNRGBA(tile, false), 320, 100, 3, 2); residual > 1 {
		t.Fatalf("8-bit seam residual=%d LSB", residual)
	}
}

func TestDitherBreaksRampBands(t *testing.T) {
	raw := RampNRGBA(736, 210, false)
	_, rawRuns := DetectBandMask(raw, 8)
	if rawRuns < 10 {
		t.Fatalf("raw ramp should expose bands, got %d runs", rawRuns)
	}
	dithered := RampNRGBA(736, 210, true)
	_, ditherRuns := DetectBandMask(dithered, 8)
	if ditherRuns != 0 {
		t.Fatalf("dithered ramp still has %d runs", ditherRuns)
	}
	again := RampNRGBA(736, 210, true)
	for i := range dithered.Pix {
		if dithered.Pix[i] != again.Pix[i] {
			t.Fatal("dither is not deterministic")
		}
	}
}

func TestGenerateOutputs(t *testing.T) {
	if testing.Short() {
		t.Skip("full artifact generation skipped in -short mode")
	}
	dir := t.TempDir()
	report, err := Generate(dir)
	if err != nil {
		t.Fatal(err)
	}
	required := []string{
		"final.png",
		"final_normal.png",
		"final_multiply.png",
		"final_screen.png",
		"layer_sheet.png",
		"blend_modes.png",
		"order_compare.png",
		"banding_detection.png",
		"seamless_tile.png",
		"layers/00_background.png",
		"layers/01_group_mask.png",
		"layers/02_linear_group.png",
		"layers/03_radial_group.png",
		"layers/04_alpha_mask.png",
		"layers/05_premask_stack.png",
		"layers/06_final_normal.png",
	}
	for _, rel := range required {
		if _, err := os.Stat(filepath.Join(dir, rel)); err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
	}
	if report.Checks.RawBandRuns <= report.Checks.DitheredBandRuns {
		t.Fatalf("band detection did not improve: raw=%g dither=%g",
			report.Checks.RawBandRuns, report.Checks.DitheredBandRuns)
	}
	if report.Checks.TileBoundaryDelta > 1e-6 {
		t.Fatalf("tile delta=%g", report.Checks.TileBoundaryDelta)
	}
	if report.Checks.SeamResidualLSB > 1 {
		t.Fatalf("seam residual=%g LSB", report.Checks.SeamResidualLSB)
	}

	f, err := os.Open(filepath.Join(dir, "final.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		t.Fatal(err)
	}
	if img.Bounds().Dx() != CanvasW || img.Bounds().Dy() != CanvasH {
		t.Fatalf("final bounds=%v", img.Bounds())
	}
}
