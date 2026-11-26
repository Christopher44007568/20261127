// Package render renders the thin-film visualizations (thickness ramp,
// reflectance spectra and shaded metal spheres) onto in-memory RGBA images
// and encodes them as PNG using only the Go standard library.
package render

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"math"
)

// Canvas is a simple RGB image buffer with 2D drawing primitives.
type Canvas struct {
	W, H int
	px   []uint8 // 3 bytes per pixel, RGB
}

// NewCanvas allocates an RGB canvas filled with the given color.
func NewCanvas(w, h int, fill RGB) *Canvas {
	c := &Canvas{W: w, H: h, px: make([]uint8, w*h*3)}
	c.Clear(fill)
	return c
}

// RGB is an 8-bit color triple.
type RGB struct{ R, G, B uint8 }

func rgb(r, g, b float64) RGB {
	clamp := func(v float64) uint8 {
		if v <= 0 {
			return 0
		}
		if v >= 255 {
			return 255
		}
		return uint8(math.Round(v))
	}
	return RGB{clamp(r), clamp(g), clamp(b)}
}

// Clear fills the whole canvas.
func (c *Canvas) Clear(col RGB) {
	for i := 0; i < c.W*c.H; i++ {
		c.px[i*3] = col.R
		c.px[i*3+1] = col.G
		c.px[i*3+2] = col.B
	}
}

// Set writes one pixel, ignoring out-of-bounds coordinates.
func (c *Canvas) Set(x, y int, col RGB) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	off := (y*c.W + x) * 3
	c.px[off] = col.R
	c.px[off+1] = col.G
	c.px[off+2] = col.B
}

// At reads one pixel; out-of-bounds reads return black.
func (c *Canvas) At(x, y int) RGB {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return RGB{}
	}
	off := (y*c.W + x) * 3
	return RGB{c.px[off], c.px[off+1], c.px[off+2]}
}

// FillRect draws a solid axis-aligned rectangle.
func (c *Canvas) FillRect(x0, y0, x1, y1 int, col RGB) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			c.Set(x, y, col)
		}
	}
}

// Line draws a 1-pixel line with the Bresenham algorithm.
func (c *Canvas) Line(x0, y0, x1, y1 int, col RGB) {
	dx := abs(x1 - x0)
	dy := -abs(y1 - y0)
	sx := 1
	if x0 > x1 {
		sx = -1
	}
	sy := 1
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		c.Set(x0, y0, col)
		if x0 == x1 && y0 == y1 {
			return
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

// Polyline connects a sequence of points.
func (c *Canvas) Polyline(pts [][2]int, col RGB) {
	for i := 1; i < len(pts); i++ {
		c.Line(pts[i-1][0], pts[i-1][1], pts[i][0], pts[i][1], col)
	}
}

// Blend alpha-composites col over the existing pixel (0..1 alpha).
func (c *Canvas) Blend(x, y int, col RGB, alpha float64) {
	bg := c.At(x, y)
	c.Set(x, y, rgb(
		float64(bg.R)*(1-alpha)+float64(col.R)*alpha,
		float64(bg.G)*(1-alpha)+float64(col.G)*alpha,
		float64(bg.B)*(1-alpha)+float64(col.B)*alpha,
	))
}

// EncodePNG writes the canvas as an 8-bit non-interlaced PNG.
func (c *Canvas) EncodePNG(w io.Writer) error {
	img := image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			col := c.At(x, y)
			img.SetRGBA(x, y, color.RGBA{col.R, col.G, col.B, 255})
		}
	}
	return png.Encode(w, img)
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
