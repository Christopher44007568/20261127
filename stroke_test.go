package bezierstroke

import (
	"math"
	"testing"
)

func TestSegmentQuadHasConstantWidth(t *testing.T) {
	mesh := StrokePolyline(
		[]Point{{X: 10, Y: 100}, {X: 190, Y: 100}},
		StrokeStyle{Width: 20, Cap: CapButt, Tolerance: 0.1},
	)
	if len(mesh.Segments) != 1 {
		t.Fatalf("got %d segments", len(mesh.Segments))
	}
	seg := mesh.Segments[0]
	if math.Abs(seg.LeftA.Y-110) > 1e-9 || math.Abs(seg.RightA.Y-90) > 1e-9 {
		t.Fatalf("wrong half-width corners: %+v", seg.Polygon)
	}
	if math.Abs(seg.LeftB.Y-110) > 1e-9 || math.Abs(seg.RightB.Y-90) > 1e-9 {
		t.Fatalf("wrong end half-width corners: %+v", seg.Polygon)
	}
}

func TestRoundCapStaysWithinRadius(t *testing.T) {
	mesh := StrokePolyline(
		[]Point{{X: 10, Y: 100}, {X: 190, Y: 100}},
		StrokeStyle{Width: 24, Cap: CapRound, Tolerance: 0.05},
	)
	if len(mesh.Caps) != 2 {
		t.Fatalf("got %d caps, want 2", len(mesh.Caps))
	}
	for _, cap := range mesh.Caps {
		for _, p := range cap.Polygon {
			d := p.Distance(cap.Center)
			if math.Abs(d-12) > 0.02 {
				t.Fatalf("cap point radius %.8f, want 12", d)
			}
		}
	}
}

func TestSharpCuspMeshIsClosedAndContinuous(t *testing.T) {
	c := Cubic{
		P0: Point{X: 0, Y: 0},
		P1: Point{X: 1, Y: 1},
		P2: Point{X: -1, Y: 1},
		P3: Point{X: 1, Y: 0},
	}
	screen := TransformCubic(c, func(p Point) Point {
		return Point{X: 100 + p.X*200, Y: 300 - p.Y*200}
	})
	vertices := screen.AdaptiveFlatten(0.2)
	mesh := StrokePolyline(vertices, StrokeStyle{
		Width: 20, Join: JoinRound, Cap: CapRound, Tolerance: 0.05,
	})
	if len(mesh.Joins) < 10 {
		t.Fatalf("cusp mesh has too few joins: %d", len(mesh.Joins))
	}
	if len(mesh.Caps) != 2 {
		t.Fatalf("cusp mesh has %d caps, want 2", len(mesh.Caps))
	}
	for i, poly := range mesh.Polygons {
		if len(poly) < 3 {
			t.Fatalf("polygon %d has %d vertices", i, len(poly))
		}
		if signedArea(poly) <= 0 {
			t.Fatalf("polygon %d winding is not positive", i)
		}
	}

	cov := Coverage(500, 400, 8, mesh.Polygons)
	for _, v := range vertices {
		p := v
		x, y := int(p.X), int(p.Y)
		if x < 0 || x >= 500 || y < 0 || y >= 400 {
			t.Fatalf("test point outside canvas: %+v", p)
		}
		if cov[y*500+x] == 0 {
			t.Fatalf("centerline has a coverage hole at %+v", p)
		}
	}
}

func TestMiterAndBevelJoinsProducePositivePolygons(t *testing.T) {
	points := []Point{{X: 20, Y: 100}, {X: 100, Y: 100}, {X: 40, Y: 20}}
	for _, join := range []JoinStyle{JoinRound, JoinMiter, JoinBevel} {
		mesh := StrokePolyline(points, StrokeStyle{
			Width: 24, Join: join, Cap: CapButt, Tolerance: 0.1, MiterLimit: 5,
		})
		if len(mesh.Joins) != 1 {
			t.Fatalf("join style %d: got %d joins", join, len(mesh.Joins))
		}
		if signedArea(mesh.Joins[0].Polygon) <= 0 {
			t.Fatalf("join style %d: bad winding", join)
		}
	}
}
