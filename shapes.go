package main

import (
	"image"
	"image/color"
)

func borderRect(dst *image.RGBA, x, y, w, h int, col color.RGBA) {
	fillRect(dst, x, y, w, 1, col)
	fillRect(dst, x, y+h-1, w, 1, col)
	fillRect(dst, x, y, 1, h, col)
	fillRect(dst, x+w-1, y, 1, h, col)
}
