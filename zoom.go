package main

import (
	"image"
)

// NearestZoom crops r from src and scales it by factor with nearest sampling.
func NearestZoom(src *image.RGBA, r image.Rectangle, factor int) *image.RGBA {
	crop := Crop(src, r)
	b := crop.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx()*factor, b.Dy()*factor))
	PasteNearest(out, crop, 0, 0, b.Dx()*factor, b.Dy()*factor)
	return out
}

// ComposeZoomFrame places a (usually pixel-cropped) zoom into a titled frame.
func ComposeZoomFrame(fs *FontSet, title, sub, footer string, crop image.Image, slotW float64) *image.RGBA {
	const W = 1720
	b := crop.Bounds()
	slotX := 60.0
	slotY := 158.0
	maxSlotW := int(slotW)
	maxSlotH := 820
	factor := maxI(1, minI(maxSlotW/b.Dx(), maxSlotH/b.Dy()))
	sw := float64(b.Dx() * factor)
	sh := float64(b.Dy() * factor)
	H := int(slotY + sh + 96)
	sc := NewScene(W, H, 2, CPaper)
	sc.Title(title, sub, fs)
	sc.FillRect(slotX-10, slotY-10, slotX+sw+10, slotY+sh+10, CWhite)
	sc.FrameRect(slotX-10+0.5, slotY-10+0.5, slotX+sw+10-0.5, slotY+sh+10-0.5, 1.6, CBorder)
	sc.FrameRect(slotX, slotY, slotX+sw, slotY+sh, 2.2, CBlueSoft)
	if footer != "" {
		sc.DrawText(footer, slotX, slotY+sh+44, fs, TextStyle{Ppem: 18, Color: CInkSoft})
	}
	img := sc.Render()
	PasteNearest(img, crop, int(slotX), int(slotY), int(slotX+sw), int(slotY+sh))
	return img
}

// PointsZoomCrop returns an interesting contour region from the points sheet.
func PointsZoomCrop(sheet *image.RGBA) image.Rectangle {
	cards := cardGrid(1680, 1180, 3, 2, 48, 158, 26, 26)
	g := cards[3] // second row, first column: lowercase g
	a := cardContentArea(g)
	return image.Rect(int(a.MinX+24), int(a.MinY-12), int(a.MaxX-36), int(a.MinY+308))
}
