package studio

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

var bayer8 = [8][8]uint8{
	{0, 48, 12, 60, 3, 51, 15, 63},
	{32, 16, 44, 28, 35, 19, 47, 31},
	{8, 56, 4, 52, 11, 59, 7, 55},
	{40, 24, 36, 20, 43, 27, 39, 23},
	{2, 50, 14, 62, 1, 49, 13, 61},
	{34, 18, 46, 30, 33, 17, 45, 29},
	{10, 58, 6, 54, 9, 57, 5, 53},
	{42, 26, 38, 22, 41, 25, 37, 21},
}

func ditherOffset(x, y int) float64 {
	ordered := (float64(bayer8[y&7][x&7]) + 0.5) / 64
	n := 52.9829189 * fract(0.06711056*float64(x)+0.00583715*float64(y))
	ign := fract(n)
	mixed := 0.45*(ordered-0.5) + 0.55*(ign-0.5)
	return 4.0 * mixed
}

func fract(v float64) float64 {
	return v - math.Floor(v)
}

func quantizeSRGB(v float64, offset float64) uint8 {
	v = clamp01(v + offset/255)
	return uint8(v*255 + 0.5)
}

// ToNRGBA converts the premultiplied linear buffer back to straight sRGB for
// PNG. A deterministic Bayer+hash dither is applied in the sRGB quantisation
// domain so smooth float ramps do not collapse into visible 8-bit bands.
func ToNRGBA(im *FloatImage, dither bool) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, im.W, im.H))
	for y := 0; y < im.H; y++ {
		for x := 0; x < im.W; x++ {
			p := im.At(x, y)
			a := clamp01f(p.A)
			off := 0.0
			if dither {
				off = ditherOffset(x, y)
			}
			var straight RGBA
			if a > 1e-7 {
				straight = Unpremultiply(p)
			}
			r := quantizeSRGB(linearToSRGB(float64(straight.R)), off)
			g := quantizeSRGB(linearToSRGB(float64(straight.G)), off)
			b := quantizeSRGB(linearToSRGB(float64(straight.B)), off)
			a8 := uint8(float64(a)*255 + 0.5)
			i := y*out.Stride + x*4
			out.Pix[i] = r
			out.Pix[i+1] = g
			out.Pix[i+2] = b
			out.Pix[i+3] = a8
		}
	}
	return out
}

func ToGray(mask []float32, w, h int) *image.Gray {
	if len(mask) != w*h {
		panic("studio: gray mask size mismatch")
	}
	out := image.NewGray(image.Rect(0, 0, w, h))
	for i, v := range mask {
		out.Pix[i] = uint8(clamp01f(v)*255 + 0.5)
	}
	return out
}

func SavePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	return enc.Encode(f, img)
}

// SolidPNG is useful for background and comparison panels.
func SolidPNG(w, h int, c color.NRGBA) *image.NRGBA {
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*out.Stride + x*4
			out.Pix[i] = c.R
			out.Pix[i+1] = c.G
			out.Pix[i+2] = c.B
			out.Pix[i+3] = c.A
		}
	}
	return out
}
