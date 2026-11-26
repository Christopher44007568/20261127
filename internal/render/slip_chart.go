package render

import (
	"fmt"
	"math"

	"bipedblend/internal/biped"
)

func SlipChartSVG(result *biped.Result) string {
	const width, height = 1280, 748
	s := newSVG(width, height)
	duration := result.Frames[len(result.Frames)-1].Time
	x := func(t float64) float64 { return 76 + t/duration*1132 }
	speedMax := math.Max(1, math.Ceil(result.Summary.MaxNaiveSlipSpeed*1.15))

	s.text(32, 38, "脚底滑移图：未同步混合 vs 相位同步 + 接触锚定", `font-size="25" font-weight="800" fill="`+colorInk+`"`)
	s.text(32, 64, "橙色为各 Clip 自由跑相位的归一混合；琥珀色为同步相位的原始足底轨迹；蓝色为接触帧锚定后的滑移速度。", `font-size="15" fill="`+colorMuted+`"`)
	drawSlipLegend(s)

	leftBox := chartBox{x0: 76, y0: 92, x1: 1208, y1: 226}
	rightBox := chartBox{x0: 76, y0: 270, x1: 1208, y1: 404}
	drawSpeedPanel(s, result, biped.Left, leftBox, x, speedMax)
	drawSpeedPanel(s, result, biped.Right, rightBox, x, speedMax)

	cumBox := chartBox{x0: 76, y0: 452, x1: 1208, y1: 620}
	drawCumulativePanel(s, result, cumBox, x, duration)

	s.text(76, 668, fmt.Sprintf("峰值滑移：未同步 %.2f m/s；同步原始轨迹 %.2f m/s；锚定后 %.3f m/s。",
		result.Summary.MaxNaiveSlipSpeed, result.Summary.MaxRawSyncedSlipSpeed, result.Summary.MaxLockedSlipSpeed),
		`font-size="15" font-weight="700" fill="`+colorInk+`"`)
	s.text(76, 696, "接触区间内锚定足的世界 X 保持不变，因此累计锚定滑移为 0；速度突变只改变权重过渡，不破坏事件合并。",
		`font-size="14" fill="`+colorMuted+`"`)
	return s.end()
}

func drawSlipLegend(s *svgBuilder) {
	items := []struct{ label, color string }{
		{"未同步/未锚定", colorNaive},
		{"同步但未锚定", colorRaw},
		{"同步+锚定", colorLocked},
	}
	x := 858.0
	for _, item := range items {
		s.line(x, 31, x+30, 31, fmt.Sprintf(`stroke="%s" stroke-width="4"`, item.color))
		s.text(x+38, 36, item.label, `font-size="14" fill="`+colorInk+`"`)
		x += 132
	}
}

func drawSpeedPanel(s *svgBuilder, result *biped.Result, side biped.Side, box chartBox, x xMap, maxValue float64) {
	s.rect(box.x0, box.y0, box.width(), box.height(), 10, `fill="#ffffff" stroke="`+colorGrid+`"`)
	title := "左脚接触滑移速度"
	if side == biped.Right {
		title = "右脚接触滑移速度"
	}
	s.text(box.x0+14, box.y0+24, title, `font-size="15" font-weight="800" fill="`+colorInk+`"`)

	y := func(v float64) float64 { return box.y1 - clampChart(v/maxValue, 0, 1)*box.height() }
	for i := 0; i <= 4; i++ {
		value := maxValue * float64(i) / 4
		py := y(value)
		s.line(box.x0, py, box.x1, py, `stroke="`+colorGrid+`" stroke-width="1"`)
		s.text(box.x0-12, py+4, fmt.Sprintf("%.1f", value), `font-size="11" text-anchor="end" fill="`+colorMuted+`"`)
	}

	naive := make([]point, 0, len(result.Naive))
	raw := make([]point, 0, len(result.Frames))
	locked := make([]point, 0, len(result.Frames))
	for i, frame := range result.Frames {
		np := result.Naive[i]
		var nSlip, rSlip, lSlip float64
		if side == biped.Left {
			nSlip = np.LeftSlipSpeed
			rSlip = frame.LeftSlip.RawSpeed
			lSlip = frame.LeftSlip.LockedSpeed
		} else {
			nSlip = np.RightSlipSpeed
			rSlip = frame.RightSlip.RawSpeed
			lSlip = frame.RightSlip.LockedSpeed
		}
		naive = append(naive, point{x(frame.Time), y(nSlip)})
		raw = append(raw, point{x(frame.Time), y(rSlip)})
		locked = append(locked, point{x(frame.Time), y(lSlip)})
	}
	s.path(areaPath(naive, box.y1), `fill="`+colorNaive+`" opacity="0.12" stroke="none"`)
	s.path(polyPath(naive), `fill="none" stroke="`+colorNaive+`" stroke-width="2.6" stroke-linejoin="round"`)
	s.path(polyPath(raw), `fill="none" stroke="`+colorRaw+`" stroke-width="1.8" opacity="0.9" stroke-linejoin="round"`)
	s.path(polyPath(locked), `fill="none" stroke="`+colorLocked+`" stroke-width="3.2" stroke-linejoin="round"`)
}

