package main

import (
	"fmt"
	"image"
	"image/color"
)

// BuildHeatSheet renders the flagship repetitiveness heatmap: original
// vs processed side by side with labels, color scale and headline metric.
func BuildHeatSheet(ho, hp *HeatMap, closeReport scaleReport) *image.RGBA {
	const (
		margin = 30
		gap    = 28
		panel  = 512
		barH   = 40
	)
	W := margin*2 + panel*2 + gap
	H := 150 + barH + panel + 96
	cv := newCanvas(W, H, uiBg)

	drawText(cv, "REPETITIVENESS HEATMAP", margin, 26, 3, uiText)
	sub := "SHIFT = ONE TILE PERIOD (256 PX BOTH AXES)  /  LOCAL PATCH METRIC"
	drawText(cv, sub, margin, 70, 2, uiMuted)

	py := 110
	// Left: original
	labelBar(cv, margin, py, panel, barH, "ORIGINAL TILING - RED EVERYWHERE", uiBad)
	paste(cv, ho.ColorizeScaled(2), margin, py+barH)
	// Right: processed
	rx := margin + panel + gap
	cut := closeReport.RepetitionCutPct
	labelBar(cv, rx, py, panel, barH,
		fmt.Sprintf("PROCESSED - %.0f%% LESS REPETITION", cut), uiGood)
	paste(cv, hp.ColorizeScaled(2), rx, py+barH)

	// Shared color bar.
	cbY := py + barH + panel + 26
	cbX0, cbX1 := margin, W-margin
	for x := cbX0; x < cbX1; x++ {
		t := float64(x-cbX0) / float64(cbX1-cbX0)
		r, g, b := heatLUT(t)
		fillRect(cv, x, cbY, 1, 18, color.RGBA{R: r, G: g, B: b, A: 255})
	}
	borderRect(cv, cbX0, cbY, cbX1-cbX0, 18, uiMuted)
	drawText(cv, "0  UNIQUE", cbX0, cbY+26, 2, uiMuted)
	hi := "1  EXACT REPEAT"
	drawText(cv, hi, cbX1-textWidth(hi, 2), cbY+26, 2, uiMuted)

	meanLine := fmt.Sprintf("MEAN REPETITION  %.2f -> %.2f   (DELTA %.1f -> %.1f)",
		closeReport.OrigMeanRepetition, closeReport.ProcMeanRepetition,
		closeReport.OrigMeanDelta, closeReport.ProcMeanDelta)
	drawText(cv, meanLine, margin, cbY+52, 2, uiAccent)
	return cv
}
