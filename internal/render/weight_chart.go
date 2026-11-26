package render

import (
	"fmt"
	"math"

	"bipedblend/internal/biped"
)

type chartBox struct {
	x0, y0, x1, y1 float64
}

func (b chartBox) width() float64  { return b.x1 - b.x0 }
func (b chartBox) height() float64 { return b.y1 - b.y0 }

func WeightChartSVG(result *biped.Result) string {
	const width, height = 1280, 540
	s := newSVG(width, height)
	plot := chartBox{x0: 76, y0: 96, x1: 1208, y1: 386}
	duration := result.Frames[len(result.Frames)-1].Time
	x := func(t float64) float64 { return plot.x0 + t/duration*plot.width() }
	wy := func(w float64) float64 { return plot.y1 - clampChart(w, 0, 1)*plot.height() }
	sy := func(v float64) float64 { return plot.y1 - clampChart(v/5, 0, 1)*plot.height() }

	s.text(32, 38, "权重归一与速度平滑曲线", `font-size="25" font-weight="800" fill="`+colorInk+`"`)
	s.text(32, 64, "实线为平滑后的归一权重；同色虚线为瞬时目标权重，紫色为速度突变后的指数平滑。", `font-size="15" fill="`+colorMuted+`"`)
	drawWeightLegend(s)

	s.rect(plot.x0, plot.y0, plot.width(), plot.height(), 10, `fill="#ffffff" stroke="`+colorGrid+`"`)
	for i := 0; i <= 4; i++ {
		value := float64(i) / 4
		y := wy(value)
		s.line(plot.x0, y, plot.x1, y, `stroke="`+colorGrid+`" stroke-width="1"`)
		s.text(plot.x0-14, y+5, fmt.Sprintf("%.2f", value), `font-size="12" text-anchor="end" fill="`+colorMuted+`"`)
		speedValue := float64(i) * 1.25
		s.text(plot.x1+12, y+5, fmt.Sprintf("%.1f", speedValue), `font-size="12" fill="`+colorSpeed+`"`)
	}
	for sec := 0; sec <= int(math.Ceil(duration)); sec++ {
		px := x(float64(sec))
		s.line(px, plot.y0, px, plot.y1, `stroke="`+colorGrid+`" stroke-width="1"`)
		s.text(px, plot.y1+24, fmt.Sprintf("%ds", sec), `font-size="12" text-anchor="middle" fill="`+colorMuted+`"`)
	}
	s.text(24, plot.y0-12, "权重", `font-size="13" font-weight="700" fill="`+colorMuted+`"`)
	s.text(plot.x1-70, plot.y0-12, "速度 m/s", `font-size="13" font-weight="700" fill="`+colorSpeed+`"`)

	drawWeightSeries(s, result, "idle", colorIdle, x, wy, true)
	drawWeightSeries(s, result, "walk", colorWalk, x, wy, true)
	drawWeightSeries(s, result, "run", colorRun, x, wy, true)
	drawSpeedSeries(s, result, x, sy)
	drawEventLane(s, result, x, plot)

	s.text(plot.x0, 462, fmt.Sprintf("归一校验：min Σw=%.6f，max Σw=%.6f，最大误差 %.2e",
		result.Summary.WeightSumMin, result.Summary.WeightSumMax, result.Summary.WeightSumMaxError),
		`font-size="14" font-weight="700" fill="`+colorInk+`"`)
	s.text(plot.x0, 488, fmt.Sprintf("事件帧：原始 %d 条 → 邻近/交叠合并为 %d 个事件；边沿触发 %d 次，无重复触发。",
		result.Summary.RawEventCount, result.Summary.MergedEventCount, result.Summary.TriggeredEventCount),
		`font-size="14" fill="`+colorMuted+`"`)
	return s.end()
}

func clampChart(v, lo, hi float64) float64 {
	return math.Max(lo, math.Min(hi, v))
}

type xMap func(float64) float64
type yMap func(float64) float64

