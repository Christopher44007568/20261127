package pipeline

import (
	"fmt"

	"materiallod/internal/chart"
	"materiallod/material"
)

func renderViewSweepPNG(dir string, art *Artifacts) error {
	const width, height = 1200, 580
	cv := chart.New(width, height, chart.Hex(pageBG))
	cv.Text(30, 34, "VIEW ANGLE SWEEP", 4, chart.Hex(ink))
	cv.Text(32, 80, "FIXED NORMAL - VIEW ANGLE 0 TO 90 DEGREES - DISPLAY LUMA", 1, chart.Hex(muted))

	panelW, panelH := 360, 360
	xs := []int{28, 420, 812}
	for i, mat := range art.Materials {
		renderSweepPanel(cv, art.Sweeps, mat, xs[i], 104, panelW, panelH)
	}
	renderSweepLegend(cv)
	return savePNG(dir, "view_sweep.png", cv.Img)
}

func renderSweepPanel(cv *chart.Canvas, samples []material.ViewSample, mat material.Material, x, y, w, h int) {
	cv.Rect(x, y, x+w, y+h, chart.Hex(cardBG))
	cv.TextCenter(x+w/2, y+24, panelTitle(mat.Kind), 2, chart.Hex(ink))

	px, py := x+62, y+58
	pw, ph := w-92, h-112
	axisBottom := py + ph

	for tick := 0.0; tick <= 1.0001; tick += 0.25 {
		ty := axisBottom - int(tick*float64(ph))
		cv.Line(px, ty, px+pw, ty, chart.Hex(gridLine), 1)
		label := fmt.Sprintf("%.2f", tick)
		cv.TextRight(px-10, ty-5, label, 1, chart.Hex(muted))
	}
	for _, angle := range []float64{0, 30, 60, 90} {
		tx := px + int(angle/90*float64(pw))
		cv.Line(tx, py, tx, axisBottom, chart.Hex(0xE7EAF1), 1)
		cv.TextCenter(tx, axisBottom+12, fmt.Sprintf("%.0f", angle), 1, chart.Hex(muted))
	}
	cv.Line(px, py-8, px, axisBottom, chart.Hex(0x475467), 3)
	cv.Line(px, axisBottom, px+pw+8, axisBottom, chart.Hex(0x475467), 3)
	cv.Text(px+pw/2-58, y+h-28, "VIEW ANGLE DEG", 1, chart.Hex(muted))
	cv.Text(x+16, y+42, "LUMA", 1, chart.Hex(muted))

	for tier := material.TierLow; tier <= material.TierHigh; tier++ {
		points := make([]chart.Point, 0, 91)
		for _, s := range samples {
			if s.MaterialID != mat.ID || s.Tier != tier.String() {
				continue
			}
			points = append(points, chart.Point{
				X: float64(px) + s.AngleDeg/90*float64(pw),
				Y: float64(axisBottom) - s.DisplayLuma*float64(ph),
			})
		}
		col := tierColor(tier)
		switch tier {
		case material.TierLow:
			cv.StyledPolyline(points, col, 3, 2, 5)
		case material.TierMedium:
			cv.StyledPolyline(points, col, 3, 9, 6)
		default:
			cv.Polyline(points, col, 3)
		}
	}
}

func panelTitle(kind string) string {
	switch kind {
	case "metal":
		return "METAL"
	case "cloth":
		return "CLOTH"
	default:
		return "PLASTIC"
	}
}

func renderSweepLegend(cv *chart.Canvas) {
	y := 512
	items := []struct {
		label string
		tier  material.Tier
		style int
	}{
		{"LOW", material.TierLow, 0},
		{"MEDIUM", material.TierMedium, 1},
		{"HIGH", material.TierHigh, 2},
	}
	totalWidth := 560
	startX := 600 - totalWidth/2
	for i, item := range items {
		x := startX + i*210
		col := tierColor(item.tier)
		switch item.style {
		case 0:
			cv.StyledPolyline([]chart.Point{{X: float64(x), Y: float64(y + 8)}, {X: float64(x + 44), Y: float64(y + 8)}}, col, 3, 2, 5)
		case 1:
			cv.StyledPolyline([]chart.Point{{X: float64(x), Y: float64(y + 8)}, {X: float64(x + 44), Y: float64(y + 8)}}, col, 3, 9, 6)
		default:
			cv.Line(x, y+8, x+44, y+8, col, 3)
		}
		cv.Text(x+58, y, item.label, 2, chart.Hex(ink))
	}
}
