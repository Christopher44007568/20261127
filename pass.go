package main

// DrawCmd is one screen-space triangle with its pipeline state.
type DrawCmd struct {
	V0, V1, V2 Vertex
	Color      pixel

	DepthTest  bool
	DepthWrite bool

	StencilTest  bool  // draw only where stencil == StencilRef
	StencilRef   uint8 // reference / replacement value
	StencilWrite bool  // replace stencil on covered samples, skip color write
}

func (d DrawCmd) bounds() Rect {
	minX := int(floor32(min3(d.V0.X, d.V1.X, d.V2.X)))
	minY := int(floor32(min3(d.V0.Y, d.V1.Y, d.V2.Y)))
	maxX := int(ceil32(max3(d.V0.X, d.V1.X, d.V2.X))) + 1
	maxY := int(ceil32(max3(d.V0.Y, d.V1.Y, d.V2.Y))) + 1
	return Rect{minX, minY, maxX, maxY}
}

// Pass is one render pass: a set of draws plus per-attachment load/store
// requests. The planner may refine these into tile-local decisions.
type Pass struct {
	Load    [numAttachments]LoadOp
	Store   [numAttachments]StoreOp
	Scissor Rect // window clip in pixels; zero value means "full frame"
	Draws   []DrawCmd
}

func (p Pass) scissor(fb *Framebuffer) Rect {
	if p.Scissor.Empty() {
		return Rect{0, 0, fb.W, fb.H}
	}
	return p.Scissor.Intersect(Rect{0, 0, fb.W, fb.H})
}

// tileDecision is the resolved per-tile, per-attachment load/store choice.
type tileDecision struct {
	load  [numAttachments]LoadOp
	store [numAttachments]StoreOp
}

// Policy turns a pass's global requests into tile-local decisions.
// tilesTouched[p] is the set of tiles any draw of pass p may write.
type Policy func(fb *Framebuffer, passes []Pass, touched [][]bool) [][]tileDecision

// touchedTiles computes, per pass, which tiles each draw may write,
// conservatively: draw bounding box ∩ scissor ∩ tile bounds.
func touchedTiles(fb *Framebuffer, passes []Pass) [][]bool {
	out := make([][]bool, len(passes))
	for pi, p := range passes {
		set := make([]bool, len(fb.tiles))
		sc := p.scissor(fb)
		for _, d := range p.Draws {
			db := d.bounds().Intersect(sc)
			if db.Empty() {
				continue
			}
			for ti := range fb.tiles {
				if !fb.tiles[ti].Bounds.Intersect(db).Empty() {
					set[ti] = true
				}
			}
		}
		out[pi] = set
	}
	return out
}

// ReferencePolicy stores exactly what each pass requests, for every tile.
// It is the ground truth used to validate optimized policies.
func ReferencePolicy(fb *Framebuffer, passes []Pass, touched [][]bool) [][]tileDecision {
	out := make([][]tileDecision, len(passes))
	for pi, p := range passes {
		out[pi] = make([]tileDecision, len(fb.tiles))
		for ti := range out[pi] {
			out[pi][ti].load = p.Load
			out[pi][ti].store = p.Store
		}
	}
	return out
}

// TileLocalPolicy implements the memoryless-tile optimization:
//   - store is emitted only for tiles whose contents changed this pass
//     (drawn into, or cleared: a clear is a write), and only when a later
//     pass loads that attachment or the attachment is presented (color in
//     the final pass);
//   - load is dropped to DontCare for tiles the pass never reads outside its
//     own writes (i.e. untouched tiles that also aren't stored later need no
//     load), and untouched tiles keep Load only if a later pass needs the
//     data carried through.
//
// Crucially, a tile partially covered by geometry is still "touched" and
// keeps its store: dropping it is exactly the bug that turns edges black.
func TileLocalPolicy(fb *Framebuffer, passes []Pass, touched [][]bool) [][]tileDecision {
	out := make([][]tileDecision, len(passes))
	// neededAfter[pi][ti][a]: some later pass loads attachment a for tile ti,
	// or (color only) this is the final presented pass.
	needed := make([][][numAttachments]bool, len(passes))
	for pi := range passes {
		needed[pi] = make([][numAttachments]bool, len(fb.tiles))
		for ti := range fb.tiles {
			for a := Attachment(0); a < numAttachments; a++ {
				for qi := pi + 1; qi < len(passes); qi++ {
					if passes[qi].Load[a] == LoadLoad {
						needed[pi][ti][a] = true
					}
				}
				if a == AttColor && pi == len(passes)-1 {
					needed[pi][ti][a] = true // presented
				}
			}
		}
	}
	for pi, p := range passes {
		out[pi] = make([]tileDecision, len(fb.tiles))
		for ti := range fb.tiles {
			var d tileDecision
			for a := Attachment(0); a < numAttachments; a++ {
				d.load[a] = p.Load[a]
				// A clear rewrites the whole tile, so it counts as a write
				// exactly like a draw call does.
				written := touched[pi][ti] || p.Load[a] == LoadClear
				// Tile-local store: only tiles this pass may write, and only
				// when the contents are consumed later or presented.
				if p.Store[a] == StoreStore && written && needed[pi][ti][a] {
					d.store[a] = StoreStore
				} else {
					d.store[a] = StoreDontCare
				}
				// Tile-local load: an untouched tile whose data is never
				// stored has no consumers, so the load is dead.
				if !touched[pi][ti] && d.store[a] == StoreDontCare {
					d.load[a] = LoadDontCare
				}
			}
			out[pi][ti] = d
		}
	}
	return out
}

// FullCoveragePolicy is the deliberately naive policy used to demonstrate
// the bug: it stores only tiles fully covered by a single draw's bounding
// box and discards the rest. Partially covered edge tiles lose their store,
// so the surface keeps stale (black) data at silhouette edges.
func FullCoveragePolicy(fb *Framebuffer, passes []Pass, touched [][]bool) [][]tileDecision {
	out := ReferencePolicy(fb, passes, touched)
	for pi, p := range passes {
		sc := p.scissor(fb)
		for ti := range fb.tiles {
			tb := fb.tiles[ti].Bounds
			full := false
			for _, d := range p.Draws {
				db := d.bounds().Intersect(sc)
				if db.MinX <= tb.MinX && db.MinY <= tb.MinY &&
					db.MaxX >= tb.MaxX && db.MaxY >= tb.MaxY {
					full = true
					break
				}
			}
			if !full {
				for a := Attachment(0); a < numAttachments; a++ {
					out[pi][ti].store[a] = StoreDontCare
				}
			}
		}
	}
	return out
}
