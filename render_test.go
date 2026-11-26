package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNormalFrameIsIndependentOfPreviousFrame(t *testing.T) {
	normal := renderScenario("normal", PolicyCorrect, BugNone)
	reference := renderFreshFrame(1)

	if got := countPixelDiff(normal.Frames[1], reference); got != 0 {
		t.Fatalf("frame 1 differs from fresh render in %d pixels", got)
	}
	for i, diff := range normal.Diffs {
		if diff.ChangedPixels != 0 {
			t.Fatalf("frame %d differs from ideal in %d pixels", i, diff.ChangedPixels)
		}
	}
	if normal.Stats.FailedAcquires != 0 || normal.Stats.ReusedRetired != 0 {
		t.Fatalf("normal path used invalid resource state: %+v", normal.Stats)
	}
	if normal.Stats.Reclaimed != 2 {
		t.Fatalf("both frame textures should be reclaimed, got %d", normal.Stats.Reclaimed)
	}
}

func TestDiagnosticScenariosHaveDistinctFailures(t *testing.T) {
	normal := renderScenario("normal", PolicyCorrect, BugNone)
	early := renderScenario("early_release", PolicyEarly, BugEarlyRelease)
	late := renderScenario("delayed_reclaim", PolicyLate, BugLateReclaim)

	for i, frame := range normal.Frames {
		if got := countExactColor(frame, colMagenta); got != 0 {
			t.Fatalf("normal frame %d contains %d magenta pixels", i, got)
		}
	}

	earlyLast := early.Diffs[len(early.Diffs)-1]
	if early.Stats.FailedAcquires == 0 || earlyLast.BlackPixels == 0 || earlyLast.MagentaPixels == 0 {
		t.Fatalf("early-release evidence missing: stats=%+v diff=%+v", early.Stats, earlyLast)
	}
	if countExactColor(early.Frames[1], colBlack) == 0 {
		t.Fatal("early-release frame has no black blocks")
	}
	if countExactColor(early.Frames[1], colMagenta) == 0 {
		t.Fatal("early-release frame has no magenta highlight")
	}

	lateLast := late.Diffs[len(late.Diffs)-1]
	if late.Stats.ReusedRetired == 0 || lateLast.ChangedPixels == 0 || lateLast.MagentaPixels == 0 {
		t.Fatalf("late-reclaim evidence missing: stats=%+v diff=%+v", late.Stats, lateLast)
	}
	if late.QueueEnd == 0 {
		t.Fatal("late-reclaim resource should still be queued")
	}
	if late.Diffs[0].ChangedPixels != 0 {
		t.Fatal("frame 0 should be clean before the delayed texture is reused")
	}
}

func TestPPMHeaderAndPayload(t *testing.T) {
	s := NewSurface(3, 2)
	colors := []RGBA{
		rgb(255, 0, 0), rgb(0, 255, 0), rgb(0, 0, 255),
		rgb(10, 20, 30), rgb(40, 50, 60), rgb(70, 80, 90),
	}
	copy(s.Pix, colors)

	path := filepath.Join(t.TempDir(), "sample.ppm")
	if err := s.WritePPM(path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	header := "P6\n3 2\n255\n"
	if len(data) != len(header)+len(colors)*3 {
		t.Fatalf("unexpected PPM length: got %d want %d", len(data), len(header)+len(colors)*3)
	}
	if string(data[:len(header)]) != header {
		t.Fatalf("unexpected PPM header %q", data[:len(header)])
	}
	payload := data[len(header):]
	for i, c := range colors {
		if payload[i*3] != c.R || payload[i*3+1] != c.G || payload[i*3+2] != c.B {
			t.Fatalf("pixel %d mismatch: %v", i, payload[i*3:i*3+3])
		}
	}
}

func TestSheetsUseExpectedDimensions(t *testing.T) {
	normal := renderScenario("normal", PolicyCorrect, BugNone)
	reference := renderFreshFrame(1)
	sheet := buildSheet(normal, reference, true)
	if sheet.W != sheetW || sheet.H != sheetH {
		t.Fatalf("sheet dimensions: got %dx%d want %dx%d", sheet.W, sheet.H, sheetW, sheetH)
	}
	if got := countExactColor(sheet, colMagenta); got != 0 {
		t.Fatalf("normal sheet contains %d magenta fault pixels", got)
	}
}
