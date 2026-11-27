package main

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

type Canvas struct {
	W, H  int
	Img   *image.RGBA
	Depth []float64
}

func NewCanvas(w, h int) *Canvas {
	return &Canvas{
		W:     w,
		H:     h,
		Img:   image.NewRGBA(image.Rect(0, 0, w, h)),
		Depth: make([]float64, w*h),
	}
}

func (c *Canvas) Clear(col color.RGBA) {
	for i := range c.Depth {
		c.Depth[i] = math.Inf(1)
	}
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			c.Img.SetRGBA(x, y, col)
		}
	}
}

func (c *Canvas) In(x, y int) bool {
	return x >= 0 && y >= 0 && x < c.W && y < c.H
}

func (c *Canvas) Set(x, y int, col color.RGBA) {
	if c.In(x, y) {
		c.Img.SetRGBA(x, y, col)
	}
}

func (c *Canvas) Blend(x, y int, col color.RGBA, alpha float64) {
	if !c.In(x, y) {
		return
	}
	bg := c.Img.RGBAAt(x, y)
	a := clampFloat(alpha*float64(col.A)/255, 0, 1)
	out := color.RGBA{
		R: uint8(clampFloat(float64(bg.R)*(1-a)+float64(col.R)*a, 0, 255)),
		G: uint8(clampFloat(float64(bg.G)*(1-a)+float64(col.G)*a, 0, 255)),
		B: uint8(clampFloat(float64(bg.B)*(1-a)+float64(col.B)*a, 0, 255)),
		A: 255,
	}
	c.Img.SetRGBA(x, y, out)
}

func (c *Canvas) FillRect(x0, y0, w, h int, col color.RGBA) {
	for y := y0; y < y0+h; y++ {
		for x := x0; x < x0+w; x++ {
			c.Set(x, y, col)
		}
	}
}

func (c *Canvas) Rect(x, y, w, h int, col color.RGBA) {
	for x1 := x; x1 < x+w; x1++ {
		c.Set(x1, y, col)
		c.Set(x1, y+h-1, col)
	}
	for y1 := y; y1 < y+h; y1++ {
		c.Set(x, y1, col)
		c.Set(x+w-1, y1, col)
	}
}

func (c *Canvas) Line(x0, y0, x1, y1 int, col color.RGBA) {
	alpha := float64(col.A) / 255
	dx := math.Abs(float64(x1 - x0))
	dy := -math.Abs(float64(y1 - y0))
	sx, sy := -1, -1
	if x0 < x1 {
		sx = 1
	}
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		if alpha >= 0.999 {
			c.Set(x0, y0, col)
		} else {
			c.Blend(x0, y0, col, alpha)
		}
		if x0 == x1 && y0 == y1 {
			break
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

func (c *Canvas) ThickLine(x0, y0, x1, y1, thick int, col color.RGBA) {
	r := thick / 2
	steps := int(math.Max(math.Abs(float64(x1-x0)), math.Abs(float64(y1-y0)))) + 1
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(lerpFloat(float64(x0), float64(x1), t))
		y := int(lerpFloat(float64(y0), float64(y1), t))
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if dx*dx+dy*dy <= r*r+r {
					c.Set(x+dx, y+dy, col)
				}
			}
		}
	}
}

func SavePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		return err
	}
	return f.Close()
}

func heatRamp(t float64) color.RGBA {
	stops := []struct {
		at  float64
		col [3]float64
	}{
		{0.00, [3]float64{38, 198, 120}},
		{0.35, [3]float64{235, 224, 75}},
		{0.70, [3]float64{245, 132, 44}},
		{1.00, [3]float64{214, 48, 49}},
	}
	t = clampFloat(t, 0, 1)
	for i := 0; i < len(stops)-1; i++ {
		a := stops[i]
		b := stops[i+1]
		if t <= b.at {
			f := (t - a.at) / (b.at - a.at)
			return color.RGBA{
				R: uint8(lerpFloat(a.col[0], b.col[0], f)),
				G: uint8(lerpFloat(a.col[1], b.col[1], f)),
				B: uint8(lerpFloat(a.col[2], b.col[2], f)),
				A: 255,
			}
		}
	}
	last := stops[len(stops)-1].col
	return color.RGBA{R: uint8(last[0]), G: uint8(last[1]), B: uint8(last[2]), A: 255}
}

func rgba(r, g, b uint8, a ...uint8) color.RGBA {
	aa := uint8(255)
	if len(a) > 0 {
		aa = a[0]
	}
	return color.RGBA{R: r, G: g, B: b, A: aa}
}
