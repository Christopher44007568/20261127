package main

import (
	"image"
	"image/draw"
)

// Crop returns a copy of the rectangle r from src.
func Crop(src *image.RGBA, r image.Rectangle) *image.RGBA {
	r = r.Intersect(src.Bounds())
	out := image.NewRGBA(image.Rect(0, 0, r.Dx(), r.Dy()))
	draw.Draw(out, out.Bounds(), src, r.Min, draw.Src)
	return out
}

// PasteNearest scales src into the destination rectangle using nearest sampling.
func PasteNearest(dst *image.RGBA, src image.Image, dx0, dy0, dx1, dy1 int) {
	dw := dx1 - dx0
	dh := dy1 - dy0
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	for y := 0; y < dh; y++ {
		sy := b.Min.Y + y*sh/dh
		for x := 0; x < dw; x++ {
			sx := b.Min.X + x*sw/dw
			r, g, bl, a := src.At(sx, sy).RGBA()
			o := dst.PixOffset(dx0+x, dy0+y)
			dst.Pix[o] = uint8(r >> 8)
			dst.Pix[o+1] = uint8(g >> 8)
			dst.Pix[o+2] = uint8(bl >> 8)
			dst.Pix[o+3] = uint8(a >> 8)
		}
	}
}

// PasteRGBA composites a same-size-or-smaller RGBA via source-over.
func PasteRGBA(dst *image.RGBA, src *image.RGBA, x, y int) {
	draw.Draw(dst, image.Rect(x, y, x+src.Bounds().Dx(), y+src.Bounds().Dy()),
		src, image.Point{}, draw.Over)
}

// DownscaleAverage produces a box-filtered thumbnail.
func DownscaleAverage(src *image.RGBA, factor int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx()/factor, b.Dy()/factor
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var rr, gg, bb, n uint32
			for dy := 0; dy < factor; dy++ {
				for dx := 0; dx < factor; dx++ {
					sx := x*factor + dx
					sy := y*factor + dy
					if sx >= b.Dx() || sy >= b.Dy() {
						continue
					}
					i := src.PixOffset(b.Min.X+sx, b.Min.Y+sy)
					rr += uint32(src.Pix[i])
					gg += uint32(src.Pix[i+1])
					bb += uint32(src.Pix[i+2])
					n++
				}
			}
			o := out.PixOffset(x, y)
			out.Pix[o] = uint8(rr / n)
			out.Pix[o+1] = uint8(gg / n)
			out.Pix[o+2] = uint8(bb / n)
			out.Pix[o+3] = 255
		}
	}
	return out
}
