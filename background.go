package main

// BuildBackground renders the immutable static background once. Dirty
// repaints restore pixels by copying this buffer, which is exactly identical
// to regenerating the background during a full repaint.
func BuildBackground(w, h int, font *Font) *Framebuffer {
	bg := NewFramebuffer(w, h)

	top := Color{R: 28, G: 44, B: 84}
	bottom := Color{R: 13, G: 20, B: 42}
	for y := 0; y < h; y++ {
		t := 0
		if h > 1 {
			t = y * 255 / (h - 1)
		}
		c := blendColor(top, bottom, uint8(t))
		for x := 0; x < w; x++ {
			i := bg.index(x, y)
			bg.Pix[i] = c.R
			bg.Pix[i+1] = c.G
			bg.Pix[i+2] = c.B
		}
	}

	minor := Color{R: 42, G: 62, B: 108}
	major := Color{R: 64, G: 90, B: 148}
	grid := 40
	for x := 0; x < w; x += grid {
		c := minor
		if x%(grid*5) == 0 {
			c = major
		}
		bg.FillRect(RectWH(x, 0, 1, h), c)
	}
	for y := 0; y < h; y += grid {
		c := minor
		if y%(grid*5) == 0 {
			c = major
		}
		bg.FillRect(RectWH(0, y, w, 1), c)
	}

	bg.StrokeRect(RectWH(1, 1, w-2, h-2), 2, Color{R: 74, G: 100, B: 158})

	title := font.Render("DIRTY RECT CANVAS", 5, Color{R: 236, G: 241, B: 255})
	bg.Blit(title, 48, 38)

	sub := font.Render("STATIC BACKGROUND IS RESTORED INSIDE EVERY DIRTY RECT", 2, Color{R: 158, G: 178, B: 220})
	bg.Blit(sub, 50, 92)

	hint := font.Render("DRAG SHAPES: ONLY OLD+NEW BOUNDS ARE REPAINTED", 2, Color{R: 150, G: 168, B: 208})
	bg.Blit(hint, 50, h-44)

	return bg
}

func blendColor(a, b Color, t uint8) Color {
	u := uint32(t)
	inv := 255 - u
	return Color{
		R: uint8((uint32(a.R)*inv + uint32(b.R)*u) / 255),
		G: uint8((uint32(a.G)*inv + uint32(b.G)*u) / 255),
		B: uint8((uint32(a.B)*inv + uint32(b.B)*u) / 255),
	}
}
