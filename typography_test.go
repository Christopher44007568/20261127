package main

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func testFontsOrSkip(t *testing.T) *FontSet {
	t.Helper()
	fs, err := LoadFontSet()
	if err != nil {
		t.Skipf("system fonts unavailable, skipping: %v", err)
	}
	return fs
}

func TestSampleArcLengthUniformSynthetic(t *testing.T) {
	c := Contour{Pts: []Pt{{0, 0}, {100, 0}, {100, 40}, {0, 40}}, Closed: true}
	if math.Abs(c.Length()-280) > 1e-9 {
		t.Fatalf("perimeter = %v, want 280", c.Length())
	}
	n := 28
	s := c.SampleArcLength(n)
	if len(s) != n {
		t.Fatalf("sample count = %d, want %d", len(s), n)
	}
	step := c.Length() / float64(n)
	for i, p := range s {
		want := c.Length() * float64(i) / float64(n)
		if math.Abs(p.Dist-want) > 1e-6 {
			t.Fatalf("sample %d dist = %v, want %v", i, p.Dist, want)
		}
		next := c.Length()
		if i+1 < n {
			next = s[i+1].Dist
		}
		if gap := next - p.Dist; math.Abs(gap-step) > 1e-6 {
			t.Fatalf("arc gap %d = %v, want %v", i, gap, step)
		}
	}
}

func TestGlyphSamplingArcUniform(t *testing.T) {
	fs := testFontsOrSkip(t)
	for _, r := range []rune{'永', 'g', 'o', '国', '8'} {
		f, g := fs.Glyph(r)
		o := GlyphOutline(g, f.UnitsPerEm, 240, 0.2)
		for ci, c := range o.Contours {
			n := 40
			s := c.SampleArcLength(n)
			if len(s) != n {
				t.Fatalf("%c contour %d: got %d samples", r, ci, len(s))
			}
			step := c.Length() / float64(n)
			for i, p := range s {
				next := c.Length()
				if i+1 < n {
					next = s[i+1].Dist
				}
				if err := math.Abs(next - p.Dist - step); err > 1e-6 {
					t.Fatalf("%c contour %d gap %d arc error %v", r, ci, i, err)
				}
			}
		}
	}
}

func TestTrackingMovesPen(t *testing.T) {
	fs := testFontsOrSkip(t)
	sty := LayoutStyle{Ppem: 100, TrackingEm: 0.1, MaxWidth: 1e7}
	lay := fs.Layout("AB", sty)
	gp := lay.Lines[0].Glyphs
	if len(gp) != 2 {
		t.Fatalf("glyphs = %d", len(gp))
	}
	want := gp[0].Advance + 10
	if got := gp[1].PenX - gp[0].PenX; math.Abs(got-want) > 1e-6 {
		t.Fatalf("pen step = %v, want advance+tracking %v", got, want)
	}
}

func TestLineGapAscenderDescenderInvariant(t *testing.T) {
	fs := testFontsOrSkip(t)
	cases := []string{
		"gjpqy\nhkbdflt",
		"py\nbh\n gq \nlt",
		demoParagraph(),
	}
	for ci, text := range cases {
		for _, ppem := range []float64{24, 44, 80} {
			sty := LayoutStyle{
				Ppem: ppem, TrackingEm: 0.04, LineHeightEm: 1.05,
				MinInkGapEm: 0.18, MaxWidth: 900,
			}
			lay := fs.Layout(text, sty)
			if lay.Overlap {
				t.Fatalf("case %d ppem %v: layout reports overlap", ci, ppem)
			}
			need := sty.MinInkGapEm * ppem
			for li, gap := range lay.PairGaps {
				if gap < need-1e-6 {
					t.Fatalf("case %d ppem %v pair %d gap %.3f < min %.3f",
						ci, ppem, li, gap, need)
				}
			}
		}
	}
}

func TestWrappingKeepsWidth(t *testing.T) {
	fs := testFontsOrSkip(t)
	sty := LayoutStyle{Ppem: 28, TrackingEm: 0.03, MaxWidth: 520, LineHeightEm: 1.6}
	lay := fs.Layout(demoParagraph(), sty)
	if len(lay.Lines) < 3 {
		t.Fatalf("expected wrapped multi-line layout, got %d lines", len(lay.Lines))
	}
	for _, ln := range lay.Lines {
		if ln.Width > sty.MaxWidth+1 {
			t.Fatalf("line width %.1f exceeds max %.1f", ln.Width, sty.MaxWidth)
		}
	}
}

func TestCounterGuardPreservesCounters(t *testing.T) {
	fs := testFontsOrSkip(t)
	for _, r := range []rune{'o', 'a', '国', '量', 'B'} {
		rep := fs.BoldReport(r)
		if !rep.HasOutline || len(rep.AtChosen) == 0 {
			continue
		}
		if rep.ChosenEm <= 0 {
			t.Fatalf("%c: chosen bold radius is zero", r)
		}
		for i, st := range rep.AtChosen {
			if st.Retention < 0.38 {
				t.Fatalf("%c counter %d retention %.2f below threshold at chosen r=%.3f",
					r, i, st.Retention, rep.ChosenEm)
			}
		}
	}
	// The deliberately excessive radius must be detected as destructive
	// for dense Han glyphs.
	_, stats, _, _, _, _, _ := fs.EvaluateRadiusEm('量', 0.11)
	closed := 0
	for _, st := range stats {
		if st.AreaBold < 28 {
			closed++
		}
	}
	if closed == 0 {
		t.Fatalf("expected r=0.11em to swallow at least one 量 counter")
	}
}

func TestEndToEndRun(t *testing.T) {
	dir := t.TempDir()
	if _, err := LoadFontSet(); err != nil {
		t.Skipf("fonts unavailable: %v", err)
	}
	if err := run(dir); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{
		"01_original_outlines.png", "03_layout_result.png",
		"06_zoom_linegap.png", "08_zoom_counters.png",
		"report.json", "outline_data.json",
	} {
		p := filepath.Join(dir, name)
		st, err := os.Stat(p)
		if err != nil {
			t.Fatalf("missing artifact %s: %v", name, err)
		}
		if st.Size() < 1024 {
			t.Fatalf("artifact %s too small: %d bytes", name, st.Size())
		}
	}
}
