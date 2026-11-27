package main

import (
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

// paintDirtyFrame collects changes, repaints merged dirty rectangles on
// front, then asserts front is pixel-identical to a full repaint.
func paintDirtyFrame(t *testing.T, front *Framebuffer, sc *Scene, tr *DirtyTracker, step string) Region {
	t.Helper()
	reg := tr.Collect(sc)
	merged := reg.Merged()
	if !RectsDisjoint(merged) {
		t.Fatalf("%s: merged rects overlap: %+v", step, merged)
	}
	for _, rc := range merged {
		PaintScene(front, sc, rc)
	}
	full := RenderFull(sc)
	if n := MismatchCount(front, full); n != 0 {
		t.Fatalf("%s: incremental render differs from full repaint at %d pixels", step, n)
	}
	if !front.Equal(full) {
		t.Fatalf("%s: framebuffers not equal", step)
	}
	return reg
}

func TestDemoRunPixelPerfectAndWritesPPM(t *testing.T) {
	dir := t.TempDir()
	cfg := DefaultConfig()
	cfg.OutDir = dir
	cfg.Updates = 3

	rep, err := Run(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !rep.PixelPerfect || rep.TotalMismatchPixels != 0 {
		t.Fatalf("demo run not pixel perfect: mismatches=%d", rep.TotalMismatchPixels)
	}

	for _, name := range []string{
		"incremental_result.ppm", "full_reference.ppm", "dirty_region.ppm",
		"dirty.ppm", "incremental.ppm", "reference.ppm", "dirty_mask.ppm",
		"diff.ppm", "report.txt", "report.json",
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("expected artifact %s: %v", name, err)
		}
	}

	inc, err := ReadPPM(filepath.Join(dir, "incremental_result.ppm"))
	if err != nil {
		t.Fatal(err)
	}
	ref, err := ReadPPM(filepath.Join(dir, "full_reference.ppm"))
	if err != nil {
		t.Fatal(err)
	}
	if !inc.Equal(ref) {
		t.Fatal("saved incremental PPM is not pixel-identical to reference PPM")
	}

	diff, err := ReadPPM(filepath.Join(dir, "diff.ppm"))
	if err != nil {
		t.Fatal(err)
	}
	if n := MismatchCount(diff, NewFramebuffer(diff.W, diff.H)); n != 0 {
		t.Fatalf("diff map must be fully black, found %d marked pixels", n)
	}

	mask, err := ReadPPM(filepath.Join(dir, "dirty_mask.ppm"))
	if err != nil {
		t.Fatal(err)
	}
	whitePx, bgPx := 0, 0
	bgMask := Color{R: 8, G: 10, B: 16}
	for y := 0; y < mask.H; y++ {
		for x := 0; x < mask.W; x++ {
			switch mask.Get(x, y) {
			case Color{R: 255, G: 255, B: 255}:
				whitePx++
			case bgMask:
				bgPx++
			}
		}
	}
	if whitePx == 0 || bgPx == 0 {
		t.Fatalf("dirty mask looks degenerate: white pixels=%d background pixels=%d", whitePx, bgPx)
	}
}

func TestOldDirtyPixelsAreRestored(t *testing.T) {
	font := DefaultFont()
	const w, h = 420, 260
	bg := NewFramebuffer(w, h)
	bgColor := Color{R: 30, G: 90, B: 60}
	bg.Clear(bgColor)

	shape := NewShape("dot", shapeDisk, 90, 130, Color{R: 230, G: 40, B: 40}, font, "")
	sc := &Scene{Width: w, Height: h, Bg: bg, Objects: []Object{shape}}

	front := RenderFull(sc)
	tr := NewDirtyTracker()
	tr.Snapshot(sc)

	old := shape.Bounds()
	if front.Get(90, 130) != (Color{R: 230, G: 40, B: 40}) {
		t.Fatalf("test setup: old center should be shape fill, got %+v", front.Get(90, 130))
	}

	shape.SetCenter(330, 130, "")
	reg := paintDirtyFrame(t, front, sc, tr, "move")

	// The tracked region must include the old position.
	coversOld := false
	for _, r := range reg.Merged() {
		if r.Contains(90, 130) && r.Intersect(old).Empty() == false {
			coversOld = true
		}
	}
	if !coversOld {
		t.Fatalf("dirty region %+v does not cover old shape bounds %+v", reg.Merged(), old)
	}

	// Old center must be restored to the static background, not retain the
	// red disk ("old dirty area cleared" requirement).
	if got := front.Get(90, 130); got != bgColor {
		t.Fatalf("old shape center = %+v, want restored background %+v", got, bgColor)
	}
}

func TestShrinkingTextLeavesNoCanvasRemnants(t *testing.T) {
	font := DefaultFont()
	const w, h = 460, 200
	bg := NewFramebuffer(w, h)
	bg.Clear(Color{R: 18, G: 26, B: 52})

	label := NewLabel("status", 24, 24, false, font, 4,
		Color{R: 255, G: 220, B: 90}, Color{R: 10, G: 16, B: 36}, Color{R: 120, G: 150, B: 220},
		"WWWWWWWWWW")
	sc := &Scene{Width: w, Height: h, Bg: bg, Objects: []Object{label}}

	front := RenderFull(sc)
	tr := NewDirtyTracker()
	tr.Snapshot(sc)
	oldBounds := label.Bounds()

	label.SetText("I")
	newBounds := label.Bounds()
	if newBounds.Width() >= oldBounds.Width() {
		t.Fatalf("new label width %d should be smaller than old %d", newBounds.Width(), oldBounds.Width())
	}

	reg := paintDirtyFrame(t, front, sc, tr, "shrink-text")

	// A point in the old tail (well outside the new bounds) must be covered
	// by the dirty region and equal the full reference / panel restoration.
	tailX := oldBounds.X1 - 8
	tailY := oldBounds.Y0 + oldBounds.Height()/2
	if !newBounds.Contains(tailX, tailY) {
		covered := false
		for _, r := range reg.Merged() {
			if r.Contains(tailX, tailY) {
				covered = true
			}
		}
		if !covered {
			t.Fatalf("old text tail point (%d,%d) not covered by dirty region", tailX, tailY)
		}
	}
	full := RenderFull(sc)
	if got, want := front.Get(tailX, tailY), full.Get(tailX, tailY); got != want {
		t.Fatalf("tail pixel = %+v, full reference = %+v (ghost text?)", got, want)
	}
}

func TestFuzzRandomMovesAndText(t *testing.T) {
	font := DefaultFont()
	const w, h = 420, 300
	bg := BuildBackground(w, h, font)
	sc := &Scene{Width: w, Height: h, Bg: bg}

	shapes := []*ShapeObject{
		NewShape("a", shapeRect, 120, 120, Color{R: 255, G: 159, B: 28}, font, "A 01"),
		NewShape("b", shapeDisk, 300, 180, Color{R: 46, G: 196, B: 182}, font, "B 02"),
		NewShape("c", shapeDiamond, 200, 220, Color{R: 131, G: 56, B: 236}, font, "C 03"),
	}
	for _, s := range shapes {
		sc.add(s)
	}
	label := NewLabel("lbl", 16, 10, false, font, 2,
		Color{R: 255, G: 255, B: 255}, Color{R: 0, G: 0, B: 0}, Color{R: 90, G: 110, B: 170}, "START")
	sc.add(label)

	front := RenderFull(sc)
	tr := NewDirtyTracker()
	tr.Snapshot(sc)

	rng := rand.New(rand.NewSource(20261127))
	alphabet := "ABCXYZ0123456789 !?:-~"
	randomText := func(maxLen int) string {
		n := rng.Intn(maxLen + 1)
		b := make([]byte, n)
		for i := range b {
			b[i] = alphabet[rng.Intn(len(alphabet))]
		}
		return string(b)
	}

	added := false
	for step := 0; step < 400; step++ {
		for idx, s := range shapes {
			cx := 40 + rng.Intn(w-80)
			cy := 70 + rng.Intn(h-110)
			caption := ""
			if rng.Intn(3) != 0 {
				caption = randomText(8)
			}
			s.SetCenter(cx, cy, caption)
			_ = idx
		}
		if rng.Intn(2) == 0 {
			label.SetText(randomText(14))
		}

		if step == 120 && !added {
			extra := NewLabel("late", w-20, h-40, true, font, 2,
				Color{R: 200, G: 255, B: 200}, Color{R: 10, G: 20, B: 10}, Color{R: 120, G: 200, B: 120}, "LATE")
			sc.add(extra)
			added = true
		}

		paintDirtyFrame(t, front, sc, tr, "fuzz-step")
	}
}
