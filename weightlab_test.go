package main

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestPointSegmentDistance(t *testing.T) {
	got, _ := PointSegmentDistance(Vec2{X: 5, Y: 3}, Vec2{X: 0, Y: 0}, Vec2{X: 10, Y: 0})
	if math.Abs(got-3) > 1e-12 {
		t.Fatalf("distance = %v, want 3", got)
	}

	d, projection := PointSegmentDistance(Vec2{X: -4, Y: 3}, Vec2{X: 0, Y: 0}, Vec2{X: 10, Y: 0})
	if math.Abs(projection) > 1e-12 || math.Abs(d-5) > 1e-12 {
		t.Fatalf("endpoint distance/projection = %v/%v", d, projection)
	}
}

func TestSegmentIntersection(t *testing.T) {
	hit, ok := SegmentIntersection(
		Vec2{X: 0, Y: 0}, Vec2{X: 10, Y: 10},
		Vec2{X: 0, Y: 10}, Vec2{X: 10, Y: 0},
	)
	if !ok {
		t.Fatal("expected crossing")
	}
	if math.Abs(hit.Point.X-5) > 1e-9 || math.Abs(hit.Point.Y-5) > 1e-9 {
		t.Fatalf("crossing point = %+v, want (5,5)", hit.Point)
	}

	if _, ok := SegmentIntersection(
		Vec2{X: 0, Y: 0}, Vec2{X: 1, Y: 0},
		Vec2{X: 2, Y: 0}, Vec2{X: 3, Y: 0},
	); ok {
		t.Fatal("disjoint collinear segments must not intersect")
	}
}

func TestFindsBowtieIntersection(t *testing.T) {
	bowtie := Contour{
		{X: 0, Y: 0},
		{X: 10, Y: 10},
		{X: 0, Y: 10},
		{X: 10, Y: 0},
	}
	hits := FindContourIntersections(bowtie)
	if len(hits) != 1 {
		t.Fatalf("hits = %d, want 1: %+v", len(hits), hits)
	}
	if math.Abs(hits[0].Point.X-5) > 1e-8 || math.Abs(hits[0].Point.Y-5) > 1e-8 {
		t.Fatalf("highlight = %+v, want center (5,5)", hits[0].Point)
	}
}

func TestSampleGlyphOrientationAndTopology(t *testing.T) {
	glyph := SampleGlyph()
	if glyph.Outer.SignedArea() <= 0 {
		t.Fatal("outer contour must be CCW after canonicalization")
	}
	if len(glyph.Holes) != 1 || glyph.Holes[0].SignedArea() >= 0 {
		t.Fatal("counter must exist and be CW after canonicalization")
	}
	if hits := FindOutlineIntersections(glyph); len(hits) != 0 {
		t.Fatalf("source glyph has intersections: %+v", hits)
	}
	if area := glyph.CounterArea(); area < 14000 || area > 16000 {
		t.Fatalf("counter area = %.2f", area)
	}
}

func TestExpansionPreservesTopologyForVariableWeights(t *testing.T) {
	glyph := SampleGlyph()
	previousCounter := math.Inf(1)
	for _, weight := range []int{400, 500, 600, 700, 800, 900} {
		delta := WeightToDelta(weight)
		result := ExpandOutline(glyph, delta)
		if !result.Valid {
			t.Fatalf("weight %d expansion invalid: %+v", weight, result.Intersections)
		}
		if result.Delta+1e-9 < delta {
			t.Fatalf("weight %d was reduced from %.3f to %.3f", weight, delta, result.Delta)
		}
		if hits := FindOutlineIntersections(result.Outline); len(hits) != 0 {
			t.Fatalf("weight %d has %d intersections", weight, len(hits))
		}
		topology := AnalyzeTopology(glyph, result.Outline)
		if !topology.Valid || topology.ContourCount != 2 {
			t.Fatalf("weight %d topology changed: %+v", weight, topology)
		}
		counter := result.Outline.CounterArea()
		if counter <= 0 || counter >= previousCounter+1e-6 {
			t.Fatalf("weight %d counter area = %.3f, previous = %.3f", weight, counter, previousCounter)
		}
		previousCounter = counter
	}
}

