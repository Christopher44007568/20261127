package main

import (
	"image"
	"image/png"
	"os"
)

func savePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := enc.Encode(f, img); err != nil {
		return err
	}
	return f.Close()
}

func loadPNG(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, err
	}
	return img, nil
}

func pngDecodeConfig(f *os.File) (image.Config, error) {
	return png.DecodeConfig(f)
}

// resizeBilinear scales src to w×h (used only if browser DPI differs).
func resizeBilinear(src *image.RGBA, w, h int) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	sw, sh := src.Bounds().Dx(), src.Bounds().Dy()
	if sw == w && sh == h {
		return src
	}
	for y := 0; y < h; y++ {
		fy := float64(y) * float64(sh) / float64(h)
		y0 := int(fy)
		ty := fy - float64(y0)
		y1 := y0 + 1
		if y1 >= sh {
			y1 = sh - 1
		}
		for x := 0; x < w; x++ {
			fx := float64(x) * float64(sw) / float64(w)
			x0 := int(fx)
			tx := fx - float64(x0)
			x1 := x0 + 1
			if x1 >= sw {
				x1 = sw - 1
			}
			c00 := src.RGBAAt(x0, y0)
			c10 := src.RGBAAt(x1, y0)
			c01 := src.RGBAAt(x0, y1)
			c11 := src.RGBAAt(x1, y1)
			r := lerp(lerpF(c00.R, c10.R, tx), lerpF(c01.R, c11.R, tx), ty)
			g := lerp(lerpF(c00.G, c10.G, tx), lerpF(c01.G, c11.G, tx), ty)
			b := lerp(lerpF(c00.B, c10.B, tx), lerpF(c01.B, c11.B, tx), ty)
			o := out.PixOffset(x, y)
			out.Pix[o] = uint8(r + 0.5)
			out.Pix[o+1] = uint8(g + 0.5)
			out.Pix[o+2] = uint8(b + 0.5)
			out.Pix[o+3] = 255
		}
	}
	return out
}

func lerpF(a, b uint8, t float64) float64 {
	return float64(a) + (float64(b)-float64(a))*t
}

func lerp(a, b float64, t float64) float64 {
	return a + (b-a)*t
}

func newCanvas(w, h int, bg [3]uint8) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i] = bg[0]
		img.Pix[i+1] = bg[1]
		img.Pix[i+2] = bg[2]
		img.Pix[i+3] = 255
	}
	return img
}
