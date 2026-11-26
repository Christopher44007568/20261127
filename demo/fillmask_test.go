package demo

import (
	"image"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func sample(t *testing.T, m *Mask, x, y int, want byte, name string) {
	t.Helper()
	if got := m.At(x, y); got != want {
		t.Fatalf("%s at (%d,%d): got %d want %d", name, x, y, got, want)
	}
}

// The same self-intersecting pentagram must produce different holes:
// EvenOdd leaves the central pentagon open, NonZero fills it (wn == 2).
func TestPentagramRulesProduceDifferentHoles(t *testing.T) {
	path := StarPath()
	eo := NewMask(SceneW, SceneH)
	nz := NewMask(SceneW, SceneH)
	FillPath(eo, path, FillEvenOdd)
	FillPath(nz, path, FillNonZero)

	sample(t, eo, 180, 152, 0, "evenodd star core")
	sample(t, nz, 180, 152, 255, "nonzero star core")

	diff := 0
	for i := range eo.Pix {
		if eo.Pix[i] != nz.Pix[i] {
			diff++
		}
	}
	if diff < 1000 {
		t.Fatalf("expected substantial rule difference, got %d pixels", diff)
	}

	evenOdd, wn := ClassifyPoint(path, 180.5, 152.5)
	if evenOdd || wn != 2 {
		t.Fatalf("center classification: parity=%v wn=%d, want false/2", evenOdd, wn)
	}
}

// Two overlapping rectangles isolate the fill-rule semantics without curve
// geometry: same winding overlap has parity 0 but winding 2; opposite
// winding overlap has winding 0 as well.
func TestOverlappingRectRules(t *testing.T) {
	same := &Path{}
	AddRect(same, 40, 40, 160, 160, WindingClockwise)
	AddRect(same, 100, 100, 220, 220, WindingClockwise)

	eo := NewMask(260, 260)
	nz := NewMask(260, 260)
	FillPath(eo, same, FillEvenOdd)
	FillPath(nz, same, FillNonZero)

	sample(t, eo, 70, 70, 255, "A only evenodd")
	sample(t, nz, 70, 70, 255, "A only nonzero")
	sample(t, eo, 130, 130, 0, "overlap evenodd hole")
	sample(t, nz, 130, 130, 255, "overlap nonzero same winding")

	opposite := &Path{}
	AddRect(opposite, 40, 40, 160, 160, WindingClockwise)
	AddRect(opposite, 100, 100, 220, 220, WindingCounterClock)
	nzOpp := NewMask(260, 260)
	FillPath(nzOpp, opposite, FillNonZero)
	sample(t, nzOpp, 130, 130, 0, "overlap nonzero opposite winding hole")
	sample(t, nzOpp, 190, 190, 255, "B only opposite winding")
}

// Nested outer/hole/island subpaths must classify identically under both
// rules because windings oppose each other at every nesting level.
func TestNestedSubpathSamples(t *testing.T) {
	for _, rule := range []FillRule{FillEvenOdd, FillNonZero} {
		m := NewMask(SceneW, SceneH)
		FillPath(m, NestedPath(), rule)
		for _, s := range NestedSamples() {
			sample(t, m, s.X, s.Y, s.Want, rule.String()+"/"+s.Name)
		}
		if m.HasIntermediateValue() {
			t.Fatalf("%s mask has fractional coverage", rule)
		}
	}
}

// Holes are never painted: pixels covered by the outer contour alone but left
// open by the unified nested mask must stay byte-identical to the paper.
func TestHolesNotCoveredByMainContour(t *testing.T) {
	nested := NewMask(SceneW, SceneH)
	FillPath(nested, NestedPath(), FillNonZero)

	paper := NewPaper(SceneW, SceneH, DefaultPalette())
	scene := cloneNRGBA(paper)
	PaintMask(scene, nested, DefaultPalette().NestedNZ)

	rep := &Report{OK: true}
	VerifyHoleCompositing(paper, scene, nested, rep)
	if !rep.OK {
		t.Fatalf("hole compositing check failed: %+v", rep.Checks)
	}
	if rep.ProtectedHolePx < 1000 {
		t.Fatalf("protected hole area too small: %d", rep.ProtectedHolePx)
	}

	// A naive union of independently filled subpaths would cover the hole;
	// assert the test can tell the difference so it is not vacuous.
	naive := NewMask(SceneW, SceneH)
	outer := &Path{}
	AddRoundedRect(outer, 36, 28, 324, 272, 30, WindingClockwise)
	FillPath(naive, outer, FillNonZero)
	holeRect := NewMask(SceneW, SceneH)
	hp := &Path{}
	AddRect(hp, 108, 88, 252, 212, WindingCounterClock)
	FillPath(holeRect, hp, FillNonZero)
	for i := range naive.Pix {
		if holeRect.Pix[i] != 0 {
			naive.Pix[i] = 255 // naive OR of "fill every contour"
		}
	}
	if naive.At(130, 110) != 255 {
		t.Fatalf("naive union sanity: expected covered pixel")
	}
	if nested.At(130, 110) != 0 {
		t.Fatalf("unified mask must keep the hole open")
	}
}

// Integer clip rectangles keep whole pixels only and leave a continuous,
// fully covered boundary column/row wherever the shape reaches the edge.
func TestClipRectSeams(t *testing.T) {
	before := NewMask(SceneW, SceneH)
	FillPath(before, NestedPath(), FillNonZero)

	clipped := before.Clone()
	ClipRect(clipped, ClipRectNested)

	if clipped.HasIntermediateValue() {
		t.Fatal("clipped mask has intermediate coverage")
	}

	// Outside is cleared, inside equals the original exactly.
	r := ClipRectNested
	for y := 0; y < SceneH; y++ {
		for x := 0; x < SceneW; x++ {
			inside := x >= r.Min.X && x < r.Max.X && y >= r.Min.Y && y < r.Max.Y
			if inside {
				if clipped.At(x, y) != before.At(x, y) {
					t.Fatalf("inside clip (%d,%d) changed: %d vs %d", x, y, clipped.At(x, y), before.At(x, y))
				}
			} else if clipped.At(x, y) != 0 {
				t.Fatalf("outside clip (%d,%d) not cleared", x, y)
			}
		}
	}

	edges := []SeamReport{
		CheckVerticalClipSeam(clipped, before, r.Min.X, false, 80, 224),
		CheckVerticalClipSeam(clipped, before, r.Max.X, true, 80, 224),
		CheckHorizontalClipSeam(clipped, before, r.Min.Y, false, 120, 250),
		CheckHorizontalClipSeam(clipped, before, r.Max.Y, true, 120, 250),
	}
	for _, e := range edges {
		if !e.OK {
			t.Fatalf("seam %s@%d failed: %+v", e.Edge, e.Fixed, e)
		}
	}

	// Explicit inclusive/exclusive boundary semantics.
	sample(t, clipped, r.Min.X, 150, 255, "min-x boundary pixel")
	sample(t, clipped, r.Max.X-1, 150, 255, "max-x last pixel")
	if clipped.At(r.Max.X, 150) != 0 {
		t.Fatal("exclusive max-x column must be outside clip")
	}
}

func TestClipByArbitraryPath(t *testing.T) {
	m := NewMask(SceneW, SceneH)
	FillPath(m, StarPath(), FillEvenOdd)
	before := m.Count()
	ClipPath(m, ClipCircleStar(), FillNonZero)
	if m.HasIntermediateValue() {
		t.Fatal("path clip introduced fractional coverage")
	}
	if m.Count() >= before {
		t.Fatalf("circle clip should remove pixels: before=%d after=%d", before, m.Count())
	}
}

func TestSnapClipRectOutward(t *testing.T) {
	got := SnapClipRectOutward(10.2, 20.7, 30.9, 40.1)
	want := image.Rect(10, 20, 31, 41)
	if got != want {
		t.Fatalf("outward snap = %v, want %v", got, want)
	}
}

// End-to-end run must write every required artifact and a passing report.
func TestRunWritesArtifacts(t *testing.T) {
	dir := t.TempDir()
	rep, err := Run(dir)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if !rep.OK {
		t.Fatalf("report not OK: %+v", rep.Checks)
	}
	required := []string{
		"composite.png",
		"boundary_zooms.png",
		"report.json",
		"mask_star_evenodd.png",
		"mask_star_nonzero.png",
		"mask_nested_evenodd.png",
		"mask_nested_nonzero.png",
		"mask_clipped_rect_nonzero.png",
		"mask_clipped_circle_evenodd.png",
		"zoom_star_core_evenodd.png",
		"zoom_star_core_nonzero.png",
		"zoom_hole_edge.png",
		"zoom_clip_edge.png",
	}
	for _, f := range required {
		p := filepath.Join(dir, f)
		st, err := os.Stat(p)
		if err != nil || st.Size() == 0 {
			t.Fatalf("missing or empty artifact %s: %v", f, err)
		}
	}

	// Composite must be a decodable PNG with substantial dimensions.
	f, err := os.Open(filepath.Join(dir, "composite.png"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		t.Fatalf("decode composite config: %v", err)
	}
	if cfg.Width < 800 || cfg.Height < 600 {
		t.Fatalf("composite too small: %dx%d", cfg.Width, cfg.Height)
	}
}
