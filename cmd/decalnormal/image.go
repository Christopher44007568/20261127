package main

import (
	"image"
	"image/png"
	"math"
	"os"
)

// FloatImage stores linear RGB values plus coverage in the [0, 1] range.
type FloatImage struct {
	W, H int
	R    []float64
	G    []float64
	B    []float64
}

func newFloatImage(w, h int) *FloatImage {
	return &FloatImage{W: w, H: h, R: make([]float64, w*h), G: make([]float64, w*h), B: make([]float64, w*h)}
}

func (im *FloatImage) set(x, y int, c Vec3) {
	i := y*im.W + x
	im.R[i], im.G[i], im.B[i] = c.X, c.Y, c.Z
}

// encode tonemaps linear color, applies the sRGB gamma and writes a PNG.
func (im *FloatImage) writePNG(path string) error {
	rgba := image.NewRGBA(image.Rect(0, 0, im.W, im.H))
	for i := range im.R {
		rgba.Pix[i*4+0] = toSRGB8(im.R[i])
		rgba.Pix[i*4+1] = toSRGB8(im.G[i])
		rgba.Pix[i*4+2] = toSRGB8(im.B[i])
		rgba.Pix[i*4+3] = 255
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, rgba)
}

func toSRGB8(linear float64) uint8 {
	v := clamp(linear, 0, 1)
	if v <= 0.0031308 {
		v *= 12.92
	} else {
		v = 1.055*math.Pow(v, 1/2.4) - 0.055
	}
	return uint8(clamp(math.Round(v*255), 0, 255))
}

// normalColor maps a unit normal to an RGB color for visualization.
func normalColor(n Vec3) Vec3 {
	return v3(n.X*0.5+0.5, n.Y*0.5+0.5, n.Z*0.5+0.5)
}
