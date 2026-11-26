package main

import (
	"image"
	"image/color"
)

// Tiny hand-authored 5x7 bitmap font so diff images can carry exact object IDs
// without third-party font packages. '#' marks a lit pixel.

var glyphUpper = map[rune][7]string{
	'A': {"..#..", ".#.#.", "#...#", "#...#", "#####", "#...#", "#...#"},
	'B': {"####.", "#...#", "#...#", "####.", "#...#", "#...#", "####."},
	'C': {".###.", "#...#", "#....", "#....", "#....", "#...#", ".###."},
	'D': {"####.", "#...#", "#...#", "#...#", "#...#", "#...#", "####."},
	'E': {"#####", "#....", "#....", "####.", "#....", "#....", "#####"},
	'F': {"#####", "#....", "#....", "####.", "#....", "#....", "#...."},
	'G': {".###.", "#...#", "#....", "#.###", "#...#", "#...#", ".###."},
	'H': {"#...#", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"},
	'I': {"#####", "..#..", "..#..", "..#..", "..#..", "..#..", "#####"},
	'J': {"..###", "...#.", "...#.", "...#.", "...#.", "#..#.", ".##.."},
	'K': {"#...#", "#..#.", "#.#..", "##...", "#.#..", "#..#.", "#...#"},
	'L': {"#....", "#....", "#....", "#....", "#....", "#....", "#####"},
	'M': {"#...#", "##.##", "#.#.#", "#.#.#", "#...#", "#...#", "#...#"},
	'N': {"#...#", "##..#", "#.#.#", "#..##", "#...#", "#...#", "#...#"},
	'O': {".###.", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'P': {"####.", "#...#", "#...#", "####.", "#....", "#....", "#...."},
	'Q': {".###.", "#...#", "#...#", "#...#", "#.#.#", "#..#.", ".##.#"},
	'R': {"####.", "#...#", "#...#", "####.", "#.#..", "#..#.", "#...#"},
	'S': {".###.", "#...#", "#....", ".###.", "....#", "#...#", ".###."},
	'T': {"#####", "..#..", "..#..", "..#..", "..#..", "..#..", "..#.."},
	'U': {"#...#", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'V': {"#...#", "#...#", "#...#", "#...#", "#...#", ".#.#.", "..#.."},
	'W': {"#...#", "#...#", "#...#", "#.#.#", "#.#.#", "##.##", "#...#"},
	'X': {"#...#", "#...#", ".#.#.", "..#..", ".#.#.", "#...#", "#...#"},
	'Y': {"#...#", "#...#", ".#.#.", "..#..", "..#..", "..#..", "..#.."},
	'Z': {"#####", "....#", "...#.", "..#..", ".#...", "#....", "#####"},
}

var glyphDigit = map[rune][7]string{
	'0': {".###.", "#...#", "#..##", "#.#.#", "##..#", "#...#", ".###."},
	'1': {"..#..", ".##..", "..#..", "..#..", "..#..", "..#..", ".###."},
	'2': {".###.", "#...#", "....#", "..##.", ".#...", "#....", "#####"},
	'3': {"####.", "....#", "....#", ".###.", "....#", "....#", "####."},
	'4': {"...#.", "..##.", ".#.#.", "#..#.", "#####", "...#.", "...#."},
	'5': {"#####", "#....", "####.", "....#", "....#", "#...#", ".###."},
	'6': {"..##.", ".#...", "#....", "####.", "#...#", "#...#", ".###."},
	'7': {"#####", "....#", "...#.", "..#..", ".#...", ".#...", ".#..."},
	'8': {".###.", "#...#", "#...#", ".###.", "#...#", "#...#", ".###."},
	'9': {".###.", "#...#", "#...#", "####.", "....#", "...#.", ".##.."},
}

func glyphFor(r rune) [7]string {
	if g, ok := glyphLower[r]; ok {
		return g
	}
	if g, ok := glyphUpper[r]; ok {
		return g
	}
	if g, ok := glyphDigit[r]; ok {
		return g
	}
	if g, ok := glyphPunct[r]; ok {
		return g
	}
	return [7]string{".....", ".....", "..#..", "..#..", ".....", "..#..", "....."}
}

// TextWidth returns the rendered width in pixels for the given scale.
func TextWidth(s string, scale int) int {
	n := len([]rune(s))
	if n == 0 {
		return 0
	}
	return n*6*scale - scale
}

// DrawText draws s with the bitmap font at top-left (x,y).
func DrawText(img *image.RGBA, x, y int, s string, scale int, c color.RGBA) {
	if scale < 1 {
		scale = 1
	}
	cx := x
	for _, r := range s {
		g := glyphFor(r)
		for row := 0; row < 7; row++ {
			for col := 0; col < 5; col++ {
				if g[row][col] == '#' {
					fillRect(img, cx+col*scale, y+row*scale, scale, scale, c)
				}
			}
		}
		cx += 6 * scale
	}
}

// DrawTextPlate draws text on a filled rounded-ish rectangular plate and
// returns the plate bounds.
func DrawTextPlate(img *image.RGBA, x, y int, s string, scale int, fg, plate color.RGBA, pad int) image.Rectangle {
	tw := TextWidth(s, scale)
	th := 7 * scale
	r := image.Rect(x, y, x+tw+pad*2, y+th+pad*2)
	fillRect(img, r.Min.X, r.Min.Y, r.Dx(), r.Dy(), plate)
	DrawText(img, x+pad, y+pad, s, scale, fg)
	return r
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.RGBA) {
	b := img.Bounds()
	x0 := clampInt(x, b.Min.X, b.Max.X)
	y0 := clampInt(y, b.Min.Y, b.Max.Y)
	x1 := clampInt(x+w, b.Min.X, b.Max.X)
	y1 := clampInt(y+h, b.Min.Y, b.Max.Y)
	for yy := y0; yy < y1; yy++ {
		o := img.PixOffset(x0, yy)
		for xx := x0; xx < x1; xx++ {
			i := o + (xx-x0)*4
			img.Pix[i] = c.R
			img.Pix[i+1] = c.G
			img.Pix[i+2] = c.B
			img.Pix[i+3] = c.A
		}
	}
}

// strokeRect draws a 1+px rectangle outline.
func strokeRect(img *image.RGBA, x, y, w, h, thick int, c color.RGBA) {
	fillRect(img, x, y, w, thick, c)
	fillRect(img, x, y+h-thick, w, thick, c)
	fillRect(img, x, y, thick, h, c)
	fillRect(img, x+w-thick, y, thick, h, c)
}

func drawLine(img *image.RGBA, x0, y0, x1, y1, thick int, c color.RGBA) {
	dx := absInt(x1 - x0)
	dy := -absInt(y1 - y0)
	sx, sy := 1, 1
	if x0 > x1 {
		sx = -1
	}
	if y0 > y1 {
		sy = -1
	}
	err := dx + dy
	for {
		fillRect(img, x0-thick/2, y0-thick/2, thick, thick, c)
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

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
