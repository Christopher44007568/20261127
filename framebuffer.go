package main

// pixel is a linear RGBA color with float32 channels in [0,1].
type pixel struct {
	R, G, B, A float32
}

func rgba(r, g, b, a float32) pixel { return pixel{r, g, b, a} }

// add accumulates src into p (used by MSAA resolve).
func (p pixel) add(s pixel) pixel { return pixel{p.R + s.R, p.G + s.G, p.B + s.B, p.A + s.A} }

func (p pixel) scale(k float32) pixel {
	return pixel{p.R * k, p.G * k, p.B * k, p.A * k}
}

// Surface is the system-memory backing of the attachments for the whole
// framebuffer. Tile memory is written back here on Store. Color is stored
// resolved (one value per pixel); depth and stencil keep every MSAA sample
// so mid-frame load/store round-trips are lossless.
type Surface struct {
	W, H    int
	Samples int
	color   []pixel
	depth   []float32
	stencil []uint8
}

func newSurface(w, h, samples int) *Surface {
	return &Surface{
		W:       w,
		H:       h,
		Samples: samples,
		color:   make([]pixel, w*h),
		depth:   make([]float32, w*h*samples),
		stencil: make([]uint8, w*h*samples),
	}
}

// Framebuffer owns the surfaces plus the tile grid, and executes passes.
type Framebuffer struct {
	W, H    int
	TilePx  int // tile edge length in pixels
	Samples int // MSAA sample count per pixel

	Surf           *Surface
	tiles          []Tile
	tilesX, tilesY int

	ClearColor   pixel
	ClearDepth   float32
	ClearStencil uint8
}

func NewFramebuffer(w, h, tilePx, samples int) *Framebuffer {
	fb := &Framebuffer{
		W: w, H: h, TilePx: tilePx, Samples: samples,
		Surf:         newSurface(w, h, samples),
		ClearColor:   rgba(0, 0, 0, 1),
		ClearDepth:   1.0,
		ClearStencil: 0,
	}
	fb.tilesX = (w + tilePx - 1) / tilePx
	fb.tilesY = (h + tilePx - 1) / tilePx
	fb.tiles = make([]Tile, fb.tilesX*fb.tilesY)
	for ty := 0; ty < fb.tilesY; ty++ {
		for tx := 0; tx < fb.tilesX; tx++ {
			b := Rect{tx * tilePx, ty * tilePx, min((tx+1)*tilePx, w), min((ty+1)*tilePx, h)}
			n := (b.MaxX - b.MinX) * (b.MaxY - b.MinY) * samples
			idx := ty*fb.tilesX + tx
			fb.tiles[idx] = Tile{
				Bounds:  b,
				index:   idx,
				color:   make([]pixel, n),
				depth:   make([]float32, n),
				stencil: make([]uint8, n),
			}
		}
	}
	return fb
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
