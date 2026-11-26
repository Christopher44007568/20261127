package studio

import "image"

func FillRect(im *FloatImage, x, y, w, h int, c RGBA) {
	p := Premultiply(c)
	for py := maxInt(0, y); py < minInt(im.H, y+h); py++ {
		for px := maxInt(0, x); px < minInt(im.W, x+w); px++ {
			im.Set(px, py, p)
		}
	}
}

func BlendRect(im *FloatImage, x, y, w, h int, c RGBA) {
	p := Premultiply(c)
	for py := maxInt(0, y); py < minInt(im.H, y+h); py++ {
		for px := maxInt(0, x); px < minInt(im.W, x+w); px++ {
			im.OverPixel(px, py, p)
		}
	}
}

func StrokeRect(im *FloatImage, x, y, w, h int, c RGBA) {
	BlendRect(im, x, y, w, 1, c)
	BlendRect(im, x, y+h-1, w, 1, c)
	BlendRect(im, x, y, 1, h, c)
	BlendRect(im, x+w-1, y, 1, h, c)
}

// DrawImageAt composites an existing byte image onto a float canvas without
// resampling. This preserves the exact dither pattern of diagnostic panels.
func DrawImageAt(dst *FloatImage, src *image.NRGBA, x0, y0 int) {
	for y := 0; y < src.Rect.Dy(); y++ {
		for x := 0; x < src.Rect.Dx(); x++ {
			i := y*src.Stride + x*4
			a := src.Pix[i+3]
			if a == 0 {
				continue
			}
			c := SRGB(
				float64(src.Pix[i])/255,
				float64(src.Pix[i+1])/255,
				float64(src.Pix[i+2])/255,
				float64(a)/255,
			)
			dst.OverPixel(x0+x, y0+y, Premultiply(c))
		}
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
