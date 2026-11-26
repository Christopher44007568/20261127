package viz

import (
	"image"
	"image/color"

	"coverlab/cover"
)

// ColorTile renders the scene crop as a gridded zoomed tile. If mark is
// non-nil it outlines that source-space pixel (used to track one brightness
// cell across a sweep).
func ColorTile(scene cover.Scene, rgb []cover.RGB, cell int, mark *image.Point) *image.NRGBA {
	x0, y0, _, _, cw, ch := cropDims(scene)
	img := image.NewNRGBA(image.Rect(0, 0, cw*cell, ch*cell))
	for jy := 0; jy < ch; jy++ {
		for jx := 0; jx < cw; jx++ {
			sx, sy := x0+jx, y0+jy
			c := rgb[sy*scene.W+sx]
			fillRect(img, jx*cell, jy*cell, cell, cell,
				color.NRGBA{R: clamp8(c.R), G: clamp8(c.G), B: clamp8(c.B), A: 255})
		}
	}
	gridAlpha := uint8(70)
	if cell < 16 {
		gridAlpha = 45
	}
	DrawCellGrid(img, 0, 0, cw, ch, cell, color.NRGBA{R: 0, G: 0, B: 0, A: gridAlpha})
	if mark != nil && mark.X >= x0 && mark.X < x0+cw && mark.Y >= y0 && mark.Y < y0+ch {
		r := image.Rect((mark.X-x0)*cell, (mark.Y-y0)*cell,
			(mark.X-x0+1)*cell, (mark.Y-y0+1)*cell)
		DrawRectOutline(img, r, 2, color.NRGBA{R: 220, G: 30, B: 30, A: 255})
	}
	return img
}
