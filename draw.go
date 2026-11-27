package main

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

func setPixel(img *image.RGBA, x, y int, c color.RGBA) {
	if x < img.Rect.Min.X || x >= img.Rect.Max.X || y < img.Rect.Min.Y || y >= img.Rect.Max.Y {
		return
	}
	if c.A == 255 {
		img.SetRGBA(x, y, c)
		return
	}
	if c.A == 0 {
		return
	}
	i := img.PixOffset(x, y)
	a := float64(c.A) / 255
	dstA := float64(img.Pix[i+3]) / 255
	outA := a + dstA*(1-a)
	if outA <= 0 {
		return
	}
	for k := 0; k < 3; k++ {
		dst := float64(img.Pix[i+k]) / 255
		src := float64([3]uint8{c.R, c.G, c.B}[k]) / 255
		img.Pix[i+k] = uint8(math.Round(((src*a + dst*dstA*(1-a)) / outA) * 255))
	}
	img.Pix[i+3] = uint8(math.Round(outA * 255))
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			setPixel(img, x, y, c)
		}
	}
}

func strokeRect(img *image.RGBA, x0, y0, x1, y1, width int, c color.RGBA) {
	fillRect(img, x0, y0, x1, y0+width, c)
	fillRect(img, x0, y1-width, x1, y1, c)
	fillRect(img, x0, y0, x0+width, y1, c)
	fillRect(img, x1-width, y0, x1, y1, c)
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	dx := int(math.Abs(float64(x1 - x0)))
	dy := -int(math.Abs(float64(y1 - y0)))
	sx, sy := -1, -1
	if x0 < x1 {
		sx = 1
	}
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		setPixel(img, x0, y0, c)
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func drawTextShadow(img *image.RGBA, x, y int, s string, scale int, c, shadow color.RGBA) {
	drawText(img, x+scale, y+scale, s, scale, shadow)
	drawText(img, x, y, s, scale, c)
}

func fillCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	r2 := r * r
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r2 {
				setPixel(img, x, y, c)
			}
		}
	}
}

func cropImage(src *image.RGBA, rect image.Rectangle) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, rect.Dx(), rect.Dy()))
	draw.Draw(dst, dst.Bounds(), src, rect.Min, draw.Src)
	return dst
}

func resizeBilinear(src *image.RGBA, w, h int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	for y := 0; y < h; y++ {
		sy := (float64(y)+0.5)*float64(sh)/float64(h) - 0.5
		y0 := int(math.Floor(sy))
		fy := sy - float64(y0)
		if y0 < 0 {
			y0, fy = 0, 0
		}
		y1 := y0 + 1
		if y1 >= sh {
			y1 = sh - 1
		}
		for x := 0; x < w; x++ {
			sx := (float64(x)+0.5)*float64(sw)/float64(w) - 0.5
			x0 := int(math.Floor(sx))
			fx := sx - float64(x0)
			if x0 < 0 {
				x0, fx = 0, 0
			}
			x1 := x0 + 1
			if x1 >= sw {
				x1 = sw - 1
			}
			c00 := src.RGBAAt(x0, y0)
			c10 := src.RGBAAt(x1, y0)
			c01 := src.RGBAAt(x0, y1)
			c11 := src.RGBAAt(x1, y1)
			lerp := func(a, b uint8, t float64) uint8 {
				return uint8(math.Round(float64(a)*(1-t) + float64(b)*t))
			}
			r0 := lerp(c00.R, c10.R, fx)
			g0 := lerp(c00.G, c10.G, fx)
			b0 := lerp(c00.B, c10.B, fx)
			r1 := lerp(c01.R, c11.R, fx)
			g1 := lerp(c01.G, c11.G, fx)
			b1 := lerp(c01.B, c11.B, fx)
			dst.SetRGBA(x, y, color.RGBA{
				R: lerp(r0, r1, fy), G: lerp(g0, g1, fy), B: lerp(b0, b1, fy), A: 255,
			})
		}
	}
	return dst
}

func cropResize(src *image.RGBA, rect image.Rectangle, w, h int) *image.RGBA {
	return resizeBilinear(cropImage(src, rect), w, h)
}

func pasteImage(dst *image.RGBA, src *image.RGBA, x, y int) {
	draw.Draw(dst, image.Rect(x, y, x+src.Bounds().Dx(), y+src.Bounds().Dy()), src, image.Point{}, draw.Over)
}

func blendColor(a color.RGBA, b color.RGBA, t float64) color.RGBA {
	f := func(x, y uint8) uint8 { return uint8(math.Round(float64(x)*(1-t) + float64(y)*t)) }
	return color.RGBA{R: f(a.R, b.R), G: f(a.G, b.G), B: f(a.B, b.B), A: f(a.A, b.A)}
}

func verticalGradient(img *image.RGBA, x0, y0, x1, y1 int, top, bottom color.RGBA) {
	if y1 <= y0 {
		return
	}
	for y := y0; y < y1; y++ {
		t := float64(y-y0) / float64(y1-y0)
		c := blendColor(top, bottom, t)
		fillRect(img, x0, y, x1, y+1, c)
	}
}
