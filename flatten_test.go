package bezierstroke

import (
	"math"
	"testing"
)

func TestChordHeightKnownCurve(t *testing.T) {
	c := Cubic{
		P0: Point{X: 0, Y: 0},
		P1: Point{X: 1, Y: 1},
		P2: Point{X: 2, Y: 1},
		P3: Point{X: 3, Y: 0},
	}
	if got := c.ChordHeight(); math.Abs(got-0.75) > 1e-12 {
		t.Fatalf("ChordHeight = %v, want 0.75", got)
	}
}

func TestChordHeightMatchesDenseScan(t *testing.T) {
	curves := []Cubic{
		{P0: Point{0, 0}, P1: Point{1, 3}, P2: Point{4, -2}, P3: Point{5, 1}},
		{P0: Point{-2, 1}, P1: Point{3, -4}, P2: Point{-1, 5}, P3: Point{2, 0}},
		{P0: Point{0, 0}, P1: Point{0.1, 0.2}, P2: Point{9.9, -0.1}, P3: Point{10, 0}},
	}
	for i, c := range curves {
		got := c.ChordHeight()
		want := denseChordHeight(c, 20000)
		if math.Abs(got-want) > 1e-5 {
			t.Fatalf("curve %d: analytic height %v, dense %v", i, got, want)
		}
	}
}

func TestAdaptiveFlattenHonorsTolerance(t *testing.T) {
	c := Cubic{
		P0: Point{X: -2, Y: 1},
		P1: Point{X: 3, Y: -4},
		P2: Point{X: -1, Y: 5},
		P3: Point{X: 2, Y: 0},
	}
	tol := 0.4
	vertices := c.AdaptiveFlattenDetailed(tol)
	if len(vertices) < 3 {
		t.Fatalf("expected adaptive subdivision, got %d vertices", len(vertices))
	}
	if vertices[0].T != 0 || vertices[len(vertices)-1].T != 1 {
		t.Fatalf("parameters do not span [0,1]")
	}
	for i := 0; i+1 < len(vertices); i++ {
		if vertices[i].T >= vertices[i+1].T {
			t.Fatalf("parameters not ordered at %d", i)
		}
		seg := c.Segment(vertices[i].T, vertices[i+1].T)
		if err := seg.ChordHeight(); err > tol+1e-9 {
			t.Fatalf("segment %d height %.12g exceeds tolerance %.12g", i, err, tol)
		}
		if got := seg.P0.Distance(vertices[i].Point); got > 1e-10 {
			t.Fatalf("segment %d start drift %.12g", i, got)
		}
		if got := seg.P3.Distance(vertices[i+1].Point); got > 1e-10 {
			t.Fatalf("segment %d end drift %.12g", i, got)
		}
	}
}

func TestNearLineFlattenGetsFinerWithZoom(t *testing.T) {
	c := Cubic{
		P0: Point{X: 0, Y: 0},
		P1: Point{X: 1.2, Y: 0.001},
		P2: Point{X: 2.4, Y: -0.001},
		P3: Point{X: 3.6, Y: 0},
	}
	coarse := c.AdaptiveFlatten(1.0)
	fine := c.AdaptiveFlatten(0.0001)
	if len(coarse) != 2 {
		t.Fatalf("near-line coarse pass should be one chord, got %d points", len(coarse))
	}
	if len(fine) <= len(coarse) {
		t.Fatalf("zoom tolerance did not refine: %d <= %d", len(fine), len(coarse))
	}
}

func TestFoldedZeroChordStillSubdivides(t *testing.T) {
	c := Cubic{
		P0: Point{X: 0, Y: 0},
		P1: Point{X: 2, Y: 0},
		P2: Point{X: -2, Y: 0},
		P3: Point{X: 0, Y: 0},
	}
	vertices := c.AdaptiveFlattenDetailed(0.5)
	if len(vertices) < 3 {
		t.Fatalf("zero-chord fold was accepted as flat: %d vertices", len(vertices))
	}
}

func TestLongCurveKeepsExactEndpoints(t *testing.T) {
	c := Cubic{
		P0: Point{X: 0, Y: 0},
		P1: Point{X: 12000, Y: 300},
		P2: Point{X: 24000, Y: -300},
		P3: Point{X: 36000, Y: 0},
	}
	vertices := c.AdaptiveFlattenDetailed(1.0)
	if len(vertices) < 2 || len(vertices) > 10000 {
		t.Fatalf("unexpected vertex count %d", len(vertices))
	}
	if vertices[0].Point.Distance(c.P0) > 1e-12 {
		t.Fatalf("start endpoint drift %v", vertices[0].Point.Distance(c.P0))
	}
	if vertices[len(vertices)-1].Point.Distance(c.P3) > 1e-12 {
		t.Fatalf("end endpoint drift %v", vertices[len(vertices)-1].Point.Distance(c.P3))
	}
}

func denseChordHeight(c Cubic, n int) float64 {
	chord := c.P3.Sub(c.P0)
	length := chord.Len()
	maxDev := 0.0
	for i := 0; i <= n; i++ {
		p := c.Eval(float64(i) / float64(n))
		d := math.Abs(p.Sub(c.P0).Cross(chord)) / length
		if d > maxDev {
			maxDev = d
		}
	}
	return maxDev
}
