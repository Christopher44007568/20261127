package main

import "unicode"

// Bitmap font geometry. Glyphs are 5x7 cells with one blank column between
// advances; integer scaling gives blocky, deterministic text sprites.
const (
	fontGlyphW = 5
	fontGlyphH = 7
	fontAdvX   = 6
)

// Font is a fixed 5x7 bitmap font. Missing runes render as a hollow box so
// output is always deterministic instead of silently disappearing.
type Font struct {
	glyphs map[rune][fontGlyphH]string
}

// DefaultFont returns the built-in ASCII bitmap font.
func DefaultFont() *Font {
	return &Font{glyphs: builtinGlyphs()}
}

func (f *Font) bitmap(r rune) [fontGlyphH]string {
	if g, ok := f.glyphs[r]; ok {
		return g
	}
	upper := unicode.ToUpper(r)
	if g, ok := f.glyphs[upper]; ok {
		return g
	}
	return fallbackGlyph
}

// Measure returns the tight pixel size of s at integer scale.
func (f *Font) Measure(s string, scale int) (w, h int) {
	if scale < 1 {
		scale = 1
	}
	n := len([]rune(s))
	h = fontGlyphH * scale
	if n == 0 {
		return 0, h
	}
	w = (n*fontAdvX - 1) * scale
	return w, h
}

// Render creates a fresh tight RGBA sprite for s.
func (f *Font) Render(s string, scale int, c Color) *Sprite {
	w, h := f.Measure(s, scale)
	if w == 0 {
		w = 1
	}
	sp := NewSprite(w, h)
	f.DrawInto(sp.View(w, h), s, 0, 0, scale, c)
	return sp.View(w, h)
}

// DrawInto rasterizes s into dst at (x0, y0). The destination sprite uses
// straight alpha; lit texels are fully opaque.
func (f *Font) DrawInto(dst *Sprite, s string, x0, y0, scale int, c Color) {
	if scale < 1 {
		scale = 1
	}
	runes := []rune(s)
	for idx, r := range runes {
		g := f.bitmap(r)
		ox := x0 + idx*fontAdvX*scale
		for row := 0; row < fontGlyphH; row++ {
			line := g[row]
			for col := 0; col < fontGlyphW && col < len(line); col++ {
				if line[col] != '#' {
					continue
				}
				for dy := 0; dy < scale; dy++ {
					py := y0 + row*scale + dy
					if py < 0 || py >= dst.H {
						continue
					}
					for dx := 0; dx < scale; dx++ {
						px := ox + col*scale + dx
						if px < 0 || px >= dst.W {
							continue
						}
						i := 4 * (py*dst.Stride + px)
						dst.Pix[i] = c.R
						dst.Pix[i+1] = c.G
						dst.Pix[i+2] = c.B
						dst.Pix[i+3] = 255
					}
				}
			}
		}
	}
}

// fallbackGlyph is the deterministic "tofu" box for unsupported runes.
var fallbackGlyph = [fontGlyphH]string{
	"#####",
	"#...#",
	"#...#",
	"#...#",
	"#...#",
	"#...#",
	"#####",
}
