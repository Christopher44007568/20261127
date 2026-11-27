package main

// Color is an opaque 8-bit RGB color.
type Color struct {
	R, G, B uint8
}

func rgb(r, g, b int) Color {
	return Color{R: clampByte(r), G: clampByte(g), B: clampByte(b)}
}

func clampByte(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

// Sprite is a straight-alpha RGBA bitmap. Stride is the number of pixels
// between consecutive rows; it can exceed W so a cached surface can be
// reused without reallocating or compacting.
type Sprite struct {
	W, H   int
	Stride int
	Pix    []byte // len(Pix) >= 4*Stride*H
}

// NewSprite allocates a zeroed (fully transparent) sprite.
func NewSprite(w, h int) *Sprite {
	return &Sprite{W: w, H: h, Stride: w, Pix: make([]byte, 4*w*h)}
}

// View returns a tight sprite header over the same pixel storage.
func (s *Sprite) View(w, h int) *Sprite {
	return &Sprite{W: w, H: h, Stride: s.Stride, Pix: s.Pix}
}

// Framebuffer is a packed RGB canvas with a current clip rectangle.
// Every writer honors Clip, which is what makes dirty-rectangle rendering
// unable to touch pixels outside the requested repaint area.
type Framebuffer struct {
	W, H int
	Pix  []byte // RGBRGB..., length 3*W*H
	clip Rect
}

// NewFramebuffer creates a black canvas.
func NewFramebuffer(w, h int) *Framebuffer {
	f := &Framebuffer{W: w, H: h, Pix: make([]byte, 3*w*h)}
	f.ResetClip()
	return f
}

// Clone returns a deep copy.
func (f *Framebuffer) Clone() *Framebuffer {
	cp := NewFramebuffer(f.W, f.H)
	copy(cp.Pix, f.Pix)
	cp.clip = f.clip
	return cp
}

// ResetClip restores clipping to the whole canvas.
func (f *Framebuffer) ResetClip() {
	f.clip = Rect{X0: 0, Y0: 0, X1: f.W, Y1: f.H}
}

// SetClip intersects r with the canvas bounds and returns the effective clip.
func (f *Framebuffer) SetClip(r Rect) Rect {
	f.clip = r.Intersect(Rect{X0: 0, Y0: 0, X1: f.W, Y1: f.H})
	return f.clip
}

// Clip returns the current clip rectangle.
func (f *Framebuffer) Clip() Rect { return f.clip }

func (f *Framebuffer) index(x, y int) int {
	return 3 * (y*f.W + x)
}

// SetPixel writes one pixel when it is inside the clip.
func (f *Framebuffer) SetPixel(x, y int, c Color) {
	if x < f.clip.X0 || x >= f.clip.X1 || y < f.clip.Y0 || y >= f.clip.Y1 {
		return
	}
	i := f.index(x, y)
	f.Pix[i] = c.R
	f.Pix[i+1] = c.G
	f.Pix[i+2] = c.B
}

// Get reads one pixel; out-of-range reads return black.
func (f *Framebuffer) Get(x, y int) Color {
	if x < 0 || y < 0 || x >= f.W || y >= f.H {
		return Color{}
	}
	i := f.index(x, y)
	return Color{R: f.Pix[i], G: f.Pix[i+1], B: f.Pix[i+2]}
}

// Clear fills the entire canvas regardless of clip.
func (f *Framebuffer) Clear(c Color) {
	for i := 0; i < len(f.Pix); i += 3 {
		f.Pix[i] = c.R
		f.Pix[i+1] = c.G
		f.Pix[i+2] = c.B
	}
}

// FillRect fills r with an opaque color, clipped to the current clip.
func (f *Framebuffer) FillRect(r Rect, c Color) {
	r = r.Intersect(f.clip)
	if r.Empty() {
		return
	}
	rowBytes := (r.X1 - r.X0) * 3
	for y := r.Y0; y < r.Y1; y++ {
		i0 := f.index(r.X0, y)
		for x := 0; x < rowBytes; x += 3 {
			f.Pix[i0+x] = c.R
			f.Pix[i0+x+1] = c.G
			f.Pix[i0+x+2] = c.B
		}
	}
}

// StrokeRect draws an inside border of thickness t around r.
func (f *Framebuffer) StrokeRect(r Rect, t int, c Color) {
	if r.Empty() || t <= 0 {
		return
	}
	f.FillRect(RectWH(r.X0, r.Y0, r.Width(), t), c)          // top
	f.FillRect(RectWH(r.X0, r.Y1-t, r.Width(), t), c)        // bottom
	f.FillRect(RectWH(r.X0, r.Y0+t, t, r.Height()-2*t), c)   // left
	f.FillRect(RectWH(r.X1-t, r.Y0+t, t, r.Height()-2*t), c) // right
}

// FillDisk fills an integer-radius disk centered at (cx, cy).
// Its conservative bounds are Rect{cx-r, cy-r, cx+r+1, cy+r+1}.
func (f *Framebuffer) FillDisk(cx, cy, rad int, c Color) {
	if rad < 0 {
		return
	}
	bounds := Rect{cx - rad, cy - rad, cx + rad + 1, cy + rad + 1}.Intersect(f.clip)
	if bounds.Empty() {
		return
	}
	rr := rad * rad
	for y := bounds.Y0; y < bounds.Y1; y++ {
		dy := y - cy
		for x := bounds.X0; x < bounds.X1; x++ {
			dx := x - cx
			if dx*dx+dy*dy <= rr {
				i := f.index(x, y)
				f.Pix[i] = c.R
				f.Pix[i+1] = c.G
				f.Pix[i+2] = c.B
			}
		}
	}
}

// FillTriangle fills a triangle using a sign test. The same code path is
// used by full and dirty renders, so results are identical by construction.
func (f *Framebuffer) FillTriangle(x0, y0, x1, y1, x2, y2 int, c Color) {
	minX := minInt(x0, minInt(x1, x2))
	maxX := maxInt(x0, maxInt(x1, x2)) + 1
	minY := minInt(y0, minInt(y1, y2))
	maxY := maxInt(y0, maxInt(y1, y2)) + 1
	bounds := Rect{minX, minY, maxX, maxY}.Intersect(f.clip)
	if bounds.Empty() {
		return
	}
	area := (x1-x0)*(y2-y0) - (y1-y0)*(x2-x0)
	for y := bounds.Y0; y < bounds.Y1; y++ {
		for x := bounds.X0; x < bounds.X1; x++ {
			w0 := (x1-x0)*(y-y0) - (y1-y0)*(x-x0)
			w1 := (x2-x1)*(y-y1) - (y2-y1)*(x-x1)
			w2 := (x0-x2)*(y-y2) - (y0-y2)*(x-x2)
			inside := false
			if area > 0 {
				inside = w0 >= 0 && w1 >= 0 && w2 >= 0
			} else {
				inside = w0 <= 0 && w1 <= 0 && w2 <= 0
			}
			if inside {
				i := f.index(x, y)
				f.Pix[i] = c.R
				f.Pix[i+1] = c.G
				f.Pix[i+2] = c.B
			}
		}
	}
}

// CopyRect copies RGB pixels from src into f for the area r, honoring clip.
// Used to restore the immutable static background inside dirty rectangles.
func (f *Framebuffer) CopyRect(src *Framebuffer, r Rect) {
	r = r.Intersect(f.clip)
	if r.Empty() || src.W != f.W || src.H != f.H {
		return
	}
	rowBytes := r.Width() * 3
	for y := r.Y0; y < r.Y1; y++ {
		dst := f.index(r.X0, y)
		srcIdx := src.index(r.X0, y)
		copy(f.Pix[dst:dst+rowBytes], src.Pix[srcIdx:srcIdx+rowBytes])
	}
}

// Blit alpha-blends spr at (x, y), honoring the current clip.
func (f *Framebuffer) Blit(spr *Sprite, x, y int) {
	if spr == nil {
		return
	}
	dstRect := RectWH(x, y, spr.W, spr.H).Intersect(f.clip)
	if dstRect.Empty() {
		return
	}
	for dy := dstRect.Y0 - y; dy < dstRect.Y1-y; dy++ {
		srcRow := dy * spr.Stride
		dstY := y + dy
		for dx := dstRect.X0 - x; dx < dstRect.X1-x; dx++ {
			si := 4 * (srcRow + dx)
			a := uint32(spr.Pix[si+3])
			if a == 0 {
				continue
			}
			di := f.index(x+dx, dstY)
			if a == 255 {
				f.Pix[di] = spr.Pix[si]
				f.Pix[di+1] = spr.Pix[si+1]
				f.Pix[di+2] = spr.Pix[si+2]
				continue
			}
			inv := 255 - a
			f.Pix[di] = uint8((uint32(spr.Pix[si])*a + uint32(f.Pix[di])*inv) / 255)
			f.Pix[di+1] = uint8((uint32(spr.Pix[si+1])*a + uint32(f.Pix[di+1])*inv) / 255)
			f.Pix[di+2] = uint8((uint32(spr.Pix[si+2])*a + uint32(f.Pix[di+2])*inv) / 255)
		}
	}
}

// Dim multiplies every channel by factor/255 (255 leaves the image unchanged).
func (f *Framebuffer) Dim(factor uint8) {
	m := uint32(factor)
	for i := 0; i < len(f.Pix); i += 3 {
		f.Pix[i] = uint8(uint32(f.Pix[i]) * m / 255)
		f.Pix[i+1] = uint8(uint32(f.Pix[i+1]) * m / 255)
		f.Pix[i+2] = uint8(uint32(f.Pix[i+2]) * m / 255)
	}
}

// OverlayRect alpha-blends a flat color over r without touching clip state.
func (f *Framebuffer) OverlayRect(r Rect, c Color, alpha uint8) {
	r = r.Intersect(Rect{0, 0, f.W, f.H})
	if r.Empty() || alpha == 0 {
		return
	}
	a := uint32(alpha)
	inv := 255 - a
	for y := r.Y0; y < r.Y1; y++ {
		for x := r.X0; x < r.X1; x++ {
			i := f.index(x, y)
			f.Pix[i] = uint8((uint32(c.R)*a + uint32(f.Pix[i])*inv) / 255)
			f.Pix[i+1] = uint8((uint32(c.G)*a + uint32(f.Pix[i+1])*inv) / 255)
			f.Pix[i+2] = uint8((uint32(c.B)*a + uint32(f.Pix[i+2])*inv) / 255)
		}
	}
}

// Equal reports whether two framebuffers have identical dimensions and pixels.
func (f *Framebuffer) Equal(o *Framebuffer) bool {
	if f.W != o.W || f.H != o.H || len(f.Pix) != len(o.Pix) {
		return false
	}
	for i := range f.Pix {
		if f.Pix[i] != o.Pix[i] {
			return false
		}
	}
	return true
}

// MismatchCount returns the number of pixels that differ between f and o.
func MismatchCount(a, b *Framebuffer) int {
	if a.W != b.W || a.H != b.H {
		return maxInt(a.W, b.W) * maxInt(a.H, b.H)
	}
	n := 0
	for y := 0; y < a.H; y++ {
		for x := 0; x < a.W; x++ {
			i := a.index(x, y)
			if a.Pix[i] != b.Pix[i] || a.Pix[i+1] != b.Pix[i+1] || a.Pix[i+2] != b.Pix[i+2] {
				n++
			}
		}
	}
	return n
}
