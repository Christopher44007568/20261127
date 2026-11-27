package pipeline

import (
	"fmt"

	"materiallod/internal/chart"
	"materiallod/material"
)

func renderBoundaryPNG(dir string, art *Artifacts) error {
	const width, height = 1200, 540
	cv := chart.New(width, height, chart.Hex(pageBG))
	cv.Text(30, 34, "LOD BOUNDARY ERROR", 4, chart.Hex(ink))
	cv.Text(32, 80, "CONTINUOUS CROSSFADE SCAN - ERROR VS HIGH AND MAX LUMA STEP", 1, chart.Hex(muted))

	xs := []int{28, 420, 812}
	for i, mat := range art.Materials {
		renderBoundaryPanel(cv, art.Continuity.Rows, mat.ID, xs[i], 102, 360, 350, art.Continuity.TransitionWidth)
	}
	renderBoundaryLegend(cv)
	return savePNG(dir, "boundary_continuity.png", cv.Img)
}

func renderBoundaryPanel(cv *chart.Canvas, rows []material.BoundaryRow, id string, x, y, w, h int, fadeWidth float64) {
	cv.Rect(x, y, x+w, y+h, chart.Hex(cardBG))
	cv.TextCenter(x+w/2, y+24, id, 2, chart.Hex(ink))

	px, py := x+60, y+58
	pw, ph := w-90, h-112
	bottom := py + ph
	yMax := 0.12
	for _, r := range rows {
		if r.MaterialID != id {
			continue
		}
		if r.RMSEvsHigh > yMax*0.92 {
			yMax = ceilStep(r.RMSEvsHigh*1.08, 0.03)
		}
	}

	lodX := func(lod float64) int { return px + int(lod/2*float64(pw)) }
	valY := func(v float64) int { return bottom - int(v/yMax*float64(ph)) }

	for _, boundary := range []float64{0.5, 1.5} {
		x0 := lodX(boundary - fadeWidth)
		x1 := lodX(boundary + fadeWidth)
		cv.Rect(x0, py, x1+1, bottom, chart.Hex(0xFBF1D3))
	}
	for tick := 0.0; tick <= yMax+1e-9; tick += 0.03 {
		ty := valY(tick)
		cv.Line(px, ty, px+pw, ty, chart.Hex(gridLine), 1)
		cv.TextRight(px-9, ty-5, fmt.Sprintf("%.3f", tick), 1, chart.Hex(muted))
	}
	for _, tick := range []float64{0, 0.5, 1, 1.5, 2} {
		tx := lodX(tick)
		cv.TextCenter(tx, bottom+12, trimFloat(tick), 1, chart.Hex(muted))
	}
	for _, boundary := range []float64{0.5, 1.5} {
		bx := lodX(boundary)
		cv.DashedLine(bx, py, bx, bottom, chart.Hex(0x98A2B3), 1, 4, 4)
	}
	cv.Line(px, py-8, px, bottom, chart.Hex(0x475467), 3)
	cv.Line(px, bottom, px+pw+8, bottom, chart.Hex(0x475467), 3)
	cv.TextCenter(x+w/2, y+h-27, "CONTINUOUS LOD  0=LOW  1=MID  2=HIGH", 1, chart.Hex(muted))
	cv.Text(x+14, y+42, "ERROR", 1, chart.Hex(muted))

	rmsePoints := []chart.Point{}
	jumpPoints := []chart.Point{}
	for _, r := range rows {
		if r.MaterialID != id {
			continue
		}
		rmsePoints = append(rmsePoints, chart.Point{X: float64(lodX(r.LOD)), Y: float64(valY(r.RMSEvsHigh))})
		jumpPoints = append(jumpPoints, chart.Point{X: float64(lodX(r.LOD)), Y: float64(valY(r.MaxStepJump))})
	}
	cv.Polyline(rmsePoints, chart.Hex(highColor), 3)
	cv.Polyline(jumpPoints, chart.Hex(0x12995B), 3)
}

func renderBoundaryLegend(cv *chart.Canvas) {
	y := 486
	startX := 350
	cv.Line(startX, y+8, startX+46, y+8, chart.Hex(highColor), 3)
	cv.Text(startX+58, y, "RMSE VS HIGH", 2, chart.Hex(ink))

	x2 := startX + 280
	cv.Line(x2, y+8, x2+46, y+8, chart.Hex(0x12995B), 3)
	cv.Text(x2+58, y, "STEP JUMP", 2, chart.Hex(ink))

	x3 := x2 + 230
	cv.DashedLine(x3, y+8, x3+46, y+8, chart.Hex(0x667085), 2, 5, 4)
	cv.Text(x3+58, y, "BOUNDARY", 2, chart.Hex(ink))
}

func ceilStep(v, step float64) float64 {
	return float64(int(v/step+0.999999)) * step
}

func trimFloat(v float64) string {
	if v == float64(int(v)) {
		return fmt.Sprintf("%.0f", v)
	}
	return fmt.Sprintf("%.1f", v)
}
