package demo

import (
	"image"
	"image/color"
)

// Tiny built-in 5x7 bitmap font so labels need no external font module.
// Glyph art uses '#' for a lit texel and '.' for an empty one.

const (
	fontW = 5
	fontH = 7
)

func glyphFor(r rune) []string {
	if g, ok := glyphs[r]; ok {
		return g
	}
	if r >= 'a' && r <= 'z' {
		if g, ok := glyphs[r-'a'+'A']; ok {
			return g
		}
	}
	return glyphs['?']
}

// TextWidth returns the pixel width of s at integer scale.
func TextWidth(s string, scale int) int {
	if scale < 1 {
		scale = 1
	}
	n := 0
	lines := 1
	maxN := 0
	for _, r := range s {
		if r == '\n' {
			lines++
			if n > maxN {
				maxN = n
			}
			n = 0
			continue
		}
		n++
	}
	if n > maxN {
		maxN = n
	}
	_ = lines
	return maxN * (fontW + 1) * scale
}

// TextHeight returns the pixel height needed for possibly multiline s.
func TextHeight(s string, scale int) int {
	lines := 1
	for _, r := range s {
		if r == '\n' {
			lines++
		}
	}
	return lines * (fontH + 2) * scale
}

// DrawText renders s with top-left origin (x,y).
func DrawText(dst *image.NRGBA, s string, x, y, scale int, c color.NRGBA) {
	if scale < 1 {
		scale = 1
	}
	cx, cy := x, y
	for _, r := range s {
		if r == '\n' {
			cx = x
			cy += (fontH + 2) * scale
			continue
		}
		if r == ' ' {
			cx += (fontW + 1) * scale
			continue
		}
		g := glyphFor(r)
		for row := 0; row < fontH; row++ {
			if row >= len(g) {
				break
			}
			line := g[row]
			for col := 0; col < len(line) && col < fontW; col++ {
				if line[col] != '#' {
					continue
				}
				for sy := 0; sy < scale; sy++ {
					for sx := 0; sx < scale; sx++ {
						px := cx + col*scale + sx
						py := cy + row*scale + sy
						if image.Pt(px, py).In(dst.Bounds()) {
							i := dst.PixOffset(px, py)
							dst.Pix[i+0] = c.R
							dst.Pix[i+1] = c.G
							dst.Pix[i+2] = c.B
							dst.Pix[i+3] = 255
						}
					}
				}
			}
		}
		cx += (fontW + 1) * scale
	}
}

// DrawTextCentered centers s inside rectangle r.
func DrawTextCentered(dst *image.NRGBA, s string, r image.Rectangle, scale int, c color.NRGBA) {
	tw := TextWidth(s, scale)
	th := TextHeight(s, scale)
	DrawText(dst, s, r.Min.X+(r.Dx()-tw)/2, r.Min.Y+(r.Dy()-th)/2, scale, c)
}
