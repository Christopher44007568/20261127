package furnace

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

const heatRangeLow = 0.70
const heatRangeHigh = 1.30

func energyHeatT(e float64) float64 {
	return (e - heatRangeLow) / (heatRangeHigh - heatRangeLow)
}

func drawHeatSphere(img *image.RGBA, cx, cy, radius int, value func(mu float64) float64) {
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx := float64(x-cx) / float64(radius)
			dy := float64(y-cy) / float64(radius)
			q := dx*dx + dy*dy
			if q > 1 {
				continue
			}
			mu := math.Sqrt(math.Max(0, 1-q))
			e := value(mu)
			c := HeatColor(energyHeatT(e))
			img.SetRGBA(x, y, c)
		}
	}

	ringColor := color.RGBA{R: 235, G: 245, B: 255, A: 58}
	for _, mu := range []float64{0.75, 0.50, 0.25} {
		rr := int(float64(radius) * math.Sqrt(math.Max(0, 1-mu*mu)))
		strokeCircle(img, cx, cy, rr, ringColor)
	}
	strokeCircle(img, cx, cy, radius, color.RGBA{R: 220, G: 232, B: 255, A: 180})
}

func strokeCircle(img *image.RGBA, cx, cy, r int, c color.RGBA) {
	steps := 128
	for i := 0; i < steps; i++ {
		a0 := 2 * math.Pi * float64(i) / float64(steps)
		a1 := 2 * math.Pi * float64(i+1) / float64(steps)
		x0 := cx + int(math.Round(float64(r)*math.Cos(a0)))
		y0 := cy + int(math.Round(float64(r)*math.Sin(a0)))
		x1 := cx + int(math.Round(float64(r)*math.Cos(a1)))
		y1 := cy + int(math.Round(float64(r)*math.Sin(a1)))
		drawLine(img, x0, y0, x1, y1, c)
	}
}

func drawThickLine(img *image.RGBA, x0, y0, x1, y1 int, c color.RGBA, thick int) {
	for d := -thick / 2; d <= thick/2; d++ {
		drawLine(img, x0, y0+d, x1, y1+d, c)
	}
}

func drawEnergyChart(img *image.RGBA, x, y, w, h int, curve EnergyCurve, report FurnaceReport) {
	fillRect(img, x, y, x+w, y+h, colorPanel)
	drawLine(img, x, y, x+w, y, colorGrid)
	drawLine(img, x, y+h-1, x+w, y+h-1, colorGrid)
	drawLine(img, x, y, x, y+h, colorGrid)
	drawLine(img, x+w-1, y, x+w-1, y+h, colorGrid)

	yMax := 2.0
	plotX := x + 44
	plotY := y + 31
	plotW := w - 64
	plotH := h - 65

	mapX := func(mu float64) int {
		return plotX + int((1-saturate(mu))*float64(plotW))
	}
	mapY := func(e float64) int {
		return plotY + plotH - int(saturate(e/yMax)*float64(plotH))
	}

	for _, e := range []float64{0, 0.5, 1, 1.5, 2} {
		py := mapY(e)
		lineColor := color.RGBA{R: 55, G: 66, B: 90, A: 255}
		if e == 1 {
			lineColor = color.RGBA{R: 50, G: 150, B: 95, A: 255}
		}
		drawLine(img, plotX, py, plotX+plotW, py, lineColor)
		drawText(img, x+10, py-5, fmt.Sprintf("%.1f", e), 1, colorMuted)
	}

	bandTop := mapY(1 + report.Tolerance)
	bandBottom := mapY(1 - report.Tolerance)
	fillRect(img, plotX, bandTop, plotX+plotW, bandBottom, color.RGBA{R: 60, G: 220, B: 140, A: 38})

	drawText(img, x+12, y+8, "DIRECTIONAL INTEGRAL", 1, colorText)
	drawText(img, plotX-8, y+h-24, "1", 1, colorMuted)
	drawText(img, plotX+plotW/2-10, y+h-24, ".5", 1, colorMuted)
	drawText(img, plotX+plotW-16, y+h-24, "0", 1, colorMuted)
	drawText(img, plotX+plotW-150, y+h-14, "CENTER TO RIM N.V", 1, colorMuted)

	rawColor := color.RGBA{R: 255, G: 184, B: 54, A: 255}
	corrColor := color.RGBA{R: 82, G: 226, B: 255, A: 255}
	for i := 1; i < len(curve.Mu); i++ {
		x0, y0 := mapX(curve.Mu[i-1]), mapY(curve.Raw[i-1])
		x1, y1 := mapX(curve.Mu[i]), mapY(curve.Raw[i])
		drawThickLine(img, x0, y0, x1, y1, rawColor, 2)

		x0, y0 = mapX(curve.Mu[i-1]), mapY(curve.Corrected[i-1])
		x1, y1 = mapX(curve.Mu[i]), mapY(curve.Corrected[i])
		drawThickLine(img, x0, y0, x1, y1, corrColor, 2)
	}

	maxRaw := 0.0
	for _, e := range curve.Raw {
		maxRaw = math.Max(maxRaw, e)
	}
	drawText(img, x+18, y+h-28, fmt.Sprintf("RAW MAX %.2f", maxRaw), 1, rawColor)
	status := fmt.Sprintf("CORR WORST %.4f", report.WorstError)
	c := colorGood
	if !report.Pass {
		c = colorBad
	}
	drawText(img, x+170, y+h-28, status, 1, c)
}