func drawCumulativePanel(s *svgBuilder, result *biped.Result, box chartBox, x xMap, duration float64) {
	s.rect(box.x0, box.y0, box.width(), box.height(), 10, `fill="#ffffff" stroke="`+colorGrid+`"`)
	s.text(box.x0+14, box.y0+24, "累计接触滑移距离", `font-size="15" font-weight="800" fill="`+colorInk+`"`)

	maxValue := 0.1
	for i, frame := range result.Frames {
		np := result.Naive[i]
		maxValue = math.Max(maxValue, np.LeftCumulativeM)
		maxValue = math.Max(maxValue, np.RightCumulativeM)
		maxValue = math.Max(maxValue, frame.LeftSlip.RawTravelM)
		maxValue = math.Max(maxValue, frame.RightSlip.RawTravelM)
	}
	maxValue = math.Ceil(maxValue * 1.1)
	y := func(v float64) float64 { return box.y1 - clampChart(v/maxValue, 0, 1)*box.height() }
	for i := 0; i <= 4; i++ {
		value := maxValue * float64(i) / 4
		py := y(value)
		s.line(box.x0, py, box.x1, py, `stroke="`+colorGrid+`" stroke-width="1"`)
		s.text(box.x0-12, py+4, fmt.Sprintf("%.1fm", value), `font-size="11" text-anchor="end" fill="`+colorMuted+`"`)
	}
	for sec := 0; sec <= int(duration); sec++ {
		px := x(float64(sec))
		s.text(px, box.y1+22, fmt.Sprintf("%ds", sec), `font-size="12" text-anchor="middle" fill="`+colorMuted+`"`)
	}

	series := map[string][]point{
		"naiveL": {}, "naiveR": {}, "rawL": {}, "rawR": {}, "lockL": {}, "lockR": {},
	}
	for i, frame := range result.Frames {
		np := result.Naive[i]
		t := frame.Time
		series["naiveL"] = append(series["naiveL"], point{x(t), y(np.LeftCumulativeM)})
		series["naiveR"] = append(series["naiveR"], point{x(t), y(np.RightCumulativeM)})
		series["rawL"] = append(series["rawL"], point{x(t), y(frame.LeftSlip.RawTravelM)})
		series["rawR"] = append(series["rawR"], point{x(t), y(frame.RightSlip.RawTravelM)})
		series["lockL"] = append(series["lockL"], point{x(t), y(frame.LeftSlip.LockedTravelM)})
		series["lockR"] = append(series["lockR"], point{x(t), y(frame.RightSlip.LockedTravelM)})
	}
	s.path(polyPath(series["naiveL"]), `fill="none" stroke="`+colorNaive+`" stroke-width="2.4"`)
	s.path(polyPath(series["naiveR"]), `fill="none" stroke="#dc2626" stroke-width="2.4" stroke-dasharray="7 5"`)
	s.path(polyPath(series["rawL"]), `fill="none" stroke="`+colorRaw+`" stroke-width="1.8" opacity="0.85"`)
	s.path(polyPath(series["rawR"]), `fill="none" stroke="`+colorRaw+`" stroke-width="1.8" stroke-dasharray="5 4" opacity="0.85"`)
	s.path(polyPath(series["lockL"]), `fill="none" stroke="`+colorLocked+`" stroke-width="3"`)
	s.path(polyPath(series["lockR"]), `fill="none" stroke="#0ea5e9" stroke-width="2.5" stroke-dasharray="6 4"`)
	drawCumulativeLegend(s, box)
}

func drawCumulativeLegend(s *svgBuilder, box chartBox) {
	items := []struct{ label, color, dash string }{
		{"未同步左脚", colorNaive, ""},
		{"未同步右脚", colorRun, "7 5"},
		{"同步原始", colorRaw, ""},
		{"锚定双脚", colorLocked, ""},
	}
	lx := box.x0 + 360
	ly := box.y0 + 24
	for _, item := range items {
		attrs := fmt.Sprintf(`stroke="%s" stroke-width="3"`, item.color)
		if item.dash != "" {
			attrs += ` stroke-dasharray="` + item.dash + `"`
		}
		s.line(lx, ly-5, lx+24, ly-5, attrs)
		s.text(lx+31, ly, item.label, `font-size="12" fill="`+colorMuted+`"`)
		lx += 126
	}
}

func areaPath(points []point, baseline float64) string {
	if len(points) == 0 {
		return ""
	}
	d := polyPath(points)
	last := points[len(points)-1]
	first := points[0]
	d += fmt.Sprintf(" L %s %s L %s %s Z", num(last.x), num(baseline), num(first.x), num(baseline))
	return d
}
