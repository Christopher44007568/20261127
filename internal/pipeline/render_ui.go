package pipeline

func renderUI() *UIImage {
	layer := &UIImage{Width: Width, Height: Height, Pixels: make([]UIPixel, Width*Height)}

	panel := uiPixel(6, 12, 18, 205)
	fillRect(layer, 20, Height-62, 326, Height-18, panel)
	fillRect(layer, 20, Height-62, 326, Height-59, uiPixel(40, 222, 218, 230))
	fillRect(layer, 20, Height-62, 24, Height-18, uiPixel(40, 222, 218, 230))
	drawText(layer, 38, Height-51, "SAME FRAME", 2, uiPixel(224, 239, 246, 255))
	drawText(layer, 38, Height-35, "UI", 2, uiPixel(40, 222, 218, 255))

	swatches := []UIPixel{
		uiPixel(241, 95, 39, 255),
		uiPixel(31, 157, 168, 255),
		uiPixel(235, 205, 116, 255),
	}
	for i, color := range swatches {
		x := 190 + i*42
		fillRect(layer, x, Height-47, x+30, Height-31, uiPixel(2, 5, 10, 210))
		fillRect(layer, x+3, Height-44, x+27, Height-34, color)
	}

	badgeX, badgeY := Width-76, 18
	fillRect(layer, badgeX, badgeY, badgeX+56, badgeY+34, uiPixel(4, 9, 14, 190))
	fillRect(layer, badgeX, badgeY, badgeX+2, badgeY+34, uiPixel(241, 95, 39, 235))
	drawText(layer, badgeX+15, badgeY+10, "UI", 2, uiPixel(241, 95, 39, 255))

	fillRect(layer, Width-29, 66, Width-27, 96, uiPixel(224, 239, 246, 145))
	fillRect(layer, Width-44, 80, Width-12, 82, uiPixel(224, 239, 246, 145))
	return layer
}

func uiPixel(r, g, b, a uint8) UIPixel {
	return UIPixel{R: r, G: g, B: b, A: a}
}

func fillRect(layer *UIImage, x0, y0, x1, y1 int, color UIPixel) {
	x0 = maxInt(0, x0)
	y0 = maxInt(0, y0)
	x1 = minInt(layer.Width, x1)
	y1 = minInt(layer.Height, y1)
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			index := y*layer.Width + x
			layer.Pixels[index] = blendUI(layer.Pixels[index], color)
		}
	}
}

func blendUI(dst, src UIPixel) UIPixel {
	sa := float64(src.A) / 255
	da := float64(dst.A) / 255
	outA := sa + da*(1-sa)
	if outA <= 0 {
		return UIPixel{}
	}
	channel := func(s, d uint8) uint8 {
		value := (float64(s)*sa + float64(d)*da*(1-sa)) / outA
		return uint8(mathRound(clamp(value, 0, 255)))
	}
	return UIPixel{
		R: channel(src.R, dst.R),
		G: channel(src.G, dst.G),
		B: channel(src.B, dst.B),
		A: uint8(mathRound(outA * 255)),
	}
}

func drawText(layer *UIImage, x, y int, text string, scale int, color UIPixel) {
	cursor := x
	for _, letter := range text {
		glyph, ok := font5x7[letter]
		if !ok {
			cursor += 6 * scale
			continue
		}
		for row, bits := range glyph {
			for column, bit := range bits {
				if bit != '1' {
					continue
				}
				px := cursor + column*scale
				py := y + row*scale
				fillRect(layer, px, py, px+scale, py+scale, color)
			}
		}
		cursor += 6 * scale
	}
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

var font5x7 = map[rune][]string{
	' ': {"00000", "00000", "00000", "00000", "00000", "00000", "00000"},
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'E': {"11111", "10000", "10000", "11110", "10000", "10000", "11111"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'I': {"11111", "00100", "00100", "00100", "00100", "00100", "11111"},
	'M': {"10001", "11011", "10101", "10101", "10001", "10001", "10001"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'S': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	'U': {"10001", "10001", "10001", "10001", "10001", "10001", "01110"},
}
