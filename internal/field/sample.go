package field

import (
	"image"
	"image/color"
	"math"
)

func bilinear(buf []float64, w, h int, x, y float64) float64 {
	fx := x - 0.5
	fy := y - 0.5
	x0 := int(math.Floor(fx))
	y0 := int(math.Floor(fy))
	tx := fx - math.Floor(fx)
	ty := fy - math.Floor(fy)
	x1 := x0 + 1
	y1 := y0 + 1
	read := func(ix, iy int) float64 {
		if ix < 0 {
			ix = 0
		} else if ix >= w {
			ix = w - 1
		}
		if iy < 0 {
			iy = 0
		} else if iy >= h {
			iy = h - 1
		}
		return buf[iy*w+ix]
	}
	a := read(x0, y0)*(1-tx) + read(x1, y0)*tx
	b := read(x0, y1)*(1-tx) + read(x1, y1)*tx
	return a*(1-ty) + b*ty
}

func (f *Field) SampleSDF(x, y float64) float64 {
	return bilinear(f.SDF, f.W, f.H, x, y)
}

func (f *Field) SampleMSDF(x, y float64) float64 {
	r := bilinear(f.MSDF[0], f.W, f.H, x, y)
	g := bilinear(f.MSDF[1], f.W, f.H, x, y)
	b := bilinear(f.MSDF[2], f.W, f.H, x, y)
	return Median3(r, g, b)
}

func encodeDistance(d, r float64) uint8 {
	v := 0.5 + d/(2*r)
	v = math.Max(0, math.Min(1, v))
	return uint8(math.Round(v * 255))
}

func (f *Field) SDFImage() *image.Gray {
	img := image.NewGray(image.Rect(0, 0, f.W, f.H))
	for i, d := range f.SDF {
		img.Pix[i] = encodeDistance(d, f.Range)
	}
	return img
}

func (f *Field) MSDFImage() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, f.W, f.H))
	for i := 0; i < f.W*f.H; i++ {
		o := i * 4
		img.Pix[o+0] = encodeDistance(f.MSDF[0][i], f.Range)
		img.Pix[o+1] = encodeDistance(f.MSDF[1][i], f.Range)
		img.Pix[o+2] = encodeDistance(f.MSDF[2][i], f.Range)
		img.Pix[o+3] = 255
	}
	return img
}

func CoverageGrayToRGBA(g *image.Gray, ink color.RGBA) *image.RGBA {
	out := image.NewRGBA(g.Bounds())
	for i, v := range g.Pix {
		c := color.RGBA{
			R: uint8(int(ink.R)*int(v)/255 + 255*(255-int(v))/255),
			G: uint8(int(ink.G)*int(v)/255 + 255*(255-int(v))/255),
			B: uint8(int(ink.B)*int(v)/255 + 255*(255-int(v))/255),
			A: 255,
		}
		out.SetRGBA(i%g.Bounds().Dx(), i/g.Bounds().Dx(), c)
	}
	return out
}
