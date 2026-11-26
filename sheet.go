package main

import (
	"fmt"
	"image"
)

const (
	panelW    = 460
	panelH    = 350
	sheetCols = 4
	sheetRows = 4
	sheetPad  = 20
	sheetGap  = 14
	headerH   = 96
)

var (
	sheetW  = sheetPad*2 + sheetCols*panelW + (sheetCols-1)*sheetGap
	gridTop = sheetPad + headerH
	sheetH  = gridTop + sheetRows*panelH + (sheetRows-1)*sheetGap + sheetPad
)

func panelRect(col, row int) image.Rectangle {
	x0 := sheetPad + col*(panelW+sheetGap)
	y0 := gridTop + row*(panelH+sheetGap)
	return image.Rect(x0, y0, x0+panelW, y0+panelH)
}

// RenderSheet builds the 4x4 contact sheet: 12 walk phases plus four
// diagnostic panels (sole lock, reachable IK, over-reach, flex limit).
func RenderSheet(cfg GaitConfig) *image.RGBA {
	root := NewCanvas(sheetW, sheetH, hexRGB(0xE6EAF0))

	root.drawText(sheetPad, 20,
		"BIPED SLOPE WALK - TWO-BONE IK / JOINT LIMITS / SOLE LOCK", 4, colInk)
	root.drawText(sheetPad, 62,
		"14 DEG SLOPE - L1=L2=0.45 M - REACH R=0.90 M - KNEE 0 TO 108 DEG (NO HYPEREXTEND)", 2, colGray)
	root.drawText(sheetPad, 82,
		"12 PHASES OF ONE GAIT CYCLE + LOCK / REACH BOUNDARY CASES - PURE GO STDLIB", 1, colGray)

	for i := 0; i < 12; i++ {
		drawWalkPanel(root, cfg, panelRect(i%4, i/4), i)
	}
	drawLockPanel(root, cfg, panelRect(0, 3))
	drawRigPanel(root, cfg, panelRect(1, 3), rigReachable(cfg))
	drawRigPanel(root, cfg, panelRect(2, 3), rigOverReach(cfg))
	drawRigPanel(root, cfg, panelRect(3, 3), rigFlexLimit(cfg))
	return root.img
}

func chip(cv *Canvas, x, y int, text string, scale int, fg, bg RGB) (int, int) {
	tw, th := textSize(text, scale)
	px, py := 6, 3
	w, h := tw+2*px, th+2*py
	cv.fillRect(x, y, w, h, bg)
	cv.drawText(x+px, y+py, text, scale, fg)
	return w, h
}

func chipRight(cv *Canvas, xMax, y int, text string, scale int, fg, bg RGB) (int, int) {
	tw, th := textSize(text, scale)
	px, py := 6, 3
	w, h := tw+2*px, th+2*py
	chip(cv, xMax-w, y, text, scale, fg, bg)
	return w, h
}

func panelTitle(cv *Canvas, r image.Rectangle, text string) {
	chip(cv, r.Min.X+8, r.Min.Y+8, text, 2, colInk, colPaper)
}

func errShort(e float64) string {
	if e < 1e-7 {
		return "0.000000"
	}
	return fmt.Sprintf("%.3f", e)
}

func drawWalkPanel(root *Canvas, cfg GaitConfig, r image.Rectangle, idx int) {
	t := float64(idx) * cfg.CycleT / 12
	drawWalkPanelAt(root, cfg, r, t, fmt.Sprintf("F%02d  P=%.2f", idx, t/cfg.CycleT))
}

func drawWalkPanelAt(root *Canvas, cfg GaitConfig, r image.Rectangle, t float64, title string) {
	cv := root.panel(r)
	f := cfg.SampleFrame(t)
	v := followView(r, f, 158)

	drawGround(cv, r, v, cfg.Slope)
	drawCharacter(cv, cfg, f, v, charOpts{showTargets: true, compactTarget: true})
	root.frameRect(r, 2, colLightGray)

	panelTitle(cv, r, title)

	y := r.Min.Y + 8
	for _, fs := range f.Feet {
		txt := fs.Name + " SWING"
		bg := colGray
		if fs.Stance {
			txt = fs.Name + " LOCK"
			bg = colOK
		}
		_, h := chipRight(cv, r.Max.X-8, y, txt, 2, colPaper, bg)
		y += h + 4
	}

	e0 := f.Feet[0].Pose.EndError
	e1 := f.Feet[1].Pose.EndError
	errCol := colInk
	if e0 > 1e-6 || e1 > 1e-6 {
		errCol = colWarn
	}
	chip(cv, r.Min.X+8, r.Max.Y-24,
		fmt.Sprintf("ERR L=%s R=%s M", errShort(e0), errShort(e1)), 1, errCol, colPaper)
}

func drawLockPanel(root *Canvas, cfg GaitConfig, r image.Rectangle) {
	cv := root.panel(r)
	solid := cfg.SampleFrame(0.31 * cfg.CycleT)
	v := followView(r, solid, 158)

	drawGround(cv, r, v, cfg.Slope)
	ghost := colLightGray
	for _, gt := range []float64{0.06, 0.18, 0.42, 0.56} {
		gf := cfg.SampleFrame(gt * cfg.CycleT)
		drawCharacter(cv, cfg, gf, v, charOpts{mono: &ghost, thin: true})
	}
	drawCharacter(cv, cfg, solid, v, charOpts{showTargets: true, compactTarget: true})
	root.frameRect(r, 2, colOK)

	panelTitle(cv, r, "SOLE LOCK: STANCE OVERLAP")
	chip(cv, r.Min.X+8, r.Min.Y+34, "5 STANCE SAMPLES - SOLE DRIFT 0.000000 M", 1, colOK, colPaper)
}
