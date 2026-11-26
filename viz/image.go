// Package viz turns coverage buffers into PNG artifacts: annotated coverage
// maps, gridded edge close-ups, error heatmaps, sweep charts and stain
// panels. It only depends on the Go standard library.
package viz

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"coverlab/cover"
)

// White is the standard opaque background.
var White = cover.RGB{R: 1, G: 1, B: 1}

// SavePNG writes img to path, creating parent directories.
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

// ColorImage converts a 0..255 RGB buffer to an opaque NRGBA image.
func ColorImage(w, h int, rgb []cover.RGB) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := rgb[y*w+x]
			img.SetNRGBA(x, y, color.NRGBA{
				R: clamp8(c.R), G: clamp8(c.G), B: clamp8(c.B), A: 255,
			})
		}
	}
	return img
}

func clamp8(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(math.Round(v))
}

// Fill fills the whole image with c.
func Fill(img *image.NRGBA, c color.NRGBA) {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
}

// DrawCellGrid overlays 1-pixel grid lines for a cols x rows grid where each
// source cell is cell x cell output pixels, offset by (ox,oy).
func DrawCellGrid(img *image.NRGBA, ox, oy, cols, rows, cell int, c color.NRGBA) {
	for i := 0; i <= cols; i++ {
		x := ox + i*cell
		drawVLine(img, x, oy, oy+rows*cell, c)
	}
	for j := 0; j <= rows; j++ {
		y := oy + j*cell
		drawHLine(img, ox, ox+cols*cell, y, c)
	}
}

func drawVLine(img *image.NRGBA, x, y0, y1 int, c color.NRGBA) {
	for y := y0; y < y1; y++ {
		blendSet(img, x, y, c)
	}
}

func drawHLine(img *image.NRGBA, x0, x1, y int, c color.NRGBA) {
	for x := x0; x < x1; x++ {
		blendSet(img, x, y, c)
	}
}

// DrawRectOutline strokes a rectangle in output pixel coordinates.
func DrawRectOutline(img *image.NRGBA, r image.Rectangle, thick int, c color.NRGBA) {
	for t := 0; t < thick; t++ {
		drawHLine(img, r.Min.X, r.Max.X, r.Min.Y+t, c)
		drawHLine(img, r.Min.X, r.Max.X, r.Max.Y-1-t, c)
		drawVLine(img, r.Min.X+t, r.Min.Y, r.Max.Y, c)
		drawVLine(img, r.Max.X-1-t, r.Min.Y, r.Max.Y, c)
	}
}

// blendSet alpha-blends c over the existing pixel.
func blendSet(img *image.NRGBA, x, y int, c color.NRGBA) {
	if !image.Pt(x, y).In(img.Bounds()) {
		return
	}
	old := img.NRGBAAt(x, y)
	a := float64(c.A) / 255
	ia := 1 - a
	img.SetNRGBA(x, y, color.NRGBA{
		R: uint8(float64(c.R)*a + float64(old.R)*ia),
		G: uint8(float64(c.G)*a + float64(old.G)*ia),
		B: uint8(float64(c.B)*a + float64(old.B)*ia),
		A: 255,
	})
}

// Viridis maps t in 0..1 to the viridis colormap (piecewise linear over the
// canonical stops).
func Viridis(t float64) color.NRGBA {
	stops := []struct {
		t   float64
		rgb [3]float64
	}{
		{0.00, [3]float64{68, 1, 84}},
		{0.25, [3]float64{59, 82, 139}},
		{0.50, [3]float64{33, 145, 140}},
		{0.75, [3]float64{94, 201, 98}},
		{1.00, [3]float64{253, 231, 37}},
	}
	t = math.Max(0, math.Min(1, t))
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].t {
			a := stops[i-1]
			b := stops[i]
			u := (t - a.t) / (b.t - a.t)
			return color.NRGBA{
				R: clamp8(a.rgb[0] + u*(b.rgb[0]-a.rgb[0])),
				G: clamp8(a.rgb[1] + u*(b.rgb[1]-a.rgb[1])),
				B: clamp8(a.rgb[2] + u*(b.rgb[2]-a.rgb[2])),
				A: 255,
			}
		}
	}
	return color.NRGBA{R: 253, G: 231, B: 37, A: 255}
}

// GrayCoverage maps coverage 0..1 to a black->white gray.
func GrayCoverage(c float64) color.NRGBA {
	v := clamp8(math.Max(0, math.Min(1, c)) * 255)
	return color.NRGBA{R: v, G: v, B: v, A: 255}
}
