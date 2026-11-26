package main

import "image/color"

// drawKneeChart plots knee flexion for both legs of both skeletons over the
// whole clip, with a cursor marking the current frame and a shaded turn band.
func drawKneeChart(fb *FrameBuffer, frames []Frame, f int) {
	x0, x1 := 24, CanvasW-24
	y0, y1 := ChartY0, ChartY0+ChartH-10
	fb.FillRect(x0, y0, x1, y1, colPanel)
	fb.DrawText(x0+8, y0+8, 1, "KNEE FLEXION ANGLE (DEG)", colText)

	plotX0, plotX1 := x0+60, x1-16
	plotY0, plotY1 := y0+40, y1-22

	// Turn band.
	turnX0 := plotX0 + int(float64(TurnStartF)/float64(FrameCount-1)*float64(plotX1-plotX0))
	turnX1 := plotX0 + int(float64(TurnEndF)/float64(FrameCount-1)*float64(plotX1-plotX0))
	fb.FillRect(turnX0, plotY0, turnX1, plotY1, withAlpha(colTurn, 28))

	const degMin, degMax = 90.0, 180.0
	mapY := func(deg float64) int {
		u := (deg - degMin) / (degMax - degMin)
		return plotY1 - int(u*float64(plotY1-plotY0))
	}
	mapX := func(frame int) int {
		return plotX0 + int(float64(frame)/float64(FrameCount-1)*float64(plotX1-plotX0))
	}

	// Horizontal grid + labels.
	for deg := 100.0; deg <= 180.0; deg += 20 {
		y := mapY(deg)
		fb.Line(plotX0, y, plotX1, y, colGrid)
		label := intString(int(deg))
		fb.DrawText(plotX0-42, y-4, 1, label, colTextDim)
	}
	// Zero time axis labels.
	for frame := 0; frame < FrameCount; frame += 16 {
		x := mapX(frame)
		fb.Line(x, plotY1, x, plotY1+4, colAxis)
		label := intString(frame) + "F"
		fb.DrawText(x-8, plotY1+8, 1, label, colTextDim)
	}

	type curve struct {
		side   Side
		target bool
		col    color.RGBA
		label  string
	}
	curves := []curve{
		{Left, false, colSrc, "SRC L"},
		{Right, false, colSrcJoint, "SRC R"},
		{Left, true, colTgt, "TGT L"},
		{Right, true, colTgtJoint, "TGT R"},
	}
	for _, cv := range curves {
		var prevX, prevY int
		for k := 0; k < len(frames); k++ {
			deg := frames[k].Source.KneeDeg[int(cv.side)]
			if cv.target {
				deg = frames[k].Target.KneeDeg[int(cv.side)]
			}
			x, y := mapX(k), mapY(deg)
			if k > 0 {
				fb.Line(prevX, prevY, x, y, cv.col)
			}
			prevX, prevY = x, y
		}
	}

	// Cursor.
	cx := mapX(f)
	fb.Line(cx, plotY0, cx, plotY1, colCop)

	// Legend.
	lx := plotX1 - 360
	for i, cv := range curves {
		x := lx + i*92
		fb.Line(x, y0+12, x+18, y0+12, cv.col)
		fb.DrawText(x+22, y0+8, 1, cv.label, colTextDim)
	}
	fb.DrawText(plotX1-90, plotY0+6, 1, "TURN ZONE", colTurn)
}

func intString(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var buf [12]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}
