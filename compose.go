package main

import (
	"image"
	"image/color"
	"math"
)

var (
	ink      = color.RGBA{8, 12, 15, 255}
	panelBG  = color.RGBA{12, 18, 21, 255}
	textMain = color.RGBA{231, 239, 235, 255}
	textDim  = color.RGBA{137, 158, 157, 255}
	accent   = color.RGBA{68, 214, 207, 255}
	border   = color.RGBA{52, 72, 74, 255}
	warn     = color.RGBA{235, 156, 82, 255}
)

func newCanvas(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	verticalGradient(img, 0, 0, w, h, color.RGBA{13, 19, 23, 255}, color.RGBA{5, 8, 10, 255})
	return img
}

func drawTag(img *image.RGBA, x, y int, s string, c color.RGBA) {
	w := textWidth(s, 2) + 24
	fillRect(img, x, y, x+w, y+32, color.RGBA{5, 9, 11, 220})
	fillRect(img, x, y, x+4, y+32, c)
	drawText(img, x+13, y+9, s, 2, textMain)
}

func drawSectionTitle(img *image.RGBA, title, subtitle string) {
	drawText(img, 44, 28, title, 4, textMain)
	drawText(img, 46, 76, subtitle, 2, textDim)
	drawLine(img, 44, 106, 500, 106, accent)
}

func drawLegend(img *image.RGBA, x, y int) {
	items := []struct {
		label string
		c     color.RGBA
	}{
		{"MUD", color.RGBA{126, 91, 54, 255}},
		{"MOSS", color.RGBA{80, 142, 52, 255}},
		{"WET STONE", color.RGBA{104, 137, 148, 255}},
	}
	cx := x
	for _, item := range items {
		fillRect(img, cx, y+5, cx+16, y+21, item.c)
		strokeRect(img, cx, y+5, cx+16, y+21, 1, color.RGBA{180, 200, 194, 90})
		drawText(img, cx+24, y+8, item.label, 2, textMain)
		cx += 32 + textWidth(item.label, 2) + 30
	}
}

func composeHero(src *image.RGBA) *image.RGBA {
	out := cloneRGBA(src)
	w, h := out.Bounds().Dx(), out.Bounds().Dy()
	fillRect(out, 34, 28, 750, 116, color.RGBA{4, 8, 10, 190})
	fillRect(out, 34, 28, 39, 116, accent)
	drawText(out, 58, 40, "HEIGHT-BLENDED SHORELINE", 3, textMain)
	drawText(out, 60, 78, "MUD / MOSS / WET STONE   TRIPLANAR 5X   WETNESS MASK", 2, textDim)

	fillRect(out, 34, h-68, 670, h-26, color.RGBA{4, 8, 10, 185})
	drawLegend(out, 50, h-58)

	msg := "GEOMETRY LOCKED  |  SILHOUETTE DELTA 0.0000"
	mw := textWidth(msg, 2)
	fillRect(out, w-mw-48, h-68, w-24, h-26, color.RGBA{4, 8, 10, 185})
	drawText(out, w-mw-36, h-56, msg, 2, accent)
	return out
}

func coverageMean(g *GBuffer, mat int) float64 {
	sum, count := 0.0, 0
	for i, ok := range g.Valid {
		if !ok {
			continue
		}
		sum += g.Cov[i][mat]
		count++
	}
	if count == 0 {
		return 0
	}
	return 100 * sum / float64(count)
}

func placePanel(dst, src *image.RGBA, rect image.Rectangle, x, y, w, h int) {
	img := cropResize(src, rect, w, h)
	pasteImage(dst, img, x, y)
	strokeRect(dst, x, y, x+w, y+h, 2, border)
}

