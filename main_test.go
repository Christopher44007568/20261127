package main

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func hasIntermediateCoverage(mask *Mask) bool {
	bounds := mask.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			coverage := mask.AlphaAt(x, y)
			if coverage > 8 && coverage < 247 {
				return true
			}
		}
	}
	return false
}

func TestShapeMasksAreAntiAliased(t *testing.T) {
	bounds := image.Rect(0, 0, 220, 220)
	circle := CircleMask(bounds, PointF{X: 110, Y: 110}, 82, 8)
	star := PolygonMask(bounds, StarVertices(PointF{X: 110, Y: 110}, 96, 42, 5, -1.570796), 8)

	for name, mask := range map[string]*Mask{"circle": circle, "star": star} {
		if mask.AlphaAt(2, 2) != 0 {
			t.Fatalf("%s outside corner should be empty", name)
		}
		if mask.AlphaAt(110, 110) != 255 {
			t.Fatalf("%s center should be fully covered", name)
		}
		if !hasIntermediateCoverage(mask) {
			t.Fatalf("%s has no anti-aliased edge coverage", name)
		}
		visual := mask.Visualize(color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		if pixel := visual.NRGBAAt(2, 2); pixel != (color.NRGBA{}) {
			t.Fatalf("%s transparent mask pixel must be zero NRGBA: %+v", name, pixel)
		}
	}
}

func TestTextCounterIsHollowAndAntiAliased(t *testing.T) {
	bounds := image.Rect(0, 0, 240, 200)
	mask := TextMask(bounds, "O", 52, 18, 160, 8)

	if coverage := mask.AlphaAt(116, 98); coverage != 0 {
		t.Fatalf("O counter must be hollow, coverage=%d", coverage)
	}
	if coverage := mask.AlphaAt(160, 98); coverage < 200 {
		t.Fatalf("O stroke should be covered, coverage=%d", coverage)
	}
	if !hasIntermediateCoverage(mask) {
		t.Fatal("text edges should be anti-aliased")
	}
}

func TestInverseAndIntersectMask(t *testing.T) {
	bounds := image.Rect(0, 0, 100, 100)
	circle := CircleMask(bounds, PointF{X: 50, Y: 50}, 40, 4)
	inverse := circle.Inverse()

	if inverse.AlphaAt(50, 50) != 0 {
		t.Fatal("inverse center must be empty")
	}
	if inverse.AlphaAt(2, 2) != 255 {
		t.Fatal("inverse outside must be fully covered")
	}

	other := CircleMask(bounds, PointF{X: 72, Y: 50}, 30, 4)
	region := Intersect(circle, other)
	if region.AlphaAt(72, 50) != 255 {
		t.Fatal("overlap center should be covered")
	}
	if region.AlphaAt(18, 50) != 0 {
		t.Fatal("circle-only point should be absent from intersection")
	}
}

func rectMask(bounds, rect image.Rectangle) *Mask {
	mask := NewMask(bounds)
	for y := rect.Min.Y; y < rect.Max.Y; y++ {
		for x := rect.Min.X; x < rect.Max.X; x++ {
			mask.setAlpha(x, y, 255)
		}
	}
	return mask
}

func TestClipStackPopIsolatesNestedClip(t *testing.T) {
	bounds := image.Rect(0, 0, 20, 20)
	canvas := NewCanvas(bounds)
	outer := rectMask(bounds, image.Rect(2, 2, 18, 18))
	inner := rectMask(bounds, image.Rect(2, 2, 10, 18))
	red := color.NRGBA{R: 255, A: 255}
	green := color.NRGBA{G: 255, A: 255}
	blue := color.NRGBA{B: 255, A: 255}

	canvas.Push(outer, ClipReplace)
	canvas.Push(inner, ClipIntersect)
	canvas.Fill(bounds, Solid(red))
	if alpha := canvas.Image().NRGBAAt(15, 10).A; alpha != 0 {
		t.Fatalf("inner clip leaked during nested fill, alpha=%d", alpha)
	}

	canvas.Pop()
	canvas.Fill(bounds, Solid(green))
	if pixel := canvas.Image().NRGBAAt(15, 10); pixel.A != 255 || pixel.G != 255 {
		t.Fatalf("sibling after Pop must paint outside nested mask: %+v", pixel)
	}
	if alpha := canvas.Image().NRGBAAt(0, 10).A; alpha != 0 {
		t.Fatalf("parent clip should remain after inner Pop, alpha=%d", alpha)
	}

	canvas.Pop()
	canvas.Fill(bounds, Solid(blue))
	if pixel := canvas.Image().NRGBAAt(0, 10); pixel.B != 255 || pixel.A != 255 {
		t.Fatalf("outer Pop should restore unclipped drawing: %+v", pixel)
	}
}

func TestCutCreatesZeroedHole(t *testing.T) {
	canvas := NewCanvas(image.Rect(0, 0, 8, 8))
	canvas.Fill(canvas.Bounds(), Solid(color.NRGBA{R: 200, G: 100, B: 50, A: 255}))
	hole := NewMask(canvas.Bounds())
	hole.setAlpha(4, 4, 255)

	canvas.Cut(hole)
	if pixel := canvas.Image().NRGBAAt(4, 4); pixel != (color.NRGBA{}) {
		t.Fatalf("cut pixel must be transparent black, got %+v", pixel)
	}
	if pixel := canvas.Image().NRGBAAt(0, 0); pixel.A != 255 {
		t.Fatalf("uncut pixel should remain opaque, got %+v", pixel)
	}
}

func TestRenderDemoWritesVerifiablePNGs(t *testing.T) {
	outputDir := t.TempDir()
	if err := RenderDemo(outputDir); err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"circle_mask.png", "star_mask.png", "text_mask.png",
		"inverse_circle_mask.png", "intersection_mask.png",
		"final_composite.png", "final_preview_checker.png",
		"triple_overlay.png", "triple_overlay_preview.png",
		"triple_clip_stack.png", "triple_clip_stack_preview.png",
		"triple_intersection_mask.png",
	}
	for _, name := range expected {
		if _, err := os.Stat(filepath.Join(outputDir, name)); err != nil {
			t.Fatalf("expected %s: %v", name, err)
		}
	}

	file, err := os.Open(filepath.Join(outputDir, "final_composite.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	decoded, err := png.Decode(file)
	if err != nil {
		t.Fatal(err)
	}

	if alpha := decoded.At(220, 700).(color.NRGBA).A; alpha != 0 {
		t.Fatalf("inverse center in final PNG must be hollow, alpha=%d", alpha)
	}
	if alpha := decoded.At(1084, 264).(color.NRGBA).A; alpha != 0 {
		t.Fatalf("text counter in final PNG must be hollow, alpha=%d", alpha)
	}
	sibling := decoded.At(900, 769).(color.NRGBA)
	if sibling.A == 0 || sibling.G < 80 || sibling.R > 80 {
		t.Fatalf("nested sibling bar missing or wrong color: %+v", sibling)
	}
}
