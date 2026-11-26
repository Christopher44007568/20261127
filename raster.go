package main

import "math"

// Vertex is a screen-space vertex; X/Y in pixels, Z in [0,1] for depth.
type Vertex struct {
	X, Y, Z float32
}

func floor32(x float32) float32 { return float32(math.Floor(float64(x))) }

func ceil32(x float32) float32 { return -floor32(-x) }

func min3(a, b, c float32) float32 {
	if b < a {
		a = b
	}
	if c < a {
		a = c
	}
	return a
}

func max3(a, b, c float32) float32 {
	if b > a {
		a = b
	}
	if c > a {
		a = c
	}
	return a
}

// sampleOffsets returns the 2x2 MSAA sample positions inside a pixel.
var sampleOffsets = [][2]float32{
	{0.25, 0.25}, {0.75, 0.25},
	{0.25, 0.75}, {0.75, 0.75},
}

// edge is the 2D edge function for edge (a,b) evaluated at (x,y).
func edge(ax, ay, bx, by, x, y float32) float32 {
	return (x-ax)*(by-ay) - (y-ay)*(bx-ax)
}

// beginPass initializes tile memory according to each tile's load decision.
func (fb *Framebuffer) beginPass(p *Pass, dec []tileDecision) {
	sc := p.scissor(fb)
	for ti := range fb.tiles {
		t := &fb.tiles[ti]
		t.touched = false
		if t.Bounds.Intersect(sc).Empty() {
			continue // fully clipped away: no load, no store
		}
		d := dec[ti]
		fb.loadAttachment(t, d, AttColor)
		fb.loadAttachment(t, d, AttDepth)
		fb.loadAttachment(t, d, AttStencil)
	}
}

func (fb *Framebuffer) loadAttachment(t *Tile, d tileDecision, att Attachment) {
	n := t.width() * t.height() * fb.Samples
	switch d.load[att] {
	case LoadClear:
		switch att {
		case AttColor:
			for i := 0; i < n; i++ {
				t.color[i] = fb.ClearColor
			}
		case AttDepth:
			for i := 0; i < n; i++ {
				t.depth[i] = fb.ClearDepth
			}
		case AttStencil:
			for i := 0; i < n; i++ {
				t.stencil[i] = fb.ClearStencil
			}
		}
	case LoadLoad:
		// Copy the surface region covered by this tile into tile memory.
		for py := t.Bounds.MinY; py < t.Bounds.MaxY; py++ {
			for px := t.Bounds.MinX; px < t.Bounds.MaxX; px++ {
				for s := 0; s < fb.Samples; s++ {
					to := t.sampleOffset(px, py, s, fb.Samples)
					from := (py*fb.W+px)*fb.Samples + s
					switch att {
					case AttColor:
						// Surface color is resolved; broadcast to samples.
						t.color[to] = fb.Surf.color[py*fb.W+px]
					case AttDepth:
						t.depth[to] = fb.Surf.depth[from]
					case AttStencil:
						t.stencil[to] = fb.Surf.stencil[from]
					}
				}
			}
		}
	case LoadDontCare:
		// Leave tile memory as-is; contents are undefined by contract.
	}
}

// endPass resolves MSAA and writes tile memory back to the surface for
// every attachment whose tile-local store decision is Store.
func (fb *Framebuffer) endPass(p *Pass, dec []tileDecision) {
	sc := p.scissor(fb)
	for ti := range fb.tiles {
		t := &fb.tiles[ti]
		if t.Bounds.Intersect(sc).Empty() {
			continue
		}
		d := dec[ti]
		for py := t.Bounds.MinY; py < t.Bounds.MaxY; py++ {
			for px := t.Bounds.MinX; px < t.Bounds.MaxX; px++ {
				pix := py*fb.W + px
				if d.store[AttColor] == StoreStore {
					// MSAA resolve: box-filter all samples. Never present
					// raw per-sample data.
					acc := pixel{}
					for s := 0; s < fb.Samples; s++ {
						acc = acc.add(t.color[t.sampleOffset(px, py, s, fb.Samples)])
					}
					fb.Surf.color[pix] = acc.scale(1 / float32(fb.Samples))
				}
				for s := 0; s < fb.Samples; s++ {
					to := pix*fb.Samples + s
					from := t.sampleOffset(px, py, s, fb.Samples)
					if d.store[AttDepth] == StoreStore {
						fb.Surf.depth[to] = t.depth[from]
					}
					if d.store[AttStencil] == StoreStore {
						fb.Surf.stencil[to] = t.stencil[from]
					}
				}
			}
		}
	}
}

// drawTriangle rasterizes one draw command into every tile it overlaps,
// clipped by the pass scissor window.
func (fb *Framebuffer) drawTriangle(p *Pass, d *DrawCmd) {
	sc := p.scissor(fb)
	db := d.bounds().Intersect(sc)
	if db.Empty() {
		return
	}
	area := edge(d.V0.X, d.V0.Y, d.V1.X, d.V1.Y, d.V2.X, d.V2.Y)
	if area == 0 {
		return
	}
	for ti := range fb.tiles {
		t := &fb.tiles[ti]
		clip := t.Bounds.Intersect(db)
		if clip.Empty() {
			continue
		}
		for py := clip.MinY; py < clip.MaxY; py++ {
			for px := clip.MinX; px < clip.MaxX; px++ {
				fb.shadePixel(t, px, py, d, area)
			}
		}
	}
}

func (fb *Framebuffer) shadePixel(t *Tile, px, py int, d *DrawCmd, area float32) {
	for s := 0; s < fb.Samples; s++ {
		fx := float32(px) + sampleOffsets[s][0]
		fy := float32(py) + sampleOffsets[s][1]
		w0 := edge(d.V1.X, d.V1.Y, d.V2.X, d.V2.Y, fx, fy)
		w1 := edge(d.V2.X, d.V2.Y, d.V0.X, d.V0.Y, fx, fy)
		w2 := edge(d.V0.X, d.V0.Y, d.V1.X, d.V1.Y, fx, fy)
		// Accept either winding: weights must agree with the area sign.
		if area > 0 {
			if w0 < 0 || w1 < 0 || w2 < 0 {
				continue
			}
		} else {
			if w0 > 0 || w1 > 0 || w2 > 0 {
				continue
			}
		}
		z := (w0*d.V0.Z + w1*d.V1.Z + w2*d.V2.Z) / area
		off := t.sampleOffset(px, py, s, fb.Samples)
		if d.DepthTest && z > t.depth[off] {
			continue
		}
		if d.StencilTest && t.stencil[off] != d.StencilRef {
			continue
		}
		t.touched = true
		if d.DepthWrite {
			t.depth[off] = z
		}
		if d.StencilWrite {
			t.stencil[off] = d.StencilRef
			continue // stencil-only pass: no color write
		}
		t.color[off] = d.Color
	}
}

// Run executes all passes under the given policy and returns the surface.
func (fb *Framebuffer) Run(passes []Pass, policy Policy) *Surface {
	touched := touchedTiles(fb, passes)
	decisions := policy(fb, passes, touched)
	for pi := range passes {
		p := &passes[pi]
		fb.beginPass(p, decisions[pi])
		for di := range p.Draws {
			fb.drawTriangle(p, &p.Draws[di])
		}
		fb.endPass(p, decisions[pi])
	}
	return fb.Surf
}
