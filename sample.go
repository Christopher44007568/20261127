package main

import (
	"image"
	"image/color"
	"math"
)

// wrapIndex maps an integer sample index into [0,n). It is applied to every
// bilinear tap, so rotated samples never fall outside the texture.
func wrapIndex(v, n int) int {
	if n <= 0 {
		return 0
	}
	v %= n
	if v < 0 {
		v += n
	}
	return v
}

// SampleWrap bilinearly samples src as an infinitely repeating texture.
// Coordinates use pixel units with pixel centers at i+0.5.
func SampleWrap(src *image.NRGBA, x, y float64) color.NRGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	if w == 0 || h == 0 {
		return color.NRGBA{A: 255}
	}

	px, py := x-0.5, y-0.5
	x0, y0 := int(math.Floor(px)), int(math.Floor(py))
	fx, fy := px-float64(x0), py-float64(y0)

	ix0 := wrapIndex(x0, w) + b.Min.X
	ix1 := wrapIndex(x0+1, w) + b.Min.X
	iy0 := wrapIndex(y0, h) + b.Min.Y
	iy1 := wrapIndex(y0+1, h) + b.Min.Y

	c00 := src.NRGBAAt(ix0, iy0)
	c10 := src.NRGBAAt(ix1, iy0)
	c01 := src.NRGBAAt(ix0, iy1)
	c11 := src.NRGBAAt(ix1, iy1)
	return mix4(c00, c10, c01, c11, fx, fy)
}

func mix4(c00, c10, c01, c11 color.NRGBA, fx, fy float64) color.NRGBA {
	w00 := (1 - fx) * (1 - fy)
	w10 := fx * (1 - fy)
	w01 := (1 - fx) * fy
	w11 := fx * fy
	return color.NRGBA{
		R: clamp8(float64(c00.R)*w00 + float64(c10.R)*w10 + float64(c01.R)*w01 + float64(c11.R)*w11),
		G: clamp8(float64(c00.G)*w00 + float64(c10.G)*w10 + float64(c01.G)*w01 + float64(c11.G)*w11),
		B: clamp8(float64(c00.B)*w00 + float64(c10.B)*w10 + float64(c01.B)*w01 + float64(c11.B)*w11),
		A: clamp8(float64(c00.A)*w00 + float64(c10.A)*w10 + float64(c01.A)*w01 + float64(c11.A)*w11),
	}
}

func clamp8(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(math.Round(v))
}