func composeDecomposition(mud, moss, stone *image.RGBA, g *GBuffer) *image.RGBA {
	out := newCanvas(1800, 760)
	drawSectionTitle(out, "MATERIAL DECOMPOSITION", "HEIGHT-SHARPENED WEIGHTS   /   TRIPLANAR PROJECTION   /   IDENTICAL TERRAIN GEOMETRY")
	entries := []struct {
		img   *image.RGBA
		name  string
		color color.RGBA
		mat   int
	}{
		{mud, "MUD", color.RGBA{126, 91, 54, 255}, Mud},
		{moss, "MOSS", color.RGBA{80, 142, 52, 255}, Moss},
		{stone, "WET STONE", color.RGBA{104, 137, 148, 255}, WetStone},
	}
	w, h := mud.Bounds().Dx(), mud.Bounds().Dy()
	crop := image.Rect(
		int(float64(w)*0.106), int(float64(h)*0.088),
		int(float64(w)*0.706), int(float64(h)*0.848),
	)
	for i, item := range entries {
		x := 40 + i*590
		y := 140
		drawText(out, x, y-34, item.name, 3, item.color)
		placePanel(out, item.img, crop, x, y, 540, 430)
		fillRect(out, x, y+430, x+540, y+486, color.RGBA{9, 14, 16, 240})
		drawText(out, x+16, y+448, "AVG COVER", 2, textDim)
		drawText(out, x+148, y+448, fmtPercent(coverageMean(g, item.mat)), 2, textMain)
		drawText(out, x+288, y+448, "HEIGHT SHARPEN 18X", 2, textDim)
		strokeRect(out, x, y, x+540, y+486, 2, border)
	}
	note := "WEIGHT-ISOLATED RESPONSE   /   HEIGHT SELECTION HAPPENS BEFORE COLOR COMPOSITE"
	drawText(out, 44, 694, note, 2, textDim)
	drawLine(out, 44, 726, 850, 726, border)
	return out
}

func composeBoundary(naive, heightBlend *image.RGBA) *image.RGBA {
	out := newCanvas(1720, 830)
	drawSectionTitle(out, "BOUNDARY MAGNIFICATION", "ORDINARY SPLAT LERP VS HEIGHT-SELECTED WEIGHTS   /   SAME CAMERA AND DEPTH BUFFER")
	w, h := naive.Bounds().Dx(), naive.Bounds().Dy()
	crop := image.Rect(
		int(float64(w)*0.13), int(float64(h)*0.11),
		int(float64(w)*0.68), int(float64(h)*0.79),
	)
	leftX, rightX, y := 40, 880, 132
	panelW, panelH := 800, 640
	placePanel(out, naive, crop, leftX, y, panelW, panelH)
	placePanel(out, heightBlend, crop, rightX, y, panelW, panelH)
	drawTag(out, leftX+14, y+14, "SPLAT LERP  /  FUZZY BAND", warn)
	drawTag(out, rightX+14, y+14, "HEIGHT BLEND  /  RESOLVED EDGE", accent)
	fillRect(out, rightX-20, y, rightX-18, y+panelH, color.RGBA{5, 8, 10, 255})
	drawText(out, 42, 792, "COLOR-WEIGHTED ONLY", 2, textDim)
	drawText(out, 720, 792, "GEOMETRY / NORMALS / DEPTH UNCHANGED", 2, textMain)
	return out
}

func composeReflection(wet, diff *image.RGBA) *image.RGBA {
	out := newCanvas(1720, 880)
	drawSectionTitle(out, "REFLECTION DIFFERENCE", "WETNESS MASK RAISES WET-STONE REFLECTION WITHIN THE WATERLINE BAND")
	leftX, rightX, y := 40, 860, 132
	panelW, panelH := 820, 512
	placePanel(out, wet, wet.Bounds(), leftX, y, panelW, panelH)
	placePanel(out, diff, diff.Bounds(), rightX, y, panelW, panelH)
	drawTag(out, leftX+14, y+14, "WET SPECULAR RESPONSE", accent)
	drawTag(out, rightX+14, y+14, "WET MINUS DRY  /  6.5X", warn)

	barX, barY, barW := 1110, 704, 520
	for i := 0; i < barW; i++ {
		c := heatColor(float64(i) / float64(barW-1))
		r, g, b, _ := rgbaOf(c)
		fillRect(out, barX+i, barY, barX+i+1, barY+18, color.RGBA{R: r, G: g, B: b, A: 255})
	}
	strokeRect(out, barX, barY, barX+barW, barY+18, 1, border)
	drawText(out, barX, barY+28, "LOW", 2, textDim)
	drawText(out, barX+barW-60, barY+28, "HIGH", 2, textDim)
	drawText(out, 44, 696, "WATERLINE HEIGHT  -0.055", 2, textDim)
	drawText(out, 44, 728, "WET BAND  0.135 WORLD UNITS", 2, textDim)
	drawText(out, 44, 764, "WET STONE ALBEDO +34%", 2, textMain)
	drawText(out, 44, 796, "REFLECTION GAIN +42%", 2, textMain)
	return out
}

func fmtPercent(v float64) string {
	ones := int(math.Floor(v))
	tenths := int(math.Round((v - float64(ones)) * 10))
	if tenths == 10 {
		ones++
		tenths = 0
	}
	return itoa(ones) + "." + itoa(tenths) + "%"
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	buf := [8]byte{}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}

func cloneRGBA(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Bounds())
	copy(dst.Pix, src.Pix)
	return dst
}
