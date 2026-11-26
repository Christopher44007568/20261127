package bezierstroke

import "testing"

func TestRoundJoinFillsOuterCorner(t *testing.T) {
	mesh := StrokePolyline(
		[]Point{{X: 20, Y: 100}, {X: 100, Y: 100}, {X: 100, Y: 180}},
		StrokeStyle{Width: 20, Join: JoinRound, Cap: CapButt, Tolerance: 0.05},
	)
	if len(mesh.Joins) != 1 {
		t.Fatalf("got %d joins, want 1", len(mesh.Joins))
	}

	// (105,95) is outside both constant-width strips but inside the outer
	// round-join sector at the right-angle corner.
	cov := Coverage(200, 200, 8, mesh.Polygons)
	if got := cov[95*200+105]; got == 0 {
		t.Fatalf("outer join gap at (105,95)")
	}
}
