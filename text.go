package main

import "strings"

func TextWidth(s string, scale int) int {
	if scale < 1 {
		scale = 1
	}
	return len(s) * 6 * scale
}

func (c *Canvas) drawGlyph(ch rune, x, y, scale int, col Vec3, alpha float64) {
	glyph, ok := font5x7[ch]
	if !ok {
		glyph = font5x7['?']
	}
	for row := 0; row < 7; row++ {
		line := glyph[row]
		for colIndex := 0; colIndex < 5; colIndex++ {
			if line[colIndex] != '#' {
				continue
			}
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					c.Blend(x+colIndex*scale+dx, y+row*scale+dy, col, alpha)
				}
			}
		}
	}
}

type TextAlign int

const (
	TextLeft TextAlign = iota
	TextCenter
	TextRight
)

func (c *Canvas) Text(s string, x, y, scale int, col Vec3, align TextAlign) {
	s = strings.ToUpper(s)
	width := TextWidth(s, scale)
	if align == TextCenter {
		x -= width / 2
	} else if align == TextRight {
		x -= width
	}
	shadow := hexLinear(0x06101f)
	for i, ch := range s {
		if ch == ' ' {
			continue
		}
		gx := x + i*6*scale
		off := maxInt(2, scale/2)
		for _, dx := range []int{-off, 0, off} {
			for _, dy := range []int{-off, 0, off} {
				if dx == 0 && dy == 0 {
					continue
				}
				c.drawGlyph(ch, gx+dx, y+dy, scale, shadow, 0.82)
			}
		}
		c.drawGlyph(ch, gx, y, scale, col, 0.96)
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func (c *Canvas) TextWorld(s string, p Vec2, scale int, col Vec3, align TextAlign) {
	x, y := c.WorldToPixel(p)
	c.Text(s, int(x), int(y), scale, col, align)
}

func (c *Canvas) RoundPanel(x0, y0, x1, y1, radius int, col Vec3, alpha float64) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			qx := 0.0
			if x < x0+radius {
				qx = float64(x0 + radius - x - 1)
			} else if x >= x1-radius {
				qx = float64(x - (x1 - radius))
			}
			qy := 0.0
			if y < y0+radius {
				qy = float64(y0 + radius - y - 1)
			} else if y >= y1-radius {
				qy = float64(y - (y1 - radius))
			}
			if qx*qx+qy*qy <= float64(radius*radius) {
				c.Blend(x, y, col, alpha)
			}
		}
	}
}
