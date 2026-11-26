package main

import "fmt"

// buildScene creates a three-pass frame:
//
//	Pass 1: clear everything, draw two background triangles with depth.
//	Pass 2: load color+depth, clear stencil, rasterize a stencil mask.
//	Pass 3: load all three attachments, draw a triangle clipped by the
//	        stencil test, inside a scissor window.
//
// The passes exercise tile-local load/store for color, depth and stencil:
// dropping a store between passes (or at present) loses data exactly the
// way a mobile tiler does when a needed store is marked DontCare.
func buildScene(w, h int) []Pass {
	fw, fh := float32(w), float32(h)
	bg := []DrawCmd{
		{V0: Vertex{0.05 * fw, 0.90 * fh, 0.6}, V1: Vertex{0.05 * fw, 0.10 * fh, 0.6},
			V2: Vertex{0.75 * fw, 0.85 * fh, 0.6}, Color: rgba(0.15, 0.35, 0.75, 1),
			DepthTest: true, DepthWrite: true},
		{V0: Vertex{0.30 * fw, 0.15 * fh, 0.4}, V1: Vertex{0.95 * fw, 0.25 * fh, 0.4},
			V2: Vertex{0.60 * fw, 0.95 * fh, 0.4}, Color: rgba(0.85, 0.55, 0.15, 1),
			DepthTest: true, DepthWrite: true},
	}
	mask := []DrawCmd{
		{V0: Vertex{0.30 * fw, 0.30 * fh, 0.5}, V1: Vertex{0.70 * fw, 0.30 * fh, 0.5},
			V2: Vertex{0.50 * fw, 0.75 * fh, 0.5}, Color: rgba(0, 0, 0, 1),
			StencilWrite: true, StencilRef: 1},
	}
	fg := []DrawCmd{
		{V0: Vertex{0.20 * fw, 0.70 * fh, 0.3}, V1: Vertex{0.85 * fw, 0.60 * fh, 0.3},
			V2: Vertex{0.50 * fw, 0.20 * fh, 0.3}, Color: rgba(0.20, 0.85, 0.45, 1),
			DepthTest: true, DepthWrite: true, StencilTest: true, StencilRef: 1},
	}
	return []Pass{
		{
			Load:  [numAttachments]LoadOp{LoadClear, LoadClear, LoadClear},
			Store: [numAttachments]StoreOp{StoreStore, StoreStore, StoreDontCare},
			Draws: bg,
		},
		{
			Load:  [numAttachments]LoadOp{LoadLoad, LoadLoad, LoadClear},
			Store: [numAttachments]StoreOp{StoreStore, StoreDontCare, StoreStore},
			Draws: mask,
		},
		{
			Load:    [numAttachments]LoadOp{LoadLoad, LoadLoad, LoadLoad},
			Store:   [numAttachments]StoreOp{StoreStore, StoreDontCare, StoreDontCare},
			Scissor: Rect{w / 8, h / 8, 7 * w / 8, 7 * h / 8},
			Draws:   fg,
		},
	}
}

func main() {
	const (
		W, H    = 320, 240
		TilePx  = 16
		Samples = 4
	)
	passes := buildScene(W, H)

	newFB := func() *Framebuffer {
		fb := NewFramebuffer(W, H, TilePx, Samples)
		fb.ClearColor = rgba(0.10, 0.10, 0.14, 1)
		return fb
	}

	// Ground truth: every requested store happens.
	ref := newFB().Run(passes, ReferencePolicy)
	must(writePNG("out/reference.png", ref))

	// Correct tile-local policy: dead stores/loads elided, edges preserved.
	fbOpt := newFB()
	opt := fbOpt.Run(passes, TileLocalPolicy)
	must(writePNG("out/frame.png", opt))
	touched := touchedTiles(fbOpt, passes)
	must(tileMapPNG("out/tiles.png", fbOpt, TileLocalPolicy(fbOpt, passes, touched)))
	badOpt := countBadTiles(fbOpt, ref, opt)
	fmt.Printf("tile-local policy: %d bad tiles vs reference (must be 0)\n", badOpt)

	// Buggy policy: stores only fully covered tiles; edge tiles are
	// discarded, so the composited frame goes black along silhouettes.
	fbBug := newFB()
	bug := fbBug.Run(passes, FullCoveragePolicy)
	must(writePNG("out/discarded.png", bug))
	badPx, err := errorMapPNG("out/errors.png", fbBug, ref, bug)
	must(err)
	fmt.Printf("full-coverage policy: %d bad pixels in %d bad tiles (see out/errors.png)\n",
		badPx, countBadTiles(fbBug, ref, bug))

	if badOpt != 0 {
		fmt.Println("FAIL: tile-local policy diverged from reference")
	} else {
		fmt.Println("OK: tile-local policy matches reference; edges preserved")
	}
}
