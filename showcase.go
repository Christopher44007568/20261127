package flakepaint

import (
	"image"
	"image/color"
)

func drawRGBAAt(dst, src *image.RGBA, x0, y0 int) {
	b := src.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			dst.SetRGBA(x0+x, y0+y, src.RGBAAt(x, y))
		}
	}
}

func framedImage(dst *image.RGBA, src *image.RGBA, x, y, w, h int, c color.RGBA) {
	scaled := scaleRGBA(src, w, h)
	drawRGBAAt(dst, scaled, x, y)
	drawLine(dst, x-1, y-1, x+w, y-1, c)
	drawLine(dst, x-1, y+h, x+w, y+h, color.RGBA{c.R, c.G, c.B, 60})
	drawLine(dst, x-1, y-1, x-1, y+h, color.RGBA{c.R, c.G, c.B, 60})
	drawLine(dst, x+w, y-1, x+w, y+h, color.RGBA{c.R, c.G, c.B, 60})
}

// ComposeShowcase builds a three-view presentation sheet from the rendered
// near, far, and density frames.
func ComposeShowcase(near, far, density *image.RGBA) *image.RGBA {
	w, h := 1800, 1020
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		f := float64(y) / float64(h-1)
		c := color.RGBA{
			R: uint8(mix(4, 8, f)),
			G: uint8(mix(7, 14, f)),
			B: uint8(mix(11, 20, f)),
			A: 255,
		}
		fillRect(img, 0, y, w, y+1, c)
	}

	drawTextShadow(img, 58, 36, "METALLIC FLAKE PAINT", 3, uiWhite)
	drawText(img, 60, 72, "LAYERED HEMISPHERE NORMALS / STABLE FAR-FIELD LOD", 1, uiDim)
	drawText(img, 60, 94, "PURE GO STANDARD LIBRARY / ZERO EXTERNAL DEPENDENCIES", 1, uiAccent)
	drawText(img, w-360, 44, "01  NEAR CAR PANEL", 1, uiAccent)
	drawText(img, w-360, 64, "02  FAR MATERIAL SPHERE", 1, uiAccent)
	drawText(img, w-360, 84, "03  FLAKE DENSITY HEATMAP", 1, uiAccent)
	drawLine(img, 58, 120, w-58, 120, color.RGBA{105, 175, 180, 100})

	framedImage(img, near, 58, 142, 1060, 671, color.RGBA{105, 205, 205, 120})
	framedImage(img, far, 1160, 142, 590, 374, color.RGBA{105, 205, 205, 90})
	framedImage(img, density, 1160, 558, 590, 374, color.RGBA{105, 205, 205, 90})

	drawText(img, 60, 844, "LAYER STACK", 1, uiWarm)
	drawText(img, 60, 866, "04 / 5.30 MM / COPPER", 1, uiDim)
	drawText(img, 60, 886, "03 / 3.30 MM / GOLD", 1, uiDim)
	drawText(img, 60, 906, "02 / 2.05 MM / AQUA", 1, uiDim)
	drawText(img, 60, 926, "01 / 1.25 MM / SILVER", 1, uiDim)
	drawText(img, 60, 962, "NORMAL  +  ROUGHNESS  +  SPECULAR LOBE", 1, uiWhite)
	drawText(img, 60, 982, "MICROFACETS AGGREGATE INTO A CONTINUOUS BAND WITH DISTANCE", 1, uiAccent)

	for i := 0; i < 4; i++ {
		y := 861 + i*20
		c := []color.RGBA{
			{100, 210, 235, 255},
			{80, 220, 180, 255},
			{235, 185, 80, 255},
			{235, 100, 55, 255},
		}[3-i]
		fillRect(img, 46, y+2, 51, y+10, c)
	}
	return img
}
