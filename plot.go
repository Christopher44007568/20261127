package main

import (
	"image"
	"image/color"
	"math"
)

var (
	colAxis = color.RGBA{200, 200, 205, 255}
	colGrid = color.RGBA{60, 60, 68, 255}
	colBg   = color.RGBA{18, 18, 22, 255}
	colPeak = color.RGBA{255, 210, 80, 255}
)

var colSpect = []color.RGBA{
	{90, 200, 120, 255},
	{90, 190, 220, 255},
	{120, 130, 255, 255},
	{220, 110, 220, 255},
}

func setPx(img *image.RGBA, x, y int, c color.RGBA) {
	if image.Pt(x, y).In(img.Bounds()) {
		img.SetRGBA(x, y, c)
	}
}

func drawLine(img *image.RGBA, x0, y0, x1, y1 float64, c color.RGBA) {
	steps := int(math.Max(math.Abs(x1-x0), math.Abs(y1-y0))) + 1
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		setPx(img, int(x0+(x1-x0)*t+0.5), int(y0+(y1-y0)*t+0.5), c)
	}
}

func fillRect(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			setPx(img, x, y, c)
		}
	}
}

// plotFrame is a simple axis box with linear coordinate mapping.
type plotFrame struct {
	img            *image.RGBA
	x0, y0, x1, y1 int // pixel bounds of plot area
	u0, u1, v0, v1 float64
}

func (f plotFrame) px(u, v float64) (float64, float64) {
	x := float64(f.x0) + (u-f.u0)/(f.u1-f.u0)*float64(f.x1-f.x0)
	y := float64(f.y1) - (v-f.v0)/(f.v1-f.v0)*float64(f.y1-f.y0)
	return x, y
}

func (f plotFrame) axes() {
	fillRect(f.img, f.x0, f.y0, f.x1, f.y1, colBg)
	for i := 0; i <= 8; i++ { // vertical gridlines
		x := f.x0 + i*(f.x1-f.x0)/8
		drawLine(f.img, float64(x), float64(f.y0), float64(x), float64(f.y1), colGrid)
	}
	for i := 0; i <= 6; i++ { // horizontal gridlines
		y := f.y0 + i*(f.y1-f.y0)/6
		drawLine(f.img, float64(f.x0), float64(y), float64(f.x1), float64(y), colGrid)
	}
	drawLine(f.img, float64(f.x0), float64(f.y1), float64(f.x1), float64(f.y1), colAxis)
	drawLine(f.img, float64(f.x0), float64(f.y0), float64(f.x0), float64(f.y1), colAxis)
}

func (f plotFrame) polyline(us, vs []float64, c color.RGBA) {
	for i := 1; i < len(us); i++ {
		x0, y0 := f.px(us[i-1], vs[i-1])
		x1, y1 := f.px(us[i], vs[i])
		drawLine(f.img, x0, y0, x1, y1, c)
	}
}

// RenderHueStability draws the hue-stability chart:
// left panel: hue swatch strip + peak wavelength vs incidence angle;
// right panel: full reflectance spectra at several angles.
func RenderHueStability(path string, lut *FilmLUT) error {
	const w, h = 1120, 480
	img := newCanvas(w, h, color.RGBA{12, 12, 15, 255})

	angles := make([]float64, 0, 81)
	for a := 0.0; a <= 80; a += 1 {
		angles = append(angles, a)
	}
	peaks := make([]float64, len(angles))
	for i, a := range angles {
		cosT := math.Cos(a * math.Pi / 180)
		s := lut.Shade(cosT, 0)
		peaks[i] = s.PeakLambda()
	}

	// Left: hue swatch strip + peak wavelength curve, x = angle 0..80.
	left := plotFrame{img: img, x0: 50, y0: 30, x1: 520, y1: 380, u0: 0, u1: 80, v0: 350, v1: 700}
	left.axes()
	for a := 0; a <= 80; a += 20 { // angle ticks (deg)
		x, _ := left.px(float64(a), 0)
		drawText(img, int(x)-6, 385, itoa(a), colAxis, 1)
	}
	for l := 400; l <= 700; l += 100 { // wavelength ticks (nm)
		_, y := left.px(0, float64(l))
		drawText(img, 14, int(y)-2, itoa(l), colAxis, 1)
	}
	// Hue strip: each angle column painted with the actual rendered color.
	for _, a := range angles {
		cosT := math.Cos(a * math.Pi / 180)
		s := lut.Shade(cosT, 0)
		r, g, b := s.ToSRGB(1.6)
		x, _ := left.px(a, 0)
		fillRect(img, int(x), 395, int(x)+3, 425,
			color.RGBA{uint8(r*255 + 0.5), uint8(g*255 + 0.5), uint8(b*255 + 0.5), 255})
	}
	left.polyline(angles, peaks, colPeak)

	// Right: reflectance spectra at 0/20/40/60 degrees.
	right := plotFrame{img: img, x0: 600, y0: 30, x1: 1090, y1: 380, u0: LambdaMin, u1: LambdaMax, v0: 0, v1: 1}
	right.axes()
	for l := 400; l <= 750; l += 50 { // wavelength ticks (nm)
		x, _ := right.px(float64(l), 0)
		drawText(img, int(x)-6, 385, itoa(l), colAxis, 1)
	}
	for _, tick := range []struct {
		v float64
		s string
	}{{0, "0"}, {0.5, "0.5"}, {1, "1"}} { // reflectance ticks
		_, y := right.px(0, tick.v)
		drawText(img, 566, int(y)-2, tick.s, colAxis, 1)
	}
	for k, a := range []float64{0, 20, 40, 60} {
		cosT := math.Cos(a * math.Pi / 180)
		s := lut.Shade(cosT, 0)
		us := make([]float64, Samples)
		vs := make([]float64, Samples)
		for i := 0; i < Samples; i++ {
			us[i] = lambdaAt(i)
			vs[i] = s[i]
		}
		right.polyline(us, vs, colSpect[k])
	}
	return savePNG(path, img)
}
