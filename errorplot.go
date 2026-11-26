package main

import (
	"fmt"
	"image"
)

// RenderErrorCharts builds the target-vs-end-effector error figure.
func RenderErrorCharts(cfg GaitConfig, rows []MetricRow, sweep []SweepPoint, summ Summary) *image.RGBA {
	const W, H = 1420, 980
	cv := NewCanvas(W, H, colPaper)
	cv.drawText(36, 26, "TARGET vs END-EFFECTOR ERROR", 3, colInk)
	cv.drawText(38, 64,
		"WALK CYCLES (LOG SCALE, MACHINE PRECISION) + COMMAND-DISTANCE REACH SWEEP", 1, colGray)

	// ---- Top: walk errors in units of 1e-16 m (machine precision) ----
	var xIdx, eL, eR, lockE []float64
	for _, r := range rows {
		if r.Foot == cfg.FootNames[1] {
			xIdx = append(xIdx, float64(r.Frame))
			eR = append(eR, r.ErrorM/1e-16)
			lockE = append(lockE, r.LockDriftM/1e-16)
		} else {
			eL = append(eL, r.ErrorM/1e-16)
		}
	}

	top := chartSpec{
		title:  "WALK: IK ERROR AND SOLE LOCK DRIFT (UNITS OF 1E-16 M)",
		xLabel: "DENSE SAMPLE FRAME (2 CYCLES)", yLabel: "ERROR (1E-16 M)",
		xMin: 0, xMax: float64(xIdx[len(xIdx)-1]), yMin: 0, yMax: 5,
		yTicks: []float64{0, 1, 2, 3, 4, 5},
		xTicks: []float64{0, 40, 80, 120, 160, 200, 240},
		series: []lineSeries{
			{name: "L IK ERR", col: colBlue, w: 2.0, x: xIdx, y: eL},
			{name: "R IK ERR", col: colTeal, w: 2.0, x: xIdx, y: eR},
			{name: "SOLE LOCK DRIFT", col: colAmber, w: 2, dash: [2]float64{6, 4}, x: xIdx, y: lockE},
		},
		notes: []chartNote{
			{x: 62, y: 4.55, text: fmt.Sprintf("MAX IK ERROR = %.2E M", summ.MaxIKError), scale: 2, col: colOK},
			{x: 62, y: 3.85, text: fmt.Sprintf("MAX SOLE DRIFT = %.2E M - MACHINE PRECISION", summ.MaxLockDrift), scale: 2, col: colOK},
		},
	}
	drawChart(cv, image.Rect(30, 92, W-30, 500), top)

	// ---- Bottom: reach sweep linear ----
	var dx, ey, ky []float64
	for _, p := range sweep {
		dx = append(dx, p.D)
		ey = append(ey, p.ErrorM)
		ky = append(ky, p.KneeDeg)
	}
	floor := cfg.Leg.FlexFloor()
	reach := cfg.Leg.Reach()
	bottom := chartSpec{
		title:  "REACH SWEEP: END ERROR vs COMMANDED HIP-TARGET DISTANCE",
		xLabel: "COMMANDED DISTANCE D (M)", yLabel: "END ERROR (M)",
		legend: "bottom-left",
		xMin:   0.2, xMax: 1.2, yMin: 0, yMax: 0.42,
		yTicks:  []float64{0, 0.05, 0.1, 0.15, 0.2, 0.25, 0.3, 0.35, 0.4},
		xTicks:  []float64{0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0, 1.1, 1.2},
		xFormat: func(v float64) string { return fmt.Sprintf("%.1f", v) },
		yFormat: func(v float64) string { return fmt.Sprintf("%.2f", v) },
		zones: []vZone{
			{x0: 0.2, x1: floor, col: mixRGB(colAmber, colPaper, 0.82)},
			{x0: floor, x1: reach, col: mixRGB(colOK, colPaper, 0.88)},
			{x0: reach, x1: 1.2, col: mixRGB(colWarn, colPaper, 0.85)},
		},
		vLines: []vLine{
			{x: floor, col: colAmber, dash: [2]float64{6, 4}, label: "FLEX FLOOR"},
			{x: reach, col: colWarn, dash: [2]float64{7, 4}, label: "R = L1+L2"},
		},
		series: []lineSeries{
			{name: "END ERROR", col: colInk, w: 3.5, x: dx, y: ey},
			{name: "KNEE FLEX/300", col: colPurple, w: 2, dash: [2]float64{3, 3}, x: dx, y: scaleSlice(ky, 1/300.0)},
		},
		notes: []chartNote{
			{x: 0.23, y: 0.395, text: "TOO CLOSE: FLEX CLAMP (AMBER)", scale: 2, col: colAmber},
			{x: 0.56, y: 0.04, text: "FEASIBLE ANNULUS: ERROR ~ 0", scale: 2, col: colOK},
			{x: 0.94, y: 0.395, text: "OVER-REACH: ERR = D - R", scale: 2, col: colWarn},
		},
	}
	drawChart(cv, image.Rect(30, 528, W-30, 968), bottom)
	return cv.img
}

func scaleSlice(a []float64, k float64) []float64 {
	out := make([]float64, len(a))
	for i, v := range a {
		out[i] = v * k
	}
	return out
}
