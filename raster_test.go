package bezierstroke

import (
	"image/color"
	"testing"
)

func TestCoverageInteriorAndHalfPixelEdge(t *testing.T) {
	poly := Polygon{
		{X: 10.5, Y: 0},
		{X: 30, Y: 0},
		{X: 30, Y: 20},
		{X: 10.5, Y: 20},
	}
	cov := Coverage(40, 24, 8, []Polygon{poly})
	if got := cov[10*40+20]; got != 64 {
		t.Fatalf("interior coverage = %d, want 64", got)
	}
	if got := cov[10*40+10]; got != 32 {
		t.Fatalf("half-pixel edge coverage = %d, want 32", got)
	}
	if got := cov[10*40+9]; got != 0 {
		t.Fatalf("outside coverage = %d, want 0", got)
	}
}

func TestCoverageUnionsOverlappingPolygons(t *testing.T) {
	a := Polygon{{X: 10, Y: 10}, {X: 60, Y: 10}, {X: 60, Y: 60}, {X: 10, Y: 60}}
	b := Polygon{{X: 40, Y: 40}, {X: 90, Y: 40}, {X: 90, Y: 90}, {X: 40, Y: 90}}
	cov := Coverage(100, 100, 8, []Polygon{a, b})
	if got := cov[50*100+50]; got != 64 {
		t.Fatalf("overlap coverage = %d, want 64", got)
	}
}

func TestCanvasCoverageComposite(t *testing.T) {
	c := NewCanvas(2, 2, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	c.FillCoverage([]uint16{243, 0, 0, 0}, 16, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	if got := c.Pix.Pix[0]; got < 240 || got > 242 {
		t.Fatalf("coverage composite red = %d, want about 242", got)
	}
	if got := c.Pix.Pix[4]; got != 0 {
		t.Fatalf("zero coverage changed pixel to %d", got)
	}
}
