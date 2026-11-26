package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
)

type Canvas struct {
	Img *image.RGBA
	W   int
	H   int
}

func NewCanvas(w, h int, bg RGB) *Canvas {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	c := &Canvas{Img: img, W: w, H: h}
	c.Clear(bg)
	return c
}

func (c *Canvas) Clear(bg RGB) {
	col := bg.RGBA()
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			c.Img.SetRGBA(x, y, col)
		}
	}
}

func (c *Canvas) Set(x, y int, col RGB) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	c.Img.SetRGBA(x, y, col.RGBA())
}

func (c *Canvas) BlendSet(x, y int, col RGB, alpha float64) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H || alpha <= 0 {
		return
	}
	old := c.Img.RGBAAt(x, y)
	r := float64(old.R)/255*(1-alpha) + col.R*alpha
	g := float64(old.G)/255*(1-alpha) + col.G*alpha
	b := float64(old.B)/255*(1-alpha) + col.B*alpha
	c.Set(x, y, RGB{r, g, b})
}

func (c *Canvas) FillRect(x0, y0, w, h int, col RGB) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			c.Set(x, y, col)
		}
	}
}

func (c *Canvas) FrameRect(x0, y0, w, h, thick int, col RGB) {
	c.FillRect(x0, y0, w, thick, col)
	c.FillRect(x0, y0+h-thick, w, thick, col)
	c.FillRect(x0, y0, thick, h, col)
	c.FillRect(x0+w-thick, y0, thick, h, col)
}

func (c *Canvas) Line(x0, y0, x1, y1, thick int, col RGB) {
	dx := math.Abs(float64(x1 - x0))
	dy := math.Abs(float64(y1 - y0))
	steps := int(math.Max(dx, dy))
	if steps == 0 {
		c.FillRect(x0-thick/2, y0-thick/2, thick, thick, col)
		return
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Round(float64(x0) + (float64(x1)-float64(x0))*t))
		y := int(math.Round(float64(y0) + (float64(y1)-float64(y0))*t))
		r := thick / 2
		c.FillRect(x-r, y-r, thick, thick, col)
	}
}

func (c *Canvas) LineFloat(x0, y0, x1, y1 float64, thick int, col RGB) {
	c.Line(int(math.Round(x0)), int(math.Round(y0)), int(math.Round(x1)), int(math.Round(y1)), thick, col)
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
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	return enc.Encode(f, img)
}

func SaveCanvasPNG(path string, c *Canvas) error {
	return SavePNG(path, c.Img)
}

func Gray(v float64) RGB { return RGB{v, v, v} }

func ColorToRGBA(col color.Color) color.RGBA {
	r, g, b, a := col.RGBA()
	return color.RGBA{R: uint8(r >> 8), G: uint8(g >> 8), B: uint8(b >> 8), A: uint8(a >> 8)}
}
