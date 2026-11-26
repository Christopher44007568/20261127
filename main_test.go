package main

import "testing"

const (
	testW, testH = 160, 120
	testTile     = 16
	testSamples  = 4
)

func newTestFB() *Framebuffer {
	fb := NewFramebuffer(testW, testH, testTile, testSamples)
	fb.ClearColor = rgba(0.10, 0.10, 0.14, 1)
	return fb
}

// The optimized tile-local policy must be pixel-identical to the reference.
func TestTileLocalMatchesReference(t *testing.T) {
	passes := buildScene(testW, testH)
	ref := newTestFB().Run(passes, ReferencePolicy)
	fb := newTestFB()
	got := fb.Run(passes, TileLocalPolicy)
	if n := countBadTiles(fb, ref, got); n != 0 {
		t.Fatalf("tile-local policy diverged in %d tiles", n)
	}
}

// Dropping stores for partially covered tiles must corrupt the surface:
// this documents the black-edge bug the planner prevents.
func TestFullCoveragePolicyCorruptsEdges(t *testing.T) {
	passes := buildScene(testW, testH)
	ref := newTestFB().Run(passes, ReferencePolicy)
	fb := newTestFB()
	got := fb.Run(passes, FullCoveragePolicy)
	if n := countBadTiles(fb, ref, got); n == 0 {
		t.Fatal("expected full-coverage policy to corrupt tiles")
	}
}

// MSAA resolve must blend samples: a pixel straddling a triangle edge ends
// up between the clear color and the triangle color, never a raw sample.
func TestResolveBlendsEdgePixels(t *testing.T) {
	fb := NewFramebuffer(32, 32, 16, 4)
	fb.ClearColor = rgba(0, 0, 0, 1)
	// Vertical edge straight through pixel column 16 (x=16.5 boundary).
	tri := DrawCmd{
		V0: Vertex{16.5, -10, 0.5}, V1: Vertex{16.5, 42, 0.5}, V2: Vertex{42, 16, 0.5},
		Color: rgba(1, 1, 1, 1), DepthTest: true, DepthWrite: true,
	}
	pass := Pass{
		Load:  [numAttachments]LoadOp{LoadClear, LoadClear, LoadClear},
		Store: [numAttachments]StoreOp{StoreStore, StoreDontCare, StoreDontCare},
		Draws: []DrawCmd{tri},
	}
	surf := fb.Run([]Pass{pass}, ReferencePolicy)
	// Pixel (16,16) straddles the edge: 2 of 4 samples covered -> 0.5.
	got := surf.color[16*32+16]
	if got.R < 0.4 || got.R > 0.6 {
		t.Fatalf("edge pixel not blended: got %v, want ~0.5", got.R)
	}
	// A fully covered pixel is exactly the triangle color.
	if inner := surf.color[16*32+24]; inner.R != 1 {
		t.Fatalf("interior pixel wrong: got %v, want 1", inner.R)
	}
	// A fully outside pixel is exactly the clear color.
	if outer := surf.color[16*32+8]; outer.R != 0 {
		t.Fatalf("outside pixel wrong: got %v, want 0", outer.R)
	}
}

// Scissor clipping must leave pixels outside the window untouched while
// keeping the window boundary sharp and resolved.
func TestScissorPreservesEdges(t *testing.T) {
	fb := NewFramebuffer(64, 64, 16, 4)
	fb.ClearColor = rgba(0, 0, 0, 1)
	tri := DrawCmd{
		V0: Vertex{0, 0, 0.5}, V1: Vertex{64, 0, 0.5}, V2: Vertex{0, 64, 0.5},
		Color: rgba(1, 0, 0, 1), DepthTest: true, DepthWrite: true,
	}
	sc := Rect{16, 16, 48, 48}
	pass := Pass{
		Load:    [numAttachments]LoadOp{LoadClear, LoadClear, LoadClear},
		Store:   [numAttachments]StoreOp{StoreStore, StoreDontCare, StoreDontCare},
		Scissor: sc,
		Draws:   []DrawCmd{tri},
	}
	surf := fb.Run([]Pass{pass}, ReferencePolicy)
	// Inside the window and inside the triangle: red.
	if p := surf.color[20*64+20]; p.R != 1 {
		t.Fatalf("inside scissor not drawn: %v", p)
	}
	// Outside the window: clear color, even though the triangle covers it.
	for _, pt := range [][2]int{{4, 20}, {20, 4}, {60, 20}, {20, 60}} {
		if p := surf.color[pt[1]*64+pt[0]]; p.R != 0 {
			t.Fatalf("scissor leak at %v: %v", pt, p)
		}
	}
}

// Stencil written in one pass must gate draws in a later pass, which only
// works if the stencil store/load between passes is lossless.
func TestStencilRoundTripBetweenPasses(t *testing.T) {
	fb := NewFramebuffer(32, 32, 16, 4)
	fb.ClearColor = rgba(0, 0, 0, 1)
	mask := DrawCmd{
		V0: Vertex{0, 0, 0.5}, V1: Vertex{16, 0, 0.5}, V2: Vertex{0, 32, 0.5},
		StencilWrite: true, StencilRef: 1,
	}
	draw := DrawCmd{
		V0: Vertex{0, 0, 0.5}, V1: Vertex{32, 0, 0.5}, V2: Vertex{0, 32, 0.5},
		Color: rgba(0, 1, 0, 1), StencilTest: true, StencilRef: 1,
	}
	passes := []Pass{
		{
			Load:  [numAttachments]LoadOp{LoadClear, LoadClear, LoadClear},
			Store: [numAttachments]StoreOp{StoreStore, StoreDontCare, StoreStore},
			Draws: []DrawCmd{mask},
		},
		{
			Load:  [numAttachments]LoadOp{LoadLoad, LoadDontCare, LoadLoad},
			Store: [numAttachments]StoreOp{StoreStore, StoreDontCare, StoreDontCare},
			Draws: []DrawCmd{draw},
		},
	}
	surf := fb.Run(passes, TileLocalPolicy)
	// (4,4) is inside the mask: green.
	if p := surf.color[4*32+4]; p.G != 1 {
		t.Fatalf("stencil-gated pixel missing: %v", p)
	}
	// (28,4) is outside the mask but inside the draw triangle: black.
	if p := surf.color[4*32+28]; p.G != 0 {
		t.Fatalf("stencil test failed to reject: %v", p)
	}
}
