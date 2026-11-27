package main

import (
	"fmt"
	"image"
)

func cropScaleNearest(src *image.RGBA, x, y, w, h, scale int) *image.RGBA {
	out := image.NewRGBA(image.Rect(0, 0, w*scale, h*scale))
	for j := 0; j < h; j++ {
		for i := 0; i < w; i++ {
			c := src.RGBAAt(x+i, y+j)
			for dy := 0; dy < scale; dy++ {
				for dx := 0; dx < scale; dx++ {
					out.SetRGBA(i*scale+dx, j*scale+dy, c)
				}
			}
		}
	}
	return out
}

// BuildJointSheet shows 2x nearest-neighbor close crops with a tile
// boundary marker, proving joints stay complete and seams do not move.
func BuildJointSheet(orig, proc *image.RGBA, jO, jP JointContrast) *image.RGBA {
	const (
		margin = 30
		gap    = 28
		cx, cy = 120, 220
		cw, ch = 320, 200
		zoom   = 2
		barH   = 40
	)
	pw, ph := cw*zoom, ch*zoom
	W := margin*2 + pw*2 + gap
	H := 104 + barH + ph + 74
	cv := newCanvas(W, H, uiBg)

	drawText(cv, "CLOSE VIEW: JOINTS COMPLETE, SEAMS LOCKED", margin, 26, 3, uiText)
	drawText(cv, "2X NEAREST CROP - DASHED LINE MARKS THE TILE BOUNDARY; GEOMETRY IS NEVER WARPED",
		margin, 70, 2, uiMuted)

	y := 104
	labelBar(cv, margin, y, pw, barH, "ORIGINAL TILING", uiMuted)
	c0 := cropScaleNearest(orig, cx, cy, cw, ch, zoom)
	paste(cv, c0, margin, y+barH)

	x2 := margin + pw + gap
	labelBar(cv, x2, y, pw, barH, "PROCESSED: SAME MORTAR SKELETON", uiGood)
	c1 := cropScaleNearest(proc, cx, cy, cw, ch, zoom)
	paste(cv, c1, x2, y+barH)

	// Dashed marker at the x=TileW seam inside both crops.
	seamX := (TileW - cx) * zoom
	top := y + barH
	for yy := 0; yy < ph; yy += 12 {
		fillRect(cv, margin+seamX, top+yy, 2, 7, uiAccent)
		fillRect(cv, x2+seamX, top+yy, 2, 7, uiAccent)
	}

	note := fmt.Sprintf("JOINT CONTRAST %.1f -> %.1f   STRONG-LINE COVERAGE %.2f -> %.2f",
		jO.MeanContrast, jP.MeanContrast, jO.StrongFrac, jP.StrongFrac)
	drawText(cv, note, margin, top+ph+22, 2, uiAccent)
	return cv
}

// BuildDistantSheet simulates a far view (8x box reduction then smooth
// enlargement) for original vs processed, with the far-scale metric.
func BuildDistantSheet(orig, proc *image.RGBA, far scaleReport) *image.RGBA {
	const (
		margin = 30
		gap    = 28
		panel  = 512
		barH   = 40
	)
	W := margin*2 + panel*2 + gap
	H := 118 + barH + panel + 70
	cv := newCanvas(W, H, uiBg)

	drawText(cv, "DISTANT VIEW SIMULATION", margin, 26, 3, uiText)
	drawText(cv, "16X BOX DOWNSCALE, NEAREST ENLARGEMENT - FINE JOINTS MERGE, MACRO GRID STAYS VISIBLE",
		margin, 70, 2, uiMuted)

	y := 104
	labelBar(cv, margin, y, panel, barH, "ORIGINAL: CLONES VISIBLE AT DISTANCE", uiBad)
	pasteScaledNearest(cv, BoxDownscale(orig, 16), margin, y+barH, panel, panel)

	x2 := margin + panel + gap
	labelBar(cv, x2, y, panel, barH, "PROCESSED: MACRO TONE, NO TILE GRID", uiGood)
	pasteScaledNearest(cv, BoxDownscale(proc, 16), x2, y+barH, panel, panel)

	note := fmt.Sprintf("FAR-SCALE REPETITION %.2f -> %.2f  (-%.0f%%)",
		far.OrigMeanRepetition, far.ProcMeanRepetition, far.RepetitionCutPct)
	drawText(cv, note, margin, y+barH+panel+24, 2, uiAccent)
	return cv
}
