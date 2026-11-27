package main

// RenderDirtyMap builds the "dirty region diagram": the final frame dimmed,
// with every merged repaint rectangle tinted red and outlined in amber.
func RenderDirtyMap(final *Framebuffer, rects []Rect) *Framebuffer {
	m := final.Clone()
	m.Dim(96)

	fill := Color{R: 255, G: 74, B: 74}
	edge := Color{R: 255, G: 209, B: 64}
	for _, r := range rects {
		m.OverlayRect(r, fill, 82)
		m.StrokeRect(r, 3, edge)
	}
	return m
}

// RenderDirtyMask returns a black/white map: white pixels belong to a merged
// dirty rectangle, black pixels are untouched.
func RenderDirtyMask(w, h int, rects []Rect) *Framebuffer {
	mask := NewFramebuffer(w, h)
	mask.Clear(Color{R: 8, G: 10, B: 16})
	white := Color{R: 255, G: 255, B: 255}
	for _, r := range rects {
		mask.FillRect(r, white)
		mask.StrokeRect(r, 2, Color{R: 120, G: 120, B: 120})
	}
	return mask
}

// RenderDiff amplifies pixel differences between a and b. The result is
// fully black when the frames are pixel-identical.
func RenderDiff(a, b *Framebuffer) *Framebuffer {
	d := NewFramebuffer(a.W, a.H)
	if a.W != b.W || a.H != b.H {
		d.Clear(Color{R: 255, G: 0, B: 255})
		return d
	}
	for y := 0; y < a.H; y++ {
		for x := 0; x < a.W; x++ {
			i := a.index(x, y)
			if a.Pix[i] != b.Pix[i] || a.Pix[i+1] != b.Pix[i+1] || a.Pix[i+2] != b.Pix[i+2] {
				d.Pix[i] = 255
				d.Pix[i+1] = 60
				d.Pix[i+2] = 60
			}
		}
	}
	return d
}

// RenderFontSheet lays out every supported glyph for visual inspection.
func RenderFontSheet(font *Font) *Framebuffer {
	charset := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 !?:-._/<>()'\"+="
	scale := 4
	cols := 16
	cellW := fontAdvX*scale + 16
	cellH := fontGlyphH*scale + 20
	rows := (len([]rune(charset)) + cols - 1) / cols
	w := cols*cellW + 24
	h := rows*cellH + 60

	sheet := NewFramebuffer(w, h)
	sheet.Clear(Color{R: 20, G: 30, B: 58})
	title := font.Render("BUILT-IN 5X7 BITMAP FONT", 3, Color{R: 235, G: 240, B: 255})
	sheet.Blit(title, 24, 18)

	for i, r := range charset {
		col := i % cols
		row := i / cols
		x := 24 + col*cellW
		y := 56 + row*cellH
		sp := font.Render(string(r), scale, Color{R: 255, G: 209, B: 64})
		sheet.Blit(sp, x, y)
	}
	return sheet
}
