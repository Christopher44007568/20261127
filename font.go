package main

import (
	"image"
	"image/color"
	"strconv"
)

func itoa(v int) string { return strconv.Itoa(v) }

// Tiny 3x5 bitmap font for plot tick labels (digits and a few symbols).
var glyphs = map[rune][5]uint8{
	'0': {0b111, 0b101, 0b101, 0b101, 0b111},
	'1': {0b010, 0b110, 0b010, 0b010, 0b111},
	'2': {0b111, 0b001, 0b111, 0b100, 0b111},
	'3': {0b111, 0b001, 0b111, 0b001, 0b111},
	'4': {0b101, 0b101, 0b111, 0b001, 0b001},
	'5': {0b111, 0b100, 0b111, 0b001, 0b111},
	'6': {0b111, 0b100, 0b111, 0b101, 0b111},
	'7': {0b111, 0b001, 0b010, 0b010, 0b010},
	'8': {0b111, 0b101, 0b111, 0b101, 0b111},
	'9': {0b111, 0b101, 0b111, 0b001, 0b111},
	'.': {0b000, 0b000, 0b000, 0b000, 0b010},
	'-': {0b000, 0b000, 0b111, 0b000, 0b000},
	'n': {0b000, 0b110, 0b101, 0b101, 0b101},
	'm': {0b000, 0b101, 0b111, 0b101, 0b101},
	'd': {0b001, 0b111, 0b101, 0b101, 0b111}, // used for "deg"
	'R': {0b110, 0b101, 0b110, 0b101, 0b101},
}

// drawText renders s at pixel (x, y) with 1px glyph spacing, scaled by scale.
func drawText(img *image.RGBA, x, y int, s string, c color.RGBA, scale int) {
	for _, r := range s {
		g, ok := glyphs[r]
		if !ok {
			x += 4 * scale
			continue
		}
		for row := 0; row < 5; row++ {
			for col := 0; col < 3; col++ {
				if g[row]>>(2-col)&1 == 1 {
					fillRect(img, x+col*scale, y+row*scale,
						x+(col+1)*scale-1, y+(row+1)*scale-1, c)
				}
			}
		}
		x += 4 * scale
	}
}