func TestStressNaiveCrossesButCorrectionDoesNot(t *testing.T) {
	glyph := SampleGlyph()
	const stress = 47.5
	naive := Outline{Outer: NaiveMiterContour(glyph.Outer, stress)}
	for _, hole := range glyph.Holes {
		naive.Holes = append(naive.Holes, NaiveMiterContour(hole, stress))
	}
	if hits := FindOutlineIntersections(naive); len(hits) == 0 {
		t.Fatal("expected unlimited naive miter stress test to self-intersect")
	}

	corrected := ExpandOutline(glyph, stress)
	if !corrected.Valid || len(corrected.Intersections) != 0 {
		t.Fatalf("corrected stress expansion invalid: %+v", corrected.Intersections)
	}
	if !corrected.Reduced || corrected.Delta >= stress {
		t.Fatalf("expected unsafe delta reduction, applied %.4f/%v", corrected.Delta, corrected.Reduced)
	}
	if corrected.Outline.CounterArea() <= 100 {
		t.Fatalf("counter should remain visibly open, area %.2f", corrected.Outline.CounterArea())
	}
}

func TestFontSizeCounterAreaIsContinuousQuadratic(t *testing.T) {
	glyph := SampleGlyph()
	first := MeasureFontSize(glyph, 900, 12, DefaultExpandOptions())
	for size := 12.0; size <= 128.0; size += 0.5 {
		metric := MeasureFontSize(glyph, 900, size, DefaultExpandOptions())
		if !metric.TopologyValid || metric.CounterAreaPixels <= 0 {
			t.Fatalf("size %.1f invalid metric: %+v", size, metric)
		}
		want := first.CounterAreaUnits * (size / UnitsPerEm) * (size / UnitsPerEm)
		if math.Abs(metric.CounterAreaPixels-want) > 1e-7 {
			t.Fatalf("size %.1f area = %.9f, want quadratic %.9f", size, metric.CounterAreaPixels, want)
		}
		if math.Abs(metric.NormalizedCounterArea-first.NormalizedCounterArea) > 1e-12 {
			t.Fatalf("normalized area changed with font size: %v", metric.NormalizedCounterArea)
		}
	}
}

func TestWeightAxisCounterAreaChangesWithoutJumps(t *testing.T) {
	glyph := SampleGlyph()
	previous := glyph.CounterArea()
	for delta := 0.0; delta <= 42.0; delta += 0.25 {
		result := ExpandOutline(glyph, delta)
		if !result.Valid || result.Delta+1e-8 < delta {
			t.Fatalf("delta %.2f was not safely expanded: %+v", delta, result)
		}
		area := result.Outline.CounterArea()
		if area > previous+1e-6 {
			t.Fatalf("counter area increased at delta %.2f: %.6f -> %.6f", delta, previous, area)
		}
		if previous-area > 220 {
			t.Fatalf("counter area jumped by %.3f at delta %.2f", previous-area, delta)
		}
		previous = area
	}
}

func TestGenerateArtifactsWritesRequiredFiles(t *testing.T) {
	report, err := GenerateArtifacts(t.TempDir(), 900, 2)
	if err != nil {
		t.Fatal(err)
	}
	if !report.Topology.Valid {
		t.Fatalf("generated topology invalid: %+v", report.Topology)
	}
	if len(report.NaiveIntersections) == 0 {
		t.Fatal("highlight report must contain at least one naive intersection")
	}
	for _, path := range report.Files {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("missing artifact %s: %v", path, err)
		}
		if info.Size() == 0 {
			t.Fatalf("empty artifact %s", path)
		}
		if filepath.Ext(path) == ".png" {
			// PNG signature gives a cheap format check without extra deps.
			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			header := make([]byte, 8)
			if _, err := f.Read(header); err != nil {
				f.Close()
				t.Fatal(err)
			}
			f.Close()
			if string(header[1:4]) != "PNG" {
				t.Fatalf("%s is not a PNG", path)
			}
		}
	}
}
