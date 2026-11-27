package flakepaint

import (
	"image"
	"image/color"
)

var (
	uiWhite  = color.RGBA{226, 240, 241, 255}
	uiDim    = color.RGBA{122, 156, 168, 255}
	uiAccent = color.RGBA{95, 228, 211, 255}
	uiWarm   = color.RGBA{245, 189, 108, 255}
)

func overlayPlate(img *image.RGBA, x0, y0, x1, y1 int) {
	fillRect(img, x0, y0, x1, y1, color.RGBA{2, 7, 10, 150})
	drawLine(img, x0, y0, x1-1, y0, color.RGBA{120, 190, 198, 75})
}

func drawCornerMarks(img *image.RGBA) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	m := 28
	l := 22
	c := color.RGBA{125, 205, 207, 105}
	drawLine(img, m, m, m+l, m, c)
	drawLine(img, m, m, m, m+l, c)
	drawLine(img, w-m, m, w-m-l, m, c)
	drawLine(img, w-m, m, w-m, m+l, c)
	drawLine(img, m, h-m, m+l, h-m, c)
	drawLine(img, m, h-m, m, h-m-l, c)
	drawLine(img, w-m, h-m, w-m-l, h-m, c)
	drawLine(img, w-m, h-m, w-m, h-m-l, c)
}

func drawScaleBar(img *image.RGBA, x, y, length int, label string) {
	drawLine(img, x, y, x+length, y, color.RGBA{225, 240, 240, 210})
	drawLine(img, x, y-5, x, y+5, color.RGBA{225, 240, 240, 210})
	drawLine(img, x+length, y-5, x+length, y+5, color.RGBA{225, 240, 240, 210})
	drawText(img, x, y+12, label, 1, uiDim)
}

func DecorateNear(img *image.RGBA, title string) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	overlayPlate(img, 48, 48, 360, 124)
	drawTextShadow(img, 64, 62, "01 / NEAR", 1, uiAccent)
	drawTextShadow(img, 64, 80, title, 2, uiWhite)
	drawText(img, 64, 106, "LAYERED HEMISPHERE NORMAL", 1, uiDim)

	overlayPlate(img, w-320, 48, w-48, 108)
	drawText(img, w-304, 62, "DETERMINISTIC HASH FIELD", 1, uiAccent)
	drawText(img, w-304, 82, "FOUR OVERLAPPING SCALES", 1, uiDim)

	overlayPlate(img, 48, h-138, 430, h-48)
	overlayPlate(img, w-342, h-108, w-48, h-54)
	drawScaleBar(img, 64, h-70, 164, "100 MM / PANEL SPACE")
	drawText(img, 64, h-124, "NORMAL + ROUGHNESS + SPECULAR LOBE", 1, uiWhite)
	drawText(img, 64, h-108, "1.25 / 2.05 / 3.30 / 5.30 MM CELLS", 1, uiDim)
	drawText(img, w-340, h-70, "CLEARCOAT / METALLIC BASE", 1, uiWarm)
	drawText(img, w-340, h-88, "MICROFACET RESPONSE", 1, uiDim)
	drawCornerMarks(img)
}

func DecorateFar(img *image.RGBA, title string) {
	w, h := img.Rect.Dx(), img.Rect.Dy()
	overlayPlate(img, 48, 48, 370, 124)
	drawTextShadow(img, 64, 62, "02 / FAR", 1, uiAccent)
	drawTextShadow(img, 64, 80, title, 2, uiWhite)
	drawText(img, 64, 106, "LONG LENS / AGGREGATE MATERIAL", 1, uiDim)

	overlayPlate(img, w-360, 48, w-48, 108)
	drawText(img, w-344, 62, "LOD: CONTINUOUS SPECULAR BAND", 1, uiAccent)
	drawText(img, w-344, 82, "NO PER-FRAME RANDOMIZATION", 1, uiDim)

	overlayPlate(img, 48, h-138, 430, h-48)
	overlayPlate(img, w-292, h-108, w-48, h-54)
	drawText(img, 64, h-126, "UNRESOLVED FLAKE VARIANCE JOINS ROUGHNESS", 1, uiWhite)
	drawText(img, 64, h-110, "HIGHLIGHT STAYS CONNECTED ACROSS PIXELS", 1, uiDim)
	drawScaleBar(img, 64, h-72, 132, "500 MM / OBJECT SPACE")
	drawText(img, w-280, h-72, "20.0 M FOCUS", 1, uiWarm)
	drawText(img, w-280, h-90, "MICROFACETS: AGGREGATED", 1, uiDim)
	drawCornerMarks(img)
}
