package geom

import (
	"math"
	"testing"
)

func approx(a, b, eps float64) bool { return math.Abs(a-b) <= eps }

func TestPixelCoverageHalfRect(t *testing.T) {
	// Rectangle x in [2,2.5], y in [3,4]: pixel (2,3) is half covered,
	// no other pixel may receive any coverage.
	poly := []Vec{{2, 3}, {2.5, 3}, {2.5, 4}, {2, 4}}
	if c := PixelCoverage(poly, 2, 3); !approx(c, 0.5, 1e-12) {
		t.Fatalf("half coverage = %v, want 0.5", c)
	}
	for _, ix := range []int{1, 3} {
		if c := PixelCoverage(poly, ix, 3); c != 0 {
			t.Fatalf("coverage outside rect x at ix=%d = %v", ix, c)
		}
	}
	for _, iy := range []int{2, 4} {
		if c := PixelCoverage(poly, 2, iy); c != 0 {
			t.Fatalf("coverage outside rect y at iy=%d = %v", iy, c)
		}
	}
}

func TestCoverageConservesArea(t *testing.T) {
	// Rotated thin rectangle (a stroke outline): summed pixel coverage
	// must equal the polygon area to floating point precision.
	poly := SegmentOutline(Vec{3.2, 7.1}, Vec{21.7, 12.9}, 0.7, 0.35)
	bb := PolyBBox(poly)
	sum := 0.0
	for iy := int(math.Floor(bb.MinY)); iy <= int(math.Floor(bb.MaxY)); iy++ {
		for ix := int(math.Floor(bb.MinX)); ix <= int(math.Floor(bb.MaxX)); ix++ {
			sum += PixelCoverage(poly, ix, iy)
		}
	}
	want := PolygonArea(poly)
	if !approx(sum, want, 1e-9) {
		t.Fatalf("coverage sum %.12f != polygon area %.12f (diff %g)", sum, want, sum-want)
	}
}

func TestCoverageInsideUnitRange(t *testing.T) {
	poly := SegmentOutline(Vec{1.3, 1.7}, Vec{9.8, 6.2}, 0.9, 0.45)
	for iy := 0; iy < 10; iy++ {
		for ix := 0; ix < 12; ix++ {
			c := PixelCoverage(poly, ix, iy)
			if c < 0 || c > 1 {
				t.Fatalf("coverage out of range at (%d,%d): %v", ix, iy, c)
			}
		}
	}
}

func TestPointInPoly(t *testing.T) {
	sq := []Vec{{0, 0}, {2, 0}, {2, 2}, {0, 2}}
	if !PointInPoly(Vec{1, 1}, sq) {
		t.Fatal("center should be inside")
	}
	if PointInPoly(Vec{3, 1}, sq) {
		t.Fatal("outside point reported inside")
	}
}
