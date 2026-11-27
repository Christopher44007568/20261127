package main

import "math"

// Canvas is an RGB8 framebuffer with simple 2D drawing helpers.
type Canvas struct {
	W, H   int
	Pixels []uint8
}

func newCanvas(w, h int) *Canvas {
	return &Canvas{W: w, H: h, Pixels: make([]uint8, w*h*3)}
}

func (c *Canvas) set(x, y int, col [3]float64) {
	if x < 0 || y < 0 || x >= c.W || y >= c.H {
		return
	}
	idx := (y*c.W + x) * 3
	c.Pixels[idx+0] = color8(col[0])
	c.Pixels[idx+1] = color8(col[1])
	c.Pixels[idx+2] = color8(col[2])
}

func (c *Canvas) rect(x0, y0, x1, y1 int, col [3]float64) {
	for y := y0; y <= y1; y++ {
		for x := x0; x <= x1; x++ {
			c.set(x, y, col)
		}
	}
}

func (c *Canvas) frame(x0, y0, x1, y1, thick int, col [3]float64) {
	c.rect(x0, y0, x1, y0+thick-1, col)
	c.rect(x0, y1-thick+1, x1, y1, col)
	c.rect(x0, y0, x0+thick-1, y1, col)
	c.rect(x1-thick+1, y0, x1, y1, col)
}

func (c *Canvas) line(x0, y0, x1, y1 int, thick int, col [3]float64) {
	dx := math.Abs(float64(x1 - x0))
	dy := math.Abs(float64(y1 - y0))
	steps := int(math.Max(dx, dy)) + 1
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Round(mix(float64(x0), float64(x1), t)))
		y := int(math.Round(mix(float64(y0), float64(y1), t)))
		for oy := -thick / 2; oy <= thick/2; oy++ {
			for ox := -thick / 2; ox <= thick/2; ox++ {
				c.set(x+ox, y+oy, col)
			}
		}
	}
}

// arrow draws an arrow from (x0,y0) to (x1,y1) with a triangular head.
func (c *Canvas) arrow(x0, y0, x1, y1 int, thick int, col [3]float64) {
	c.line(x0, y0, x1, y1, thick, col)
	ang := math.Atan2(float64(y1-y0), float64(x1-x0))
	head := 22.0
	open := 0.55
	for k := -1; k <= 1; k++ {
		a := ang + math.Pi + float64(k)*open
		hx := x1 + int(head*math.Cos(a))
		hy := y1 + int(head*math.Sin(a))
		c.line(x1, y1, hx, hy, thick, col)
	}
}

// font5x7 covers the glyphs needed by the scene labels.
var font5x7 = map[rune][]string{
	'0': {"01110", "10001", "10011", "10101", "11001", "10001", "01110"},
	'1': {"00100", "01100", "00100", "00100", "00100", "00100", "01110"},
	'2': {"01110", "10001", "00001", "00010", "00100", "01000", "11111"},
	'3': {"11110", "00001", "00001", "01110", "00001", "00001", "11110"},
	'4': {"00010", "00110", "01010", "10010", "11111", "00010", "00010"},
	'5': {"11111", "10000", "11110", "00001", "00001", "10001", "01110"},
	'6': {"00110", "01000", "10000", "11110", "10001", "10001", "01110"},
	'7': {"11111", "00001", "00010", "00100", "01000", "01000", "01000"},
	'8': {"01110", "10001", "10001", "01110", "10001", "10001", "01110"},
	'9': {"01110", "10001", "10001", "01111", "00001", "00010", "01100"},
	'.': {"00000", "00000", "00000", "00000", "00000", "01100", "01100"},
	'-': {"00000", "00000", "00000", "11111", "00000", "00000", "00000"},
	'°': {"01010", "01010", "00100", "00000", "00000", "00000", "00000"},
	' ': {"00000", "00000", "00000", "00000", "00000", "00000", "00000"},
}

// drawText renders a string centered at (cx, top) with a given pixel scale.
func (c *Canvas) drawText(text string, cx, top, scale int, col [3]float64) {
	glyphW := 5*scale + scale
	total := len(text)*glyphW - scale
	x0 := cx - total/2
	for i, r := range text {
		glyph, ok := font5x7[r]
		if !ok {
			glyph = font5x7[' ']
		}
		gx := x0 + i*glyphW
		for gy, row := range glyph {
			for gx2, bit := range row {
				if bit == '1' {
					c.rect(gx+gx2*scale, top+gy*scale,
						gx+gx2*scale+scale-1, top+gy*scale+scale-1, col)
				}
			}
		}
	}
}
