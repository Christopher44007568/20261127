package main

import (
	"fmt"
	"image"
	"math"
)

type rigDemo struct {
	title  string
	center Vec
	scale  float64
	hip    Vec
	target Vec
	note   string
}

func rigReachable(cfg GaitConfig) rigDemo {
	return rigDemo{
		title:  "IK: REACHABLE TARGET",
		center: vec(0.10, 0.56), scale: 235,
		hip: vec(-0.20, 0.98), target: vec(0.30, 0.34),
		note: "TWO-BONE SOLVE - POLE=UPHILL",
	}
}

func rigOverReach(cfg GaitConfig) rigDemo {
	return rigDemo{
		title:  "IK: TARGET BEYOND REACH",
		center: vec(0.26, 0.56), scale: 215,
		hip: vec(-0.20, 0.98), target: vec(0.80, 0.40),
		note: "CHAIN STRAIGHTENED - MARKED RED",
	}
}

func rigFlexLimit(cfg GaitConfig) rigDemo {
	return rigDemo{
		title:  "IK: TOO-CLOSE / FLEX LIMIT",
		center: vec(0.06, 0.55), scale: 255,
		hip: vec(0.04, 0.95), target: vec(0.08, 0.74),
		note: "KNEE CLAMPED AT 108 DEG",
	}
}

func drawRigPanel(root *Canvas, cfg GaitConfig, r image.Rectangle, d rigDemo) {
	cv := root.panel(r)
	v := View{
		CX:     float64(r.Min.X + r.Dx()/2),
		CY:     float64(r.Min.Y) + 0.50*float64(r.Dy()),
		Center: d.center,
		Scale:  d.scale,
	}
	drawGround(cv, r, v, cfg.Slope)

	pose := cfg.Leg.Solve(SolveArgs{
		Hip: d.hip, Target: d.target, Pole: cfg.Slope.Uphill,
		PelvisAng: cfg.Slope.Ang, FootAng: cfg.Slope.Ang,
	})

	// Reach sphere (L1+L2) around the hip.
	hx, hy := v.px(d.hip)
	cv.ring(hx, hy, cfg.Leg.Reach()*v.Scale, 2, colGray)
	rightReach := d.hip.add(vec(cfg.Leg.Reach(), 0))
	lx, ly := v.px(rightReach)
	cv.drawText(lx-30, ly-18, "R=0.90", 1, colGray)

	wc := wcanvas{cv: cv, v: v}
	legCol := statusColor(pose.Status, colBlue)
	wc.line(pose.Hip, pose.Knee, 0.075, legCol)
	wc.line(pose.Knee, pose.Ankle, 0.062, legCol)
	wc.circle(pose.Knee, 0.05, legCol)
	kx, ky := v.px(pose.Knee)
	cv.ring(kx, ky, 0.05*v.Scale, 2, colInk)
	cv.fillCircle(hx, hy, 7, colTorso)

	drawTargetHandle(cv, v, FootSample{Pose: pose}, false)
	drawKneeArc(cv, v, pose)

	// World labels.
	labelAt := func(p Vec, text string, fg RGB) {
		x, y := v.px(p)
		chip(cv, x+6, y-18, text, 1, fg, colPaper)
	}
	labelAt(d.hip, "HIP", colInk)
	labelAt(pose.Knee, fmt.Sprintf("K %.0f", deg(pose.KneeFlex)), colPurple)
	labelAt(d.target, "TARGET", statusColor(pose.Status, colInk))

	root.frameRect(r, 2, colLightGray)
	panelTitle(cv, r, d.title)

	statusTxt := "REACHABLE"
	statusCol := colOK
	if pose.Status == StatusOverReach {
		statusTxt = "UNREACHABLE - STRAIGHT + RED"
		statusCol = colWarn
	} else if pose.Status != StatusOK {
		statusTxt = "LIMIT CLAMPED - AMBER"
		statusCol = colAmber
	}
	rows := []infoRow{
		{fmt.Sprintf("D=%.2f M  (R=0.90 M)", pose.CmdDistance), 2, colInk},
		{fmt.Sprintf("END-ERR=%.3f M  KNEE=%.0f DEG", pose.EndError, deg(pose.KneeFlex)), 2, statusCol},
		{statusTxt, 2, statusCol},
		{d.note, 1, colGray},
	}
	drawInfoBox(cv, r, rows, statusCol)
}

func drawKneeArc(cv *Canvas, v View, p LegPose) {
	a0 := p.ShinAng
	span := clampF(p.KneeFlex, 0.02, math.Pi-0.02)
	const n = 20
	var prev Vec
	for i := 0; i <= n; i++ {
		a := a0 + span*float64(i)/float64(n)
		q := p.Knee.add(vec(math.Cos(a), math.Sin(a)).mul(0.11))
		if i > 0 {
			wc := wcanvas{cv: cv, v: v}
			wc.line(prev, q, 0.012, colPurple)
		}
		prev = q
	}
}

type infoRow struct {
	text  string
	scale int
	col   RGB
}

func drawInfoBox(cv *Canvas, r image.Rectangle, rows []infoRow, accent RGB) {
	boxH := 14
	for _, row := range rows {
		boxH += fontGlyphH*row.scale + 8
	}
	x := r.Min.X + 8
	y := r.Max.Y - boxH - 8
	w := r.Dx() - 16
	cv.fillRect(x, y, w, boxH, colPaper)
	cv.frameRect(image.Rect(x, y, x+w, y+boxH), 2, accent)
	ry := y + 7
	for _, row := range rows {
		cv.drawText(x+10, ry, row.text, row.scale, row.col)
		ry += fontGlyphH*row.scale + 8
	}
}
