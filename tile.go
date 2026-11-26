package main

// LoadOp describes how a tile attachment is initialized at pass begin.
type LoadOp int

const (
	LoadClear    LoadOp = iota // initialize tile memory to the clear value
	LoadLoad                   // load previous surface contents into tile memory
	LoadDontCare               // tile memory starts undefined
)

// StoreOp describes what happens to tile memory at pass end.
type StoreOp int

const (
	StoreStore    StoreOp = iota // write tile memory back to the surface
	StoreDontCare                // discard tile memory (surface keeps old data)
)

// Attachment identifies one of the three tile-local attachments.
type Attachment int

const (
	AttColor Attachment = iota
	AttDepth
	AttStencil
	numAttachments = 3
)

// Rect is a pixel-space rectangle; Max is exclusive.
type Rect struct {
	MinX, MinY, MaxX, MaxY int
}

func (r Rect) Empty() bool { return r.MinX >= r.MaxX || r.MinY >= r.MaxY }

func (r Rect) Intersect(o Rect) Rect {
	if o.MinX > r.MinX {
		r.MinX = o.MinX
	}
	if o.MinY > r.MinY {
		r.MinY = o.MinY
	}
	if o.MaxX < r.MaxX {
		r.MaxX = o.MaxX
	}
	if o.MaxY < r.MaxY {
		r.MaxY = o.MaxY
	}
	return r
}

// Tile is one tile-local chunk of framebuffer memory. On a tiler GPU this
// maps to on-chip tile memory; contents only reach the surface via Store.
type Tile struct {
	Bounds Rect
	index  int // tile index in row-major tile grid

	color   []pixel   // Samples per pixel
	depth   []float32 // Samples per pixel
	stencil []uint8   // Samples per pixel

	touched bool // any draw call wrote at least one sample this pass
}

func (t *Tile) width() int  { return t.Bounds.MaxX - t.Bounds.MinX }
func (t *Tile) height() int { return t.Bounds.MaxY - t.Bounds.MinY }

// sampleOffset returns the index of (px,py,sample) inside tile-local buffers.
func (t *Tile) sampleOffset(px, py, sample, samples int) int {
	return ((py-t.Bounds.MinY)*t.width()+(px-t.Bounds.MinX))*samples + sample
}
