package main

import "image/color"

// FrameBuffer is a CPU RGBA canvas with software rasterisation primitives.
type FrameBuffer struct {
	W, H int
	Pix  []uint8
}

func NewFrameBuffer(w, h int) *FrameBuffer {
	return &FrameBuffer{W: w, H: h, Pix: make([]uint8, w*h*4)}
}

func (fb *FrameBuffer) set(x, y int, c color.Color) {
	if x < 0 || y < 0 || x >= fb.W || y >= fb.H {
		return
	}
	r, g, b, a := c.RGBA()
	i := (y*fb.W + x) * 4
	af := float64(a) / 0xffff
	fb.Pix[i] = uint8(float64(r>>8) * af)
	fb.Pix[i+1] = uint8(float64(g>>8) * af)
	fb.Pix[i+2] = uint8(float64(b>>8) * af)
	fb.Pix[i+3] = 0xff
}

// Fill paints the whole canvas.
func (fb *FrameBuffer) Fill(c color.Color) {
	r, g, b, _ := c.RGBA()
	for i := 0; i < len(fb.Pix); i += 4 {
		fb.Pix[i] = uint8(r >> 8)
		fb.Pix[i+1] = uint8(g >> 8)
		fb.Pix[i+2] = uint8(b >> 8)
		fb.Pix[i+3] = 0xff
	}
}

// FillRect paints an axis-aligned rectangle.
func (fb *FrameBuffer) FillRect(x0, y0, x1, y1 int, c color.Color) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			fb.set(x, y, c)
		}
	}
}

// Line rasterises a 1-pixel segment (Bresenham).
func (fb *FrameBuffer) Line(x0, y0, x1, y1 int, c color.Color) {
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
		fb.set(x0, y0, c)
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

// Circle fills a disc of the given radius.
func (fb *FrameBuffer) Circle(cx, cy, r int, c color.Color) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				fb.set(cx+x, cy+y, c)
			}
		}
	}
}

// Ring draws a hollow circle.
func (fb *FrameBuffer) Ring(cx, cy, r int, c color.Color) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			d2 := x*x + y*y
			if d2 <= r*r && d2 >= (r-1)*(r-1) {
				fb.set(cx+x, cy+y, c)
			}
		}
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
