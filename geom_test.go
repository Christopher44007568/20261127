package main

import "testing"

func TestRectIntersectUnion(t *testing.T) {
	a := RectWH(0, 0, 10, 10)
	b := RectWH(5, 5, 10, 10)
	if got := a.Intersect(b); got != (Rect{5, 5, 10, 10}) {
		t.Fatalf("intersection = %+v", got)
	}
	if got := a.Union(b); got != (Rect{0, 0, 15, 15}) {
		t.Fatalf("union = %+v", got)
	}
	if a.Intersect(RectWH(20, 20, 2, 2)).Empty() != true {
		t.Fatal("disjoint rects must intersect to empty")
	}
	if got := (Rect{}).Union(b); got != b {
		t.Fatalf("empty union should be b, got %+v", got)
	}
}

func TestMergeRectsDisjointAndCovering(t *testing.T) {
	in := []Rect{
		RectWH(0, 0, 10, 10),
		RectWH(8, 8, 10, 10),   // overlaps first
		RectWH(100, 100, 4, 4), // separate
		RectWH(102, 102, 4, 4), // overlaps the small one
		{},
	}
	merged := MergeRects(in)
	if !RectsDisjoint(merged) {
		t.Fatalf("merged rectangles overlap: %+v", merged)
	}
	if len(merged) != 2 {
		t.Fatalf("expected 2 merged rects, got %d: %+v", len(merged), merged)
	}

	// Every input pixel must still be covered by some merged rectangle.
	for _, r := range in {
		if r.Empty() {
			continue
		}
		for y := r.Y0; y < r.Y1; y++ {
			for x := r.X0; x < r.X1; x++ {
				covered := false
				for _, m := range merged {
					if m.Contains(x, y) {
						covered = true
						break
					}
				}
				if !covered {
					t.Fatalf("pixel (%d,%d) from %+v not covered by merged region", x, y, r)
				}
			}
		}
	}
}
