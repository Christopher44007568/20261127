package main

import (
	"fmt"
	"image"
	"math"
)

type lineSeries struct {
	name string
	col  RGB
	w    float64
	dash [2]float64 // {dash,gap} pixels; zero means solid
	x, y []float64
}

type hLine struct {
	y     float64
	col   RGB
	dash  [2]float64
	label string
}

type vZone struct {
	x0, x1 float64
	col    RGB
}

type vLine struct {
	x     float64
	col   RGB
	dash  [2]float64
	label string
}

type chartNote struct {
	x, y  float64
	text  string
	col   RGB
	scale int
}

type chartSpec struct {
	title   string
	xLabel  string
	yLabel  string
	xMin    float64
	xMax    float64
	yMin    float64
	yMax    float64
	yTicks  []float64
	xTicks  []float64
	yFormat func(float64) string
	xFormat func(float64) string
	legend  string // "top-left" (default) or "bottom-left"
	series  []lineSeries
	hLines  []hLine
	vLines  []vLine
	zones   []vZone
	notes   []chartNote
}

func defaultFloatFmt(v float64) string { return fmt.Sprintf("%g", v) }

// drawChart renders spec inside box on cv.
func drawChart(cv *Canvas, box image.Rectangle, s chartSpec) {
	cv.fillRect(box.Min.X, box.Min.Y, box.Dx(), box.Dy(), colPaper)

	mL, mR, mT, mB := 78, 26, 58, 52
	pr := image.Rect(box.Min.X+mL, box.Min.Y+mT, box.Max.X-mR, box.Max.Y-mB)
	pcv := cv.panel(pr)

	xr := s.xMax - s.xMin
	yr := s.yMax - s.yMin
	px := func(x float64) int { return pr.Min.X + int(math.Round((x-s.xMin)/xr*float64(pr.Dx()))) }
	py := func(y float64) int { return pr.Max.Y - int(math.Round((y-s.yMin)/yr*float64(pr.Dy()))) }

	// Zones.
	for _, z := range s.zones {
		x0, x1 := px(z.x0), px(z.x1)
		if x0 > x1 {
			x0, x1 = x1, x0
		}
		pcv.fillRect(x0, pr.Min.Y, x1-x0, pr.Dy(), z.col)
	}

	// Grid + tick labels.
	yFmt := s.yFormat
	if yFmt == nil {
		yFmt = defaultFloatFmt
	}
	xFmt := s.xFormat
	if xFmt == nil {
		xFmt = defaultFloatFmt
	}
	for _, t := range s.yTicks {
		yy := py(t)
		cv.line(pr.Min.X, yy, pr.Max.X, yy, 1, colGrid)
		cv.textRight(pr.Min.X-8, yy-7, yFmt(t), 1, colGray)
	}
	for _, t := range s.xTicks {
		xx := px(t)
		cv.line(xx, pr.Min.Y, xx, pr.Max.Y, 1, colGrid)
		txt := xFmt(t)
		w, _ := textSize(txt, 1)
		cv.drawText(xx-w/2, pr.Max.Y+8, txt, 1, colGray)
	}

	// Horizontal limit lines.
	for _, hl := range s.hLines {
		yy := py(hl.y)
		if hl.dash[0] > 0 {
			cv.dashedLine(pr.Min.X, yy, pr.Max.X, yy, 1.5, hl.dash[0], hl.dash[1], hl.col)
		} else {
			cv.line(pr.Min.X, yy, pr.Max.X, yy, 1.5, hl.col)
		}
		if hl.label != "" {
			w, h := textSize(hl.label, 1)
			cv.fillRect(pr.Max.X-w-6, yy-h-4, w+6, h+3, colPaper)
			cv.textRight(pr.Max.X-3, yy-h-3, hl.label, 1, hl.col)
		}
	}

	// Vertical reference lines.
	for _, vl := range s.vLines {
		xx := px(vl.x)
		if vl.dash[0] > 0 {
			cv.dashedLine(xx, pr.Min.Y, xx, pr.Max.Y, 1.5, vl.dash[0], vl.dash[1], vl.col)
		} else {
			cv.line(xx, pr.Min.Y, xx, pr.Max.Y, 1.5, vl.col)
		}
		if vl.label != "" {
			tw, th := textSize(vl.label, 1)
			ly := pr.Max.Y - th - 8
			cv.fillRect(xx+4, ly-2, tw+6, th+4, colPaper)
			cv.drawText(xx+7, ly, vl.label, 1, vl.col)
		}
	}

	// Data series.
	for _, ln := range s.series {
		for i := 1; i < len(ln.x); i++ {
			x0, y0 := px(ln.x[i-1]), py(ln.y[i-1])
			x1, y1 := px(ln.x[i]), py(ln.y[i])
			if ln.dash[0] > 0 {
				pcv.dashedLine(x0, y0, x1, y1, ln.w, ln.dash[0], ln.dash[1], ln.col)
			} else {
				pcv.line(x0, y0, x1, y1, ln.w, ln.col)
			}
		}
	}

	// Frame + labels.
	cv.frameRect(pr, 2, colInk)

	// Data-coordinate notes.
	for _, n := range s.notes {
		sc := n.scale
		if sc == 0 {
			sc = 2
		}
		nx, ny := px(n.x), py(n.y)
		tw, th := textSize(n.text, sc)
		cv.fillRect(nx-5, ny-th-4, tw+10, th+8, colPaper)
		cv.drawText(nx, ny-th-1, n.text, sc, n.col)
	}
	cv.drawText(box.Min.X+10, box.Min.Y+10, s.title, 2, colInk)
	if s.xLabel != "" {
		w, _ := textSize(s.xLabel, 2)
		cv.drawText(pr.Min.X+pr.Dx()/2-w/2, box.Max.Y-24, s.xLabel, 2, colInk)
	}
	if s.yLabel != "" {
		cv.drawText(box.Min.X+10, box.Min.Y+40, s.yLabel, 1, colGray)
	}
	drawLegend(cv, pr, s.series, s.legend)
}

