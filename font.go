package main

import (
	"image/color"
)

var digitGlyphs = [10][5]byte{
	{7, 5, 5, 5, 7}, // 0
	{2, 6, 2, 2, 7}, // 1
	{7, 1, 7, 4, 7}, // 2
	{7, 1, 7, 1, 7}, // 3
	{5, 5, 7, 1, 1}, // 4
	{7, 4, 7, 1, 7}, // 5
	{7, 4, 7, 5, 7}, // 6
	{7, 1, 2, 2, 2}, // 7
	{7, 5, 7, 5, 7}, // 8
	{7, 5, 7, 1, 7}, // 9
}

func drawTinyText(img interface {
	SetRGBA(x, y int, c color.RGBA)
}, text string, x, y, pixel int, col color.RGBA) {
	if pixel <= 0 {
		pixel = 1
	}
	cursor := x
	for _, r := range text {
		if r >= '0' && r <= '9' {
			glyph := digitGlyphs[r-'0']
			for row, bits := range glyph {
				for column := 0; column < 3; column++ {
					if bits&(1<<(2-column)) != 0 {
						fillPixelBlock(img, cursor+column*pixel, y+row*pixel, pixel, col)
					}
				}
			}
		}
		cursor += 4 * pixel
	}
}

func fillPixelBlock(img interface {
	SetRGBA(x, y int, c color.RGBA)
}, x, y, size int, col color.RGBA) {
	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			img.SetRGBA(x+dx, y+dy, col)
		}
	}
}

func drawLabelHalo(c *canvas, text string, x, y, pixel int) {
	black := color.RGBA{R: 0, G: 0, B: 0, A: 230}
	white := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for _, dx := range []int{-pixel, 0, pixel} {
		for _, dy := range []int{-pixel, 0, pixel} {
			if dx != 0 || dy != 0 {
				drawTinyText(c.img, text, x+dx, y+dy, pixel, black)
			}
		}
	}
	drawTinyText(c.img, text, x, y, pixel, white)
}
