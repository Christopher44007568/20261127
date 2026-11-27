package main

// CachedText is a single-slot rendered-text cache backed by a reusable RGBA
// surface. Updating the string re-rasters into the same storage; the whole
// surface is cleared first so glyph texels left by a previous, longer string
// can never survive past the new text bounds.
type CachedText struct {
	font  *Font
	scale int
	color Color

	text    string
	version int

	// tight w/h of the current text, while surf may be larger.
	w, h int
	surf *Sprite // surf.W == surf.Stride == capW, surf.H == capH

	rebuilds int
}

// NewCachedText creates an empty cache line.
func NewCachedText(font *Font, scale int, color Color) *CachedText {
	if font == nil {
		font = DefaultFont()
	}
	if scale < 1 {
		scale = 1
	}
	h := fontGlyphH * scale
	return &CachedText{font: font, scale: scale, color: color, h: h}
}

// SetText updates the cached string. It returns true when the text changed.
func (c *CachedText) SetText(s string) bool {
	if s == c.text && c.surf != nil {
		return false
	}
	c.text = s
	c.version++
	c.rebuilds++

	w, h := c.font.Measure(s, c.scale)
	if h == 0 {
		h = fontGlyphH * c.scale
	}
	c.w, c.h = w, h

	// Grow the reusable surface when needed. New storage starts transparent.
	if c.surf == nil || w > c.surf.W || h > c.surf.H {
		capW, capH := w, h
		if capW < 1 {
			capW = 1
		}
		if capH < 1 {
			capH = 1
		}
		c.surf = NewSprite(capW, capH)
	}

	// Critical invalidation step: wipe the ENTIRE old surface, including the
	// area occupied by the previous longer string. Without this, shrinking
	// e.g. "PIXEL-PERFECT" to "GO!" leaves ghost texels in reused storage.
	clear(c.surf.Pix)

	if w > 0 {
		c.font.DrawInto(c.surf.View(w, h), s, 0, 0, c.scale, c.color)
	}
	return true
}

func (c *CachedText) Text() string  { return c.text }
func (c *CachedText) Version() int  { return c.version }
func (c *CachedText) Rebuilds() int { return c.rebuilds }

// Size returns the tight pixel size of the current text.
func (c *CachedText) Size() (w, h int) { return c.w, c.h }

// Surface returns the full reusable cache texture. Callers may blit it
// directly: SetText guarantees every texel outside the current glyphs is
// transparent because the whole surface is cleared on each update.
func (c *CachedText) Surface() *Sprite {
	return c.surf
}

// SurfaceAlpha reports the alpha value of a cache-storage texel. Tests use
// it to prove that stale texels beyond the current text width are zero.
func (c *CachedText) SurfaceAlpha(x, y int) uint8 {
	if c.surf == nil || x < 0 || y < 0 || x >= c.surf.W || y >= c.surf.H {
		return 0
	}
	return c.surf.Pix[4*(y*c.surf.Stride+x)+3]
}
