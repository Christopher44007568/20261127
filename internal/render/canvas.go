package render

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

func NewCanvas(w, h int, bg color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bg}, image.Point{}, draw.Src)
	return img
}

func FillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	fillRect(img, x, y, w, h, c)
}

func DrawImage(img *image.RGBA, src image.Image, x, y int) {
	r := src.Bounds()
	draw.Draw(img, image.Rect(x, y, x+r.Dx(), y+r.Dy()), src, r.Min, draw.Over)
}

// DrawNearest scales a field visualization with hard texel edges. This is
// used for the field panels, where individual texels should be inspectable.
func DrawNearest(img *image.RGBA, src image.Image, dst image.Rectangle) {
	b := src.Bounds()
	sw, sh := b.Dx(), b.Dy()
	dw, dh := dst.Dx(), dst.Dy()
	if sw <= 0 || sh <= 0 || dw <= 0 || dh <= 0 {
		return
	}
	for dy := 0; dy < dh; dy++ {
		sy := b.Min.Y + dy*sh/dh
		for dx := 0; dx < dw; dx++ {
			sx := b.Min.X + dx*sw/dw
			img.Set(dst.Min.X+dx, dst.Min.Y+dy, src.At(sx, sy))
		}
	}
}

func DrawCoverageTinted(img *image.RGBA, cov *image.Gray, x, y int, ink, paper color.RGBA) {
	b := cov.Bounds()
	for yy := 0; yy < b.Dy(); yy++ {
		for xx := 0; xx < b.Dx(); xx++ {
			a := cov.GrayAt(b.Min.X+xx, b.Min.Y+yy).Y
			t := float64(a) / 255
			c := blend(paper, ink, t)
			img.SetRGBA(x+xx, y+yy, c)
		}
	}
}

func blend(a, b color.RGBA, t float64) color.RGBA {
	return color.RGBA{
		R: uint8(float64(a.R)*(1-t) + float64(b.R)*t + 0.5),
		G: uint8(float64(a.G)*(1-t) + float64(b.G)*t + 0.5),
		B: uint8(float64(a.B)*(1-t) + float64(a.B)*t + 0.5),
		A: 255,
	}
}

func StrokeCircle(img *image.RGBA, cx, cy, radius, thickness int, c color.RGBA) {
	if thickness < 1 {
		thickness = 1
	}
	rOut := float64(radius)
	rIn := math.Max(0, rOut-float64(thickness))
	r2Out := rOut * rOut
	r2In := rIn * rIn
	for y := cy - radius - thickness; y <= cy+radius+thickness; y++ {
		for x := cx - radius - thickness; x <= cx+radius+thickness; x++ {
			dx := float64(x - cx)
			dy := float64(y - cy)
			d2 := dx*dx + dy*dy
			if d2 <= r2Out && d2 >= r2In {
				setSafe(img, x, y, c)
			}
		}
	}
}

func DrawCross(img *image.RGBA, cx, cy, r, thickness int, c color.RGBA) {
	FillRect(img, cx-r, cy-thickness/2, 2*r+1, thickness, c)
	FillRect(img, cx-thickness/2, cy-r, thickness, 2*r+1, c)
}

func setSafe(img *image.RGBA, x, y int, c color.RGBA) {
	b := img.Bounds()
	if x >= b.Min.X && x < b.Max.X && y >= b.Min.Y && y < b.Max.Y {
		img.SetRGBA(x, y, c)
	}
}
