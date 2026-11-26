package main

import (
	"image"
	"image/color"

	"glyphsdf/internal/atlas"
	"glyphsdf/internal/render"
)

var (
	inkColor   = color.RGBA{R: 24, G: 28, B: 38, A: 255}
	blueInk    = color.RGBA{R: 18, G: 31, B: 128, A: 255}
	paperColor = color.RGBA{R: 246, G: 244, B: 238, A: 255}
	panelColor = color.RGBA{R: 255, G: 255, B: 252, A: 255}
	lineColor  = color.RGBA{R: 210, G: 205, B: 194, A: 255}
	mutedColor = color.RGBA{R: 92, G: 92, B: 88, A: 255}
)

func panel(img *image.RGBA, x, y, w, h int, fill color.RGBA) {
	render.FillRect(img, x, y, w, h, fill)
	render.FillRect(img, x, y, w, 2, lineColor)
	render.FillRect(img, x, y+h-2, w, 2, lineColor)
	render.FillRect(img, x, y, 2, h, lineColor)
	render.FillRect(img, x+w-2, y, 2, h, lineColor)
}

func buildMontage(bundles []bundle) (*image.RGBA, error) {
	const width = 2200
	const height = 1540
	img := render.NewCanvas(width, height, paperColor)

	render.DrawText(img, 44, 34, "GLYPH SDF AND MSDF LAB", 5, inkColor)
	render.DrawText(img, 48, 82, "PURE GO + STDLIB  /  MEDIAN(R,G,B) RECONSTRUCTION  /  RED MARKS VECTOR-INSIDE TIP LOSS", 2, mutedColor)

	drawSmallTextSection(img, 44, 116, width-88, 270)

	cardW := (width - 88 - 28) / 2
	cardH := 548
	positions := []image.Point{
		{X: 44, Y: 414},
		{X: 44 + cardW + 28, Y: 414},
		{X: 44, Y: 414 + cardH + 24},
		{X: 44 + cardW + 28, Y: 414 + cardH + 24},
	}
	for i, b := range bundles {
		if i >= len(positions) {
			break
		}
		drawCard(img, b, positions[i].X, positions[i].Y, cardW, cardH)
	}
	return img, nil
}

func drawSmallTextSection(img *image.RGBA, x, y, w, h int) {
	panel(img, x, y, w, h, panelColor)
	render.DrawText(img, x+24, y+20, "SMALL SIZES / DOWNSAMPLED FROM THE SAME FIELDS", 3, inkColor)
	path := fontPath
	if path == "" {
		path = "/usr/share/fonts/truetype/dejavu/DejaVuSans-Bold.ttf"
	}
	a, err := atlas.New(path, 40, 7)
	if err != nil {
		render.DrawText(img, x+24, y+70, "SYSTEM TTF NOT AVAILABLE; GEOMETRIC SPECIMENS STILL RENDERED", 2, breakRedText())
		return
	}
	render.DrawText(img, x+190, y+66, "SDF", 2, mutedColor)
	render.DrawText(img, x+1080, y+66, "MSDF", 2, mutedColor)
	sizes := []float64{12, 15, 20, 28}
	baselines := []int{104, 146, 196, 248}
	text := "A8R SHARP WM"
	for i, size := range sizes {
		by := y + baselines[i]
		label := formatSize(size)
		render.DrawText(img, x+28, by-18, label, 2, mutedColor)
		a.DrawLine(img, text, render.MethodSDF, float64(x+190), float64(by), size, inkColor, panelColor)
		a.DrawLine(img, text, render.MethodMSDF, float64(x+1080), float64(by), size, inkColor, panelColor)
	}
}

func formatSize(size float64) string {
	if size == float64(int(size)) {
		return itoa(int(size)) + "PX"
	}
	return "1X"
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [12]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func breakRedText() color.RGBA { return color.RGBA{R: 210, G: 30, B: 38, A: 255} }

func drawCard(img *image.RGBA, b bundle, x, y, w, h int) {
	panel(img, x, y, w, h, panelColor)
	render.DrawText(img, x+22, y+20, b.spec.label, 3, inkColor)
	summary := "SDF RED " + itoa(b.report.SDFBrokenCount) + " / MSDF RED " + itoa(b.report.MSDFBrokenCount)
	render.DrawText(img, x+w-360, y+25, summary, 2, mutedColor)

	fieldSize := 150
	fx := x + 24
	sdfFY := y + 78
	msdfFY := y + 78 + fieldSize + 54
	render.DrawText(img, fx, y+54, "SDF FIELD", 2, mutedColor)
	render.DrawNearest(img, b.field.SDFImage(), image.Rect(fx, sdfFY, fx+fieldSize, sdfFY+fieldSize))
	render.DrawText(img, fx, sdfFY+fieldSize+12, "MSDF RGB", 2, mutedColor)
	render.DrawNearest(img, b.field.MSDFImage(), image.Rect(fx, msdfFY, fx+fieldSize, msdfFY+fieldSize))

	covSize := 420
	cx1 := x + 210
	cx2 := x + 210 + covSize + 34
	cy := y + 78
	render.DrawText(img, cx1, y+54, "SDF COVERAGE + RED BREAKS", 2, breakRedText())
	render.DrawText(img, cx2, y+54, "MSDF MEDIAN COVERAGE", 2, color.RGBA{R: 24, G: 130, B: 70, A: 255})
	render.DrawImage(img, b.sdfView, cx1, cy)
	render.DrawImage(img, b.msdfView, cx2, cy)
	_ = covSize
}