// RenderEnergyHeatmap creates the three-row directional energy dashboard.
func RenderEnergyHeatmap(reports []FurnaceReport) *image.RGBA {
	w, h := 1500, 980
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	fillRect(img, 0, 0, w, h, colorBackground)

	drawText(img, 28, 24, "WHITE FURNACE ENERGY HEATMAP", 3, colorText)
	drawText(img, 30, 58, "E(N.V) = INTEGRAL F(V,L) COS(THETA_L) D W", 1, colorMuted)

	barX, barY, barW, barH := 1080, 28, 360, 20
	for x := 0; x < barW; x++ {
		t := float64(x) / float64(barW-1)
		fillRect(img, barX+x, barY, barX+x+1, barY+barH, HeatColor(t))
	}
	drawLine(img, barX, barY, barX+barW, barY, colorText)
	drawLine(img, barX, barY+barH, barX+barW, barY+barH, colorText)
	drawText(img, barX, barY+28, ".70", 1, colorMuted)
	drawText(img, barX+barW/2-12, barY+28, "1.00", 1, colorText)
	drawText(img, barX+barW-42, barY+28, "1.30", 1, colorMuted)

	drawText(img, 92, 92, "RAW E", 2, colorWarn)
	drawText(img, 344, 92, "NORMALIZED E", 2, colorGood)
	drawText(img, 620, 94, "RAW", 1, colorWarn)
	drawText(img, 670, 94, "CORRECTED", 1, colorGood)

	for i, m := range Presets() {
		rowY := 118 + i*286
		centerY := rowY + 126
		radius := 98
		table := DefaultEnergyTable(m)
		curve := BuildEnergyCurve(m, table, 44, 8192)
		report := FurnaceReport{Tolerance: 0.025, Pass: true, WorstError: 0}
		if i < len(reports) {
			report = reports[i]
		}

		fillRect(img, 24, rowY-10, 560, rowY+252, color.RGBA{R: 20, G: 26, B: 40, A: 255})
		drawHeatSphere(img, 142, centerY, radius, table.WhiteEnergy)
		drawTextShadow(img, 52, rowY+8, kindTitle(m.Kind), 2, colorText)

		corrValue := func(mu float64) float64 { return curve.CorrectedAt(mu) }
		drawHeatSphere(img, 414, centerY, radius, corrValue)

		drawEnergyChart(img, 610, rowY+8, 866, 244, curve, report)

		pass := "PASS"
		passColor := colorGood
		if !report.Pass {
			pass = "FAIL"
			passColor = colorBad
		}
		passWidth := textWidth(pass, 2)
		drawTextShadow(img, 414-passWidth/2, rowY+228, pass, 2, passColor)
	}

	drawText(img, 28, h-28, "GREEN SPHERE MEANS E=1 FROM CENTER TO RIM; RINGS ARE N.V=.75 .50 .25", 1, colorMuted)
	return img
}
