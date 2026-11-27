package furnace

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
)

var (
	colorBackground = color.RGBA{R: 14, G: 18, B: 28, A: 255}
	colorPanel      = color.RGBA{R: 24, G: 30, B: 44, A: 255}
	colorGrid       = color.RGBA{R: 70, G: 84, B: 110, A: 255}
	colorText       = color.RGBA{R: 232, G: 238, B: 248, A: 255}
	colorMuted      = color.RGBA{R: 150, G: 165, B: 190, A: 255}
	colorGood       = color.RGBA{R: 70, G: 230, B: 140, A: 255}
	colorBad        = color.RGBA{R: 255, G: 92, B: 76, A: 255}
	colorWarn       = color.RGBA{R: 255, G: 196, B: 64, A: 255}
)

func kindTitle(k Kind) string {
	switch k {
	case RoughMetal:
		return "ROUGH METAL"
	case Clearcoat:
		return "CLEARCOAT"
	case Velvet:
		return "VELVET"
	default:
		return "MATERIAL"
	}
}

// RenderMaterialSheet creates the original or corrected three-material image.
func RenderMaterialSheet(corrected bool, panel int, reports []FurnaceReport) *image.RGBA {
	if panel < 180 {
		panel = 300
	}
	header := 76
	footer := 44
	w := panel * 3
	h := header + panel + footer
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: colorBackground}, image.Point{}, draw.Src)

	title := "ORIGINAL MATERIALS"
	subtitle := "LEGAL ALBEDO + LEGAL BRDF, NAIVE COMBINATION"
	if corrected {
		title = "ENERGY-NORMALIZED MATERIALS"
		subtitle = "DIRECTIONAL INTEGRATION + PER-VIEW CORRECTION"
	}
	drawText(img, 24, 20, title, 3, colorText)
	drawText(img, 26, 52, subtitle, 1, colorMuted)

	materials := Presets()
	for i, m := range materials {
		x := i * panel
		cell := image.Rect(x, header, x+panel, header+panel)
		draw.Draw(img, cell, &image.Uniform{C: colorPanel}, image.Point{}, draw.Src)

		cfg := DefaultRenderConfig(corrected)
		cfg.Width = panel
		cfg.Height = panel
		cfg.AA = 2
		cfg.Table = DefaultEnergyTable(m)
		sphere := RenderSpecimen(m, cfg)
		sphereImg := sphere.RGBA(cfg.Exposure)
		draw.Draw(img, cell, sphereImg, image.Point{}, draw.Src)

		drawLine(img, x, header, x+panel, header, colorGrid)
		drawLine(img, x, header+panel-1, x+panel, header+panel-1, colorGrid)
		drawLine(img, x, header, x, header+panel, colorGrid)
		drawLine(img, x+panel-1, header, x+panel-1, header+panel, colorGrid)

		drawTextShadow(img, x+16, header+16, fmt.Sprintf("%d %s", i+1, kindTitle(m.Kind)), 2, colorText)

		status := "RAW COMBINED ENERGY"
		statusColor := colorWarn
		if corrected {
			status = "WHITE FURNACE PASS"
			statusColor = colorGood
			if i < len(reports) && !reports[i].Pass {
				status = "CHECK TOLERANCE"
				statusColor = colorBad
			}
		}
		drawTextShadow(img, x+16, header+panel-34, status, 1, statusColor)
		if i < len(reports) {
			if corrected {
				msg := fmt.Sprintf("WORST |E-1| = %.4f", reports[i].WorstError)
				drawTextShadow(img, x+16, header+panel-18, msg, 1, colorText)
			} else {
				msg := fmt.Sprintf("RAW WORST |E-1| = %.3f", reports[i].RawWorst)
				drawTextShadow(img, x+16, header+panel-18, msg, 1, colorText)
			}
		}
	}

	note := "GO STANDARD LIBRARY ONLY - DETERMINISTIC CPU RENDER"
	drawText(img, 24, h-27, note, 1, colorMuted)
	if corrected && len(reports) == 3 && AllPass(reports) {
		msg := "ALL WHITE FURNACE TESTS PASS"
		tw := textWidth(msg, 2)
		drawText(img, w-tw-24, h-31, msg, 2, colorGood)
	}
	return img
}
