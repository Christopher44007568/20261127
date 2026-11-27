package main

import (
	"image"
	"image/color"
)

var (
	uiBg     = color.RGBA{R: 24, G: 26, B: 33, A: 255}
	uiPanel  = color.RGBA{R: 34, G: 38, B: 48, A: 255}
	uiBar    = color.RGBA{R: 12, G: 14, B: 19, A: 235}
	uiText   = color.RGBA{R: 236, G: 238, B: 242, A: 255}
	uiMuted  = color.RGBA{R: 150, G: 158, B: 170, A: 255}
	uiAccent = color.RGBA{R: 255, G: 196, B: 64, A: 255}
	uiBad    = color.RGBA{R: 235, G: 84, B: 84, A: 255}
	uiGood   = color.RGBA{R: 80, G: 210, B: 140, A: 255}
)

func newCanvas(w, h int, bg color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	fillRect(img, 0, 0, w, h, bg)
	return img
}

// paste copies src at integer 1:1 size into dst at (x,y).
func paste(dst *image.RGBA, src image.Image, x, y int) {
	b := src.Bounds()
	for sy := b.Min.Y; sy < b.Max.Y; sy++ {
		for sx := b.Min.X; sx < b.Max.X; sx++ {
			dst.Set(x+sx-b.Min.X, y+sy-b.Min.Y, src.At(sx, sy))
		}
	}
}

// pasteScaledNearest copies an nw x nh nearest-neighbor scaling.
func pasteScaledNearest(dst *image.RGBA, src *image.RGBA, x, y, nw, nh int) {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	for dy := 0; dy < nh; dy++ {
		sy := b.Min.Y + dy*sh/nh
		for dx := 0; dx < nw; dx++ {
			sx := b.Min.X + dx*sw/nw
			dst.Set(x+dx, y+dy, src.RGBAAt(sx, sy))
		}
	}
}

// labelBar draws a solid header strip with left-aligned text.
func labelBar(dst *image.RGBA, x, y, w, h int, text string, accent color.RGBA) {
	fillRect(dst, x, y, w, h, uiBar)
	fillRect(dst, x, y+h-3, w, 3, accent)
	scale := 2
	tw := textWidth(text, scale)
	tx := x + 14
	ty := y + (h-5*scale)/2
	_ = tw
	drawText(dst, text, tx, ty, scale, uiText)
}
