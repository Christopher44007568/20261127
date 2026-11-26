package render

import (
	"image"
	"image/color"

	"glyphsdf/internal/field"
)

type Method int

const (
	MethodSDF Method = iota
	MethodMSDF
)

// Coverage reconstructs an alpha coverage bitmap at output scale. Scale is
// output pixels per distance-field texel. Supersampling is ordinary spatial
// anti-aliasing and does not change the represented field.
func Coverage(f *field.Field, method Method, scale float64, supersample int) *image.Gray {
	if scale <= 0 {
		scale = 1
	}
	if supersample < 1 {
		supersample = 1
	}
	w := int(float64(f.W)*scale + 0.5)
	h := int(float64(f.H)*scale + 0.5)
	img := image.NewGray(image.Rect(0, 0, w, h))
	aa := 0.32 / (scale * float64(supersample))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sum float64
			for sy := 0; sy < supersample; sy++ {
				for sx := 0; sx < supersample; sx++ {
					px := float64(x) + (float64(sx)+0.5)/float64(supersample)
					py := float64(y) + (float64(sy)+0.5)/float64(supersample)
					tx := px / scale
					ty := py / scale
					var d float64
					if method == MethodMSDF {
						d = f.SampleMSDF(tx, ty)
					} else {
						d = f.SampleSDF(tx, ty)
					}
					sum += fieldSmooth(-aa, aa, d)
				}
			}
			cov := sum / float64(supersample*supersample)
			img.SetGray(x, y, color.Gray{Y: gray(cov)})
		}
	}
	return img
}

func fieldSmooth(e0, e1, x float64) float64 {
	t := (x - e0) / (e1 - e0)
	if t < 0 {
		t = 0
	} else if t > 1 {
		t = 1
	}
	return t * t * (3 - 2*t)
}

func gray(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 1 {
		return 255
	}
	return uint8(v*255 + 0.5)
}
