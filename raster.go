package main

import (
	"image"
	"image/color"
	"image/png"
	"io"
)

type RGBA [4]uint8

// Canvas is a minimal RGBA raster supporting the primitives the reports need.
type Canvas struct {
	W, H int
	Pix  []RGBA
}

func NewCanvas(w, h int, bg RGBA) *Canvas {
	c := &Canvas{W: w, H: h, Pix: make([]RGBA, w*h)}
	for i := range c.Pix {
		c.Pix[i] = bg
	}
	return c
}

func (c *Canvas) Set(x, y int, col RGBA) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	c.Pix[y*c.W+x] = col
}

func (c *Canvas) FillRect(x, y, w, h int, col RGBA) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			c.Set(xx, yy, col)
		}
	}
}

func (c *Canvas) Line(x0, y0, x1, y1 int, col RGBA) {
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
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

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func (c *Canvas) Circle(cx, cy, r int, col RGBA) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				c.Set(cx+x, cy+y, col)
			}
		}
	}
}

func (c *Canvas) Text(x, y int, s string, col RGBA) {
	for i := 0; i < len(s); i++ {
		glyph, ok := font5x7[s[i]]
		if !ok {
			glyph = font5x7[' ']
		}
		for gy := 0; gy < 7; gy++ {
			for gx := 0; gx < 5; gx++ {
				if glyph[gy][gx] == '1' {
					c.Set(x+i*6+gx, y+gy, col)
				}
			}
		}
	}
}

func (c *Canvas) EncodePNG(w io.Writer) error {
	img := image.NewRGBA(image.Rect(0, 0, c.W, c.H))
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			p := c.Pix[y*c.W+x]
			img.SetRGBA(x, y, color.RGBA{p[0], p[1], p[2], p[3]})
		}
	}
	return png.Encode(w, img)
}