func drawLegend(cv *Canvas, pr image.Rectangle, series []lineSeries, loc string) {
	x := pr.Min.X + 12
	y := pr.Min.Y + 10
	if loc == "bottom-left" {
		y = pr.Max.Y - 24
	}
	for _, ln := range series {
		cv.line(x, y+7, x+26, y+7, 3, ln.col)
		tw, th := textSize(ln.name, 1)
		cv.drawText(x+32, y, ln.name, 1, colInk)
		x += 32 + tw + 18
		_ = th
	}
}

// RenderJointAngles builds the two-leg joint angle curve figure.
func RenderJointAngles(cfg GaitConfig) *image.RGBA {
	const W, H = 1420, 980
	cv := NewCanvas(W, H, colPaper)
	cv.drawText(36, 26, "JOINT ANGLE CURVES - ONE GAIT CYCLE ON 14 DEG SLOPE", 3, colInk)
	cv.drawText(38, 64, "HIP FLEX (+ FORWARD) / KNEE FLEX (0=STRAIGHT, NEVER NEGATIVE) / ANKLE DEV", 1, colGray)

	xs, left, right := gaitAngleData(cfg)

	ticks := []float64{-75, -50, -25, 0, 25, 50, 75, 100, 125}
	xticks := []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0}

	mkSpec := func(title string, ang [3][]float64) chartSpec {
		return chartSpec{
			title: title, xLabel: "GAIT CYCLE PHASE", yLabel: "ANGLE (DEG)",
			xMin: 0, xMax: 1, yMin: -75, yMax: 125,
			yTicks: ticks, xTicks: xticks,
			xFormat: func(v float64) string { return fmt.Sprintf("%.1f", v) },
			series: []lineSeries{
				{name: "HIP", col: colBlue, w: 3, x: xs, y: ang[0]},
				{name: "KNEE", col: colWarn, w: 4, x: xs, y: ang[1]},
				{name: "ANKLE", col: colTeal, w: 3, x: xs, y: ang[2]},
			},
			hLines: []hLine{
				{y: 0, col: mixRGB(colWarn, colPaper, 0.25), dash: [2]float64{7, 5}, label: "KNEE 0: NO-HYPEREXTEND"},
				{y: 108, col: mixRGB(colWarn, colPaper, 0.25), dash: [2]float64{3, 4}, label: "KNEE MAX 108"},
				{y: -42, col: mixRGB(colBlue, colPaper, 0.4), dash: [2]float64{2, 4}, label: "HIP MIN -42"},
				{y: 78, col: mixRGB(colBlue, colPaper, 0.4), dash: [2]float64{2, 4}, label: "HIP MAX 78"},
				{y: -70, col: mixRGB(colTeal, colPaper, 0.45), dash: [2]float64{2, 4}},
				{y: 70, col: mixRGB(colTeal, colPaper, 0.45), dash: [2]float64{2, 4}, label: "ANKLE +-70"},
			},
		}
	}

	drawChart(cv, image.Rect(30, 92, W-30, 520), mkSpec("LEFT / FAR LEG (PHASE +0.5)", left))
	drawChart(cv, image.Rect(30, 540, W-30, 968), mkSpec("RIGHT / NEAR LEG", right))
	return cv.img
}

func gaitAngleData(cfg GaitConfig) (xs []float64, left, right [3][]float64) {
	const n = 140
	for i := 0; i <= n; i++ {
		phase := float64(i) / n
		f := cfg.SampleFrame(phase * cfg.CycleT)
		xs = append(xs, phase)
		lf, rf := f.Feet[0], f.Feet[1]
		left[0] = append(left[0], deg(lf.Pose.HipFlex))
		left[1] = append(left[1], deg(lf.Pose.KneeFlex))
		left[2] = append(left[2], deg(lf.Pose.AnkleDev))
		right[0] = append(right[0], deg(rf.Pose.HipFlex))
		right[1] = append(right[1], deg(rf.Pose.KneeFlex))
		right[2] = append(right[2], deg(rf.Pose.AnkleDev))
	}
	return
}
