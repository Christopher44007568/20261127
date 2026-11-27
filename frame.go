package main

import (
	"image"
	"image/png"
	"math"
	"os"
)

type Frame struct {
	Width  int
	Height int
	Pix    []Vec3
}

func NewFrame(w, h int) *Frame {
	return &Frame{Width: w, Height: h, Pix: make([]Vec3, w*h)}
}

func (f *Frame) At(x, y int) int { return y*f.Width + x }

func (f *Frame) Set(x, y int, c Vec3) {
	if x >= 0 && x < f.Width && y >= 0 && y < f.Height {
		f.Pix[f.At(x, y)] = c
	}
}

func (f *Frame) Add(x, y int, c Vec3) {
	if x >= 0 && x < f.Width && y >= 0 && y < f.Height {
		i := f.At(x, y)
		f.Pix[i] = f.Pix[i].Add(c)
	}
}

func srgbToLinear8(v uint8) float64 {
	c := float64(v) / 255
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func linearToSRGB(c float64) uint8 {
	c = clamp01(c)
	var s float64
	if c <= 0.0031308 {
		s = 12.92 * c
	} else {
		s = 1.055*math.Pow(c, 1/2.4) - 0.055
	}
	return uint8(math.Round(clamp01(s) * 255))
}

func acesTonemap(c float64) float64 {
	a, b, cc, d, e := 2.51, 0.03, 2.43, 0.59, 0.14
	return clamp01((c * (a*c + b)) / (c*(cc*c+d) + e))
}

func luminance(c Vec3) float64 { return 0.2126*c.X + 0.7152*c.Y + 0.0722*c.Z }

// Bloom extracts an HDR bright pass into a half-resolution buffer and applies
// two box passes. Repeated box filtering is a cheap deterministic Gaussian
// approximation and keeps zero third-party dependencies.
func (f *Frame) Bloom(threshold, intensity float64, radius int) {
	hw, hh := f.Width/2, f.Height/2
	if hw < 2 || hh < 2 {
		return
	}
	bright := make([]Vec3, hw*hh)
	for y := 0; y < hh; y++ {
		for x := 0; x < hw; x++ {
			c := f.Pix[f.At(2*x, 2*y)]
			lum := luminance(c)
			if lum > threshold {
				k := (lum - threshold) / math.Max(lum, 1e-5)
				bright[y*hw+x] = c.Mul(k)
			}
		}
	}
	blur := boxBlurHalf(bright, hw, hh, radius)
	blur = boxBlurHalf(blur, hw, hh, radius)
	blur = boxBlurHalf(blur, hw, hh, radius)
	for y := 0; y < f.Height; y++ {
		hy := clampInt(y/2, 0, hh-1)
		for x := 0; x < f.Width; x++ {
			hx := clampInt(x/2, 0, hw-1)
			b := blur[hy*hw+hx]
			i := f.At(x, y)
			f.Pix[i] = f.Pix[i].Add(b.Mul(intensity))
		}
	}
}

func clampInt(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func boxBlurHalf(src []Vec3, w, h, radius int) []Vec3 {
	tmp := make([]Vec3, len(src))
	out := make([]Vec3, len(src))
	for y := 0; y < h; y++ {
		var sum Vec3
		for x := -radius; x <= radius; x++ {
			sum = sum.Add(src[y*w+clampInt(x, 0, w-1)])
		}
		for x := 0; x < w; x++ {
			tmp[y*w+x] = sum.Div(float64(2*radius + 1))
			old := clampInt(x-radius, 0, w-1)
			next := clampInt(x+radius+1, 0, w-1)
			sum = sum.Sub(src[y*w+old]).Add(src[y*w+next])
		}
	}
	for x := 0; x < w; x++ {
		var sum Vec3
		for y := -radius; y <= radius; y++ {
			sum = sum.Add(tmp[clampInt(y, 0, h-1)*w+x])
		}
		for y := 0; y < h; y++ {
			out[y*w+x] = sum.Div(float64(2*radius + 1))
			old := clampInt(y-radius, 0, h-1)
			next := clampInt(y+radius+1, 0, h-1)
			sum = sum.Sub(tmp[old*w+x]).Add(tmp[next*w+x])
		}
	}
	return out
}

func (f *Frame) ToRGBA(exposure float64) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, f.Width, f.Height))
	for i, c := range f.Pix {
		r := linearToSRGB(acesTonemap(c.X * exposure))
		g := linearToSRGB(acesTonemap(c.Y * exposure))
		b := linearToSRGB(acesTonemap(c.Z * exposure))
		o := 4 * i
		img.Pix[o+0] = r
		img.Pix[o+1] = g
		img.Pix[o+2] = b
		img.Pix[o+3] = 255
	}
	return img
}

func WritePNG(path string, img image.Image) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	enc := png.Encoder{CompressionLevel: png.BestCompression}
	if err := enc.Encode(file, img); err != nil {
		return err
	}
	return nil
}
