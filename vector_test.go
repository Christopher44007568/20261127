package main

import (
	"image/color"
	"strings"
	"testing"
)

func TestExportSVGKeepsEditableSemantics(t *testing.T) {
	s := BuildScene()
	svg := ExportSVG(s)

	for _, want := range []string{
		`<svg xmlns="http://www.w3.org/2000/svg"`,
		`id="obj-radial-gradient"`,
		`id="obj-clip-group"`,
		`id="clip-stripe-1"`,
		`<linearGradient id="grad-linear-bar"`,
		`<radialGradient id="grad-radial-sun"`,
		`<clipPath id="clip-circle-window"`,
		`fill="url(#grad-linear-bar)"`,
		`stroke="url(#grad-wave-stroke)"`,
		`clip-path="url(#clip-circle-window)"`,
		`stroke-width="9"`,
	} {
		if !strings.Contains(svg, want) {
			t.Fatalf("scene.svg missing %q", want)
		}
	}
	if strings.Contains(strings.ToLower(svg), "<image") {
		t.Fatalf("editable scene.svg must not embed an <image> bitmap")
	}
}

func TestReloadPreservesIDsAndSeparatePaths(t *testing.T) {
	s := BuildScene()
	svg := ExportSVG(s)
	sc, audit, err := ParseSVG(svg)
	if err != nil {
		t.Fatal(err)
	}
	if audit.ImageElements != 0 {
		t.Fatalf("reloaded editable SVG has %d images", audit.ImageElements)
	}
	if audit.GradientElements != len(s.Grads) {
		t.Fatalf("gradients = %d want %d", audit.GradientElements, len(s.Grads))
	}
	if audit.ClipPathElements != len(s.Clips) {
		t.Fatalf("clipPaths = %d want %d", audit.ClipPathElements, len(s.Clips))
	}

	got := map[string]bool{}
	for _, id := range audit.IDs {
		got[id] = true
	}
	for _, o := range s.All {
		if !got[o.ID] {
			t.Fatalf("original id %s missing after reload", o.ID)
		}
	}
	drawables := countDrawables(s)
	if audit.PathElements != drawables {
		t.Fatalf("path elements = %d want %d separate paths", audit.PathElements, drawables)
	}
	if audit.PathElements < 10 {
		t.Fatalf("expected many separate paths, got %d", audit.PathElements)
	}
	if len(sc.All) != len(s.All) {
		t.Fatalf("reloaded object count = %d want %d", len(sc.All), len(s.All))
	}
}

func TestRenderIsOpaqueAndClipped(t *testing.T) {
	s := BuildScene()
	rr := Render(s, 2)
	if rr.Img.Bounds().Dx() != s.W || rr.Img.Bounds().Dy() != s.H {
		t.Fatalf("size = %v", rr.Img.Bounds())
	}
	// Background fill at a canvas corner.
	c := rr.Img.RGBAAt(4, 4)
	if c.R < 220 || c.G < 225 || c.B < 235 {
		t.Fatalf("background corner unexpected color %v", c)
	}
	// Inside the clip window many stripe pixels are vivid.
	vivid := 0
	for y := 170; y < 370; y += 2 {
		for x := 690; x < 905; x += 2 {
			if isVivid(rr.Img.RGBAAt(x, y)) {
				vivid++
			}
		}
	}
	if vivid < 200 {
		t.Fatalf("expected many vivid clipped stripe pixels, got %d", vivid)
	}
	out := rr.Img.RGBAAt(714, 165) // inside a stripe quad, but outside circle clip
	if out.R < 245 || out.G < 245 || out.B < 245 {
		t.Fatalf("clip leaked outside clipPath: %v", out)
	}
	// ID buffer assigns the clip group children inside the window.
	if id := rr.ModalID(760, 200); id < 0 {
		t.Fatalf("expected an object id inside clip window")
	}
}

func isVivid(c color.RGBA) bool {
	mx := maxInt(int(c.R), maxInt(int(c.G), int(c.B)))
	mn := minInt(int(c.R), minInt(int(c.G), int(c.B)))
	return mx-mn > 80
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func TestRoundTripRenderFaithful(t *testing.T) {
	s := BuildScene()
	reloaded, _, err := ParseSVG(ExportSVG(s))
	if err != nil {
		t.Fatal(err)
	}
	a := Render(s, 2).Img
	b := Render(reloaded, 2).Img
	if d := meanImageDelta(a, b); d > 1.5 {
		t.Fatalf("round-trip mean delta too high: %v", d)
	}
}

func TestFaultAttributionFindsInjectedID(t *testing.T) {
	s := BuildScene()
	good := Render(s, 2)
	ref := good.Img

	var target *Obj
	for _, o := range s.All {
		if o.ID == "obj-radial-gradient" {
			target = o
		}
	}
	target.Fill = solid("#64748b")
	bad := Render(s, 2)
	_, rep := ComputeDiff(ref, bad, 42)

	top, topN := "", -1
	for _, st := range rep.Objects {
		if st.ErrorPixels > topN {
			top, topN = st.ID, st.ErrorPixels
		}
	}
	if top != target.ID {
		t.Fatalf("top attributed id = %q (%d), want %q", top, topN, target.ID)
	}
	if topN < 1000 {
		t.Fatalf("expected large fault area, got %d px", topN)
	}
}

func TestFlattenedBadSVGIsFlagged(t *testing.T) {
	bad := BuildFlattenedBadSVG([]byte("AAAA"), 960, 600)
	_, audit, err := ParseSVG(bad)
	if err != nil {
		t.Fatal(err)
	}
	if audit.ImageElements != 1 {
		t.Fatalf("images = %d want 1", audit.ImageElements)
	}
	if audit.PathElements != 0 {
		t.Fatalf("paths = %d want 0 for merged bitmap baseline", audit.PathElements)
	}
}
