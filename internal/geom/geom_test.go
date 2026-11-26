package geom

import "testing"

func TestWindingNumberAndHole(t *testing.T) {
	outer := Contour{Points: []V{{0, 0}, {10, 0}, {10, 10}, {0, 10}}}
	hole := Contour{Points: []V{{4, 4}, {4, 6}, {6, 6}, {6, 4}}}
	shape := Shape{Contours: []Contour{outer, hole}}

	if !Inside(V{1, 1}, shape.Contours) {
		t.Fatal("outer corner should be inside")
	}
	if Inside(V{5, 5}, shape.Contours) {
		t.Fatal("oppositely wound hole should be outside under non-zero fill")
	}
	if Inside(V{-1, 5}, shape.Contours) {
		t.Fatal("point outside box should be outside")
	}
}

func TestExtractBoundaryRemovesOverlapSeams(t *testing.T) {
	up := Contour{Points: []V{{0, 10}, {-8, -5}, {8, -5}}}
	down := Contour{Points: []V{{0, -10}, {8, 5}, {-8, 5}}}
	shape := Shape{Contours: []Contour{up, down}}

	boundary := ExtractBoundaryShape(shape, 0.01)
	if len(boundary.Contours) != 1 {
		t.Fatalf("expected one stitched union contour, got %d", len(boundary.Contours))
	}
	if len(boundary.Contours[0].Points) < 12 {
		t.Fatalf("expected six tips and six crossings, got %d points", len(boundary.Contours[0].Points))
	}
	if !Inside(V{0, 0}, shape.Contours) {
		t.Fatal("overlap center must remain filled by non-zero rule")
	}
}

func TestPointSegmentDistance(t *testing.T) {
	seg := Segment{A: V{0, 0}, B: V{10, 0}}
	if d, _ := PointSegment(V{5, 3}, seg); d < 2.999 || d > 3.001 {
		t.Fatalf("perpendicular distance = %v", d)
	}
	if d, _ := PointSegment(V{-4, 0}, seg); d < 3.999 || d > 4.001 {
		t.Fatalf("endpoint distance = %v", d)
	}
}
