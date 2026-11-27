package pipeline

import (
	"image"
	"image/color"
	"image/draw"

	"materiallod/internal/chart"
	"materiallod/material"
)

const (
	pageBG    = 0xF3F5F9
	ink       = 0x20283A
	muted     = 0x667085
	cardBG    = 0xFFFFFF
	gridLine  = 0xD9DEE8
	lowColor  = 0x0E7490
	midColor  = 0xD97706
	highColor = 0xB42318
)

func renderSpheresPNG(dir string, art *Artifacts) error {
	const width, height = 1080, 930
	cv := chart.New(width, height, chart.Hex(pageBG))

	cv.Text(40, 34, "MATERIAL LOD SPHERES", 4, chart.Hex(ink))
	cv.Text(42, 82, "SHARED PARAMETER SET - LOW / MEDIUM / HIGH LIGHTING", 1, chart.Hex(muted))

	cardW, cardH := 260, 232
	startX, startY := 190, 126
	gapX, gapY := 26, 24
	titles := []struct {
		name string
		hex  uint32
	}{
		{"LOW", lowColor}, {"MEDIUM", midColor}, {"HIGH", highColor},
	}
	for col, title := range titles {
		x := startX + col*(cardW+gapX) + cardW/2
		cv.TextCenter(x, 104, title.name, 3, chart.Hex(title.hex))
	}

	for row, mat := range art.Materials {
		y := startY + row*(cardH+gapY)
		renderMaterialRowLabel(cv, mat, y, cardH)
		buffers := art.Spheres[mat.ID]
		for col := 0; col < 3; col++ {
			x := startX + col*(cardW+gapX)
			cv.Rect(x, y, x+cardW, y+cardH, chart.Hex(cardBG))
			cv.Frame(x, y, x+cardW, y+cardH, chart.Hex(0xE5E9F2), 1)

			centerX := x + cardW/2
			fillEllipse(cv, centerX, y+190, 62, 13, chart.Hex(0xD4DAE7))

			buf := buffers[col]
			sphere := buf.Image()
			pos := image.Pt(x+(cardW-material.SphereSize)/2, y+12)
			draw.Draw(cv.Img, image.Rectangle{Min: pos, Max: pos.Add(sphere.Bounds().Size())},
				sphere, image.Point{}, draw.Over)

			caption := "R " + f2(mat.Roughness) + " [" + f2(mat.RoughnessMin) +
				"-" + f2(mat.RoughnessMax) + "]"
			cv.TextCenter(centerX, y+211, caption, 1, chart.Hex(muted))
		}
	}

	return savePNG(dir, "spheres.png", cv.Img)
}

func renderMaterialRowLabel(cv *chart.Canvas, mat material.Material, y, cardH int) {
	title := map[string]string{
		"metal":   "METAL",
		"cloth":   "CLOTH",
		"plastic": "PLASTIC",
	}[mat.Kind]
	cv.Text(38, y+72, title, 3, chart.Hex(ink))
	c := swatchColor(mat.BaseColor)
	cv.Rect(42, y+116, 128, y+150, c)
	cv.Frame(42, y+116, 128, y+150, chart.Hex(0xFFFFFF), 1)
	cv.Text(40, y+166, mat.ID, 1, chart.Hex(muted))
	_ = cardH
}

func tierColor(tier material.Tier) color.RGBA {
	switch tier {
	case material.TierLow:
		return chart.Hex(lowColor)
	case material.TierMedium:
		return chart.Hex(midColor)
	default:
		return chart.Hex(highColor)
	}
}

var _ = gridLine
