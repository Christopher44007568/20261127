package furnace

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// RenderWhiteFurnaceSheet shows literal constant-environment responses. The
// background is unit white; a corrected specimen should blend into it from
// center through rim.
func RenderWhiteFurnaceSheet(reports []FurnaceReport) *image.RGBA {
	w, h := 920, 1080
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	light := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	fillRect(img, 0, 0, w, h, light)

	ink := color.RGBA{R: 24, G: 30, B: 42, A: 255}
	muted := color.RGBA{R: 82, G: 94, B: 112, A: 255}
	drawText(img, 28, 26, "WHITE FURNACE TEST", 3, ink)
	drawText(img, 30, 58, "CONSTANT WHITE ENVIRONMENT E=1; CORRECTED SPHERE SHOULD VANISH", 1, muted)
	drawText(img, 112, 96, "RAW", 2, colorBad)
	drawText(img, 552, 96, "CORRECTED", 2, colorGood)

	for i, m := range Presets() {
		top := 120 + i*310
		cx0, cx1, cy, radius := 220, 680, top+120, 112
		table := DefaultEnergyTable(m)
		curve := BuildEnergyCurve(m, table, 48, 8192)
		report := FurnaceReport{Pass: true, WorstError: 0.001}
		if i < len(reports) {
			report = reports[i]
		}

		drawText(img, 28, top+8, kindTitle(m.Kind), 2, ink)
		drawFurnaceBall(img, cx0, cy, radius, table.WhiteEnergy, false)
		drawFurnaceBall(img, cx1, cy, radius, curve.CorrectedAt, true)

		rawMsg := fmt.Sprintf("RAW WORST %.3f", report.RawWorst)
		corrMsg := fmt.Sprintf("CORR WORST %.4f", report.WorstError)
		drawText(img, cx0-92, cy+radius+18, rawMsg, 1, colorBad)
		c := colorGood
		if !report.Pass {
			c = colorBad
		}
		drawText(img, cx1-104, cy+radius+18, corrMsg, 1, c)
	}

	drawText(img, 28, h-28, "FAINT CIRCLES MARK SPHERE SILHOUETTE; NUMBERS ARE DIRECTIONAL INTEGRATION ERROR", 1, muted)
	return img
}

func drawFurnaceBall(img *image.RGBA, cx, cy, radius int, energy func(mu float64) float64, corrected bool) {
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx := float64(x-cx) / float64(radius)
			dy := float64(y-cy) / float64(radius)
			q := dx*dx + dy*dy
			if q > 1 {
				continue
			}
			mu := math.Sqrt(math.Max(0, 1-q))
			e := energy(mu)
			var col color.RGBA
			if corrected {
				g := uint8(clamp(math.Pow(clamp(e, 0, 1), 1/2.2)*255, 0, 255))
				col = color.RGBA{R: g, G: g, B: g, A: 255}
			} else if e <= 1 {
				g := uint8(clamp(math.Pow(clamp(e, 0, 1), 1/2.2)*255, 0, 255))
				col = color.RGBA{R: g, G: g, B: g, A: 255}
			} else {
				excess := saturate((e - 1) / 0.65)
				base := uint8(255 - 40*excess)
				green := uint8(uint32(255) - uint32(180*excess))
				col = color.RGBA{R: 255, G: green, B: base, A: 255}
			}
			img.SetRGBA(x, y, col)
		}
	}
	strokeCircle(img, cx, cy, radius, color.RGBA{R: 120, G: 132, B: 150, A: 120})
}