func drawWeightSeries(s *svgBuilder, result *biped.Result, id, color string, x xMap, y yMap, target bool) {
	actual := make([]point, 0, len(result.Frames))
	targets := make([]point, 0, len(result.Frames))
	for _, frame := range result.Frames {
		actual = append(actual, point{x(frame.Time), y(biped.WeightValue(frame.Weights, id))})
		if target {
			targets = append(targets, point{x(frame.Time), y(biped.WeightValue(frame.TargetWeights, id))})
		}
	}
	if target {
		s.path(polyPath(targets), `fill="none" stroke="`+color+`" stroke-width="1.6" stroke-dasharray="6 5" opacity="0.38"`)
	}
	s.path(polyPath(actual), `fill="none" stroke="`+color+`" stroke-width="3.2" stroke-linejoin="round" stroke-linecap="round"`)
}

func drawSpeedSeries(s *svgBuilder, result *biped.Result, x xMap, y yMap) {
	smooth := make([]point, 0, len(result.Frames))
	target := make([]point, 0, len(result.Frames))
	for _, frame := range result.Frames {
		smooth = append(smooth, point{x(frame.Time), y(frame.Speed)})
		target = append(target, point{x(frame.Time), y(frame.TargetSpeed)})
	}
	s.path(polyPath(target), `fill="none" stroke="`+colorInk+`" stroke-width="1.8" stroke-dasharray="3 6" opacity="0.45"`)
	s.path(polyPath(smooth), `fill="none" stroke="`+colorSpeed+`" stroke-width="3" stroke-linecap="round"`)
}

type point struct{ x, y float64 }

func polyPath(points []point) string {
	if len(points) == 0 {
		return ""
	}
	d := fmt.Sprintf("M %s %s", num(points[0].x), num(points[0].y))
	for _, p := range points[1:] {
		d += fmt.Sprintf(" L %s %s", num(p.x), num(p.y))
	}
	return d
}

func drawWeightLegend(s *svgBuilder) {
	items := []struct {
		label, color string
		dash         string
	}{
		{"待机 w", colorIdle, ""},
		{"行走 w", colorWalk, ""},
		{"跑步 w", colorRun, ""},
		{"平滑速度", colorSpeed, ""},
		{"目标/瞬时权重", colorMuted, "6 5"},
	}
	x := 620.0
	y := 34.0
	for _, item := range items {
		attrs := fmt.Sprintf(`stroke="%s" stroke-width="4" stroke-linecap="round"`, item.color)
		if item.dash != "" {
			attrs += ` stroke-dasharray="` + item.dash + `"`
		}
		s.line(x, y-5, x+28, y-5, attrs)
		s.text(x+36, y, item.label, `font-size="14" fill="`+colorInk+`"`)
		x += 126
	}
}

func drawEventLane(s *svgBuilder, result *biped.Result, x xMap, plot chartBox) {
	laneY := plot.y1 + 48
	s.line(plot.x0, laneY, plot.x1, laneY, `stroke="#94a3b8" stroke-width="2"`)
	s.text(plot.x0, laneY-12, "合并后的接触事件帧", `font-size="13" font-weight="700" fill="`+colorMuted+`"`)
	for _, event := range result.Events {
		px := x(float64(event.Frame) / float64(biped.ScenarioFPS))
		color := colorLeft
		switch event.Kind {
		case biped.EventRightHeel:
			color = colorRight
		case biped.EventLeftToe, biped.EventRightToe:
			color = colorMuted
		}
		if len(event.SourceIDs) >= 2 {
			s.circle(px, laneY, 7, `fill="none" stroke="`+colorSpeed+`" stroke-width="1.8"`)
		}
		r := 4.5
		if event.Kind == biped.EventLeftToe || event.Kind == biped.EventRightToe {
			r = 3
		}
		s.circle(px, laneY, r, `fill="`+color+`" stroke="#fff" stroke-width="1.2"`)
	}
	s.circle(plot.x0+210, laneY-16, 5, `fill="`+colorLeft+`"`)
	s.text(plot.x0+220, laneY-12, "左脚跟", `font-size="12" fill="`+colorMuted+`"`)
	s.circle(plot.x0+286, laneY-16, 5, `fill="`+colorRight+`"`)
	s.text(plot.x0+296, laneY-12, "右脚跟", `font-size="12" fill="`+colorMuted+`"`)
	s.circle(plot.x0+362, laneY-16, 3.5, `fill="`+colorMuted+`"`)
	s.text(plot.x0+372, laneY-12, "脚尖离地；紫圈=多 Clip 合并", `font-size="12" fill="`+colorMuted+`"`)
}
