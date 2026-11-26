package report

import (
	"bytes"
	"fmt"
	"html"
	"math"
	"strings"

	"skinlab/internal/skin"
)

func svgFloat(value float64) string {
	if math.Abs(value) < 0.0005 {
		value = 0
	}
	return fmt.Sprintf("%.2f", value)
}

func svgText(value string) string {
	return html.EscapeString(value)
}

func lineSegments(points [][2]float64, stroke string, width float64, dash string) string {
	var builder strings.Builder
	dashAttribute := ""
	if dash != "" {
		dashAttribute = fmt.Sprintf(` stroke-dasharray="%s"`, dash)
	}
	for index := 1; index < len(points); index++ {
		fmt.Fprintf(
			&builder,
			`<line x1="%s" y1="%s" x2="%s" y2="%s" stroke="%s" stroke-width="%s" stroke-linecap="round"%s/>`,
			svgFloat(points[index-1][0]), svgFloat(points[index-1][1]),
			svgFloat(points[index][0]), svgFloat(points[index][1]),
			stroke, svgFloat(width), dashAttribute,
		)
	}
	return builder.String()
}

func writeHeatLegend(buffer *bytes.Buffer, x, y, width, height float64) {
	steps := 30
	stepWidth := width / float64(steps)
	for step := 0; step < steps; step++ {
		ratio := 0.25 + 1.5*float64(step)/float64(steps-1)
		fmt.Fprintf(
			buffer,
			`<rect x="%s" y="%s" width="%s" height="%s" fill="%s"/>`,
			svgFloat(x+float64(step)*stepWidth), svgFloat(y), svgFloat(stepWidth+0.15), svgFloat(height), heatColor(ratio),
		)
	}
}

func RenderAreaChartSVG(areas []skin.AreaSample) string {
	const (
		width       = 1000
		height      = 330
		leftMargin  = 68.0
		rightMargin = 28.0
		topMargin   = 58.0
		bottom      = 54.0
	)
	plotWidth := width - leftMargin - rightMargin
	plotHeight := height - topMargin - bottom

	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, area := range areas {
		minX = math.Min(minX, area.X)
		maxX = math.Max(maxX, area.X)
		minY = math.Min(minY, math.Min(area.LBSRatio, area.DQRatio))
		maxY = math.Max(maxY, math.Max(area.LBSRatio, area.DQRatio))
	}
	minY = math.Floor((minY-0.05)*10) / 10
	maxY = math.Ceil((maxY+0.05)*10) / 10
	minY = math.Min(minY, 0.9)
	maxY = math.Max(maxY, 1.1)

	mapX := func(x float64) float64 { return leftMargin + (x-minX)/(maxX-minX)*plotWidth }
	mapY := func(y float64) float64 { return topMargin + (maxY-y)/(maxY-minY)*plotHeight }

	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, `<svg class="chart" viewBox="0 0 %d %d" role="img" aria-label="截面积曲线">`, width, height)
	buffer.WriteString(`<rect width="1000" height="330" rx="18" fill="#f7f5ef"/>`)
	fmt.Fprintf(&buffer, `<text x="%.0f" y="28" fill="#202b36" font-size="14" font-weight="700">截面积沿上肢轴线的变化</text>`, leftMargin)
	buffer.WriteString(`<text x="972" y="28" text-anchor="end" fill="#667483" font-size="11">变形后面积 / 绑定姿态面积</text>`)

	zoneLeft := math.Max(leftMargin, mapX(-0.38))
	zoneRight := math.Min(leftMargin+plotWidth, mapX(0.38))
	fmt.Fprintf(&buffer, `<rect x="%s" y="%s" width="%s" height="%s" fill="#e9dcc2" opacity=".55"/>`, svgFloat(zoneLeft), svgFloat(topMargin), svgFloat(zoneRight-zoneLeft), svgFloat(plotHeight))
	fmt.Fprintf(&buffer, `<text x="%s" y="%s" text-anchor="middle" fill="#8a6d3b" font-size="10">肘部权重过渡区</text>`, svgFloat((zoneLeft+zoneRight)/2), svgFloat(topMargin-10))

	for tick := math.Ceil(minY*10) / 10; tick <= maxY+0.001; tick += 0.1 {
		y := mapY(tick)
		strokeWidth := "0.5"
		if math.Abs(tick-1) < 0.001 {
			strokeWidth = "1"
		}
		fmt.Fprintf(&buffer, `<line x1="%s" y1="%s" x2="%s" y2="%s" stroke="#d9d5cb" stroke-width="%s"/>`, svgFloat(leftMargin), svgFloat(y), svgFloat(leftMargin+plotWidth), svgFloat(y), strokeWidth)
		fmt.Fprintf(&buffer, `<text x="%s" y="%s" text-anchor="end" fill="#667483" font-size="10">%s%%</text>`, svgFloat(leftMargin-9), svgFloat(y+3), svgFloat(tick*100))
	}
	for _, tick := range []float64{-1.5, -1, -0.5, 0, 0.5, 1, 1.5} {
		x := mapX(tick)
		fmt.Fprintf(&buffer, `<line x1="%s" y1="%s" x2="%s" y2="%s" stroke="#d9d5cb" stroke-width=".5"/>`, svgFloat(x), svgFloat(topMargin), svgFloat(x), svgFloat(topMargin+plotHeight))
		fmt.Fprintf(&buffer, `<text x="%s" y="%s" text-anchor="middle" fill="#667483" font-size="10">%s</text>`, svgFloat(x), svgFloat(topMargin+plotHeight+20), svgFloat(tick))
	}
	buffer.WriteString(`<text x="500" y="322" text-anchor="middle" fill="#667483" font-size="10">绑定姿态轴向坐标 X（肩 → 手）</text>`)

	lbsPath := make([][2]float64, 0, len(areas))
	dqPath := make([][2]float64, 0, len(areas))
	for _, area := range areas {
		lbsPath = append(lbsPath, [2]float64{mapX(area.X), mapY(area.LBSRatio)})
		dqPath = append(dqPath, [2]float64{mapX(area.X), mapY(area.DQRatio)})
	}

	buffer.WriteString(lineSegments(lbsPath, "#d65a3a", 3, ""))
	buffer.WriteString(lineSegments(dqPath, "#227f82", 3, ""))

	lbsMin, dqMax := areas[0], areas[0]
	for _, area := range areas {
		if area.LBSRatio < lbsMin.LBSRatio {
			lbsMin = area
		}
		if area.DQRatio > dqMax.DQRatio {
			dqMax = area
		}
	}
	drawExtremum(&buffer, mapX(lbsMin.X), mapY(lbsMin.LBSRatio), "#d65a3a", fmt.Sprintf("LBS 最小 %.1f%%", lbsMin.LBSRatio*100), -8, 22, "end")
	drawExtremum(&buffer, mapX(dqMax.X), mapY(dqMax.DQRatio), "#227f82", fmt.Sprintf("DQ 最大 %.1f%%", dqMax.DQRatio*100), 8, -12, "start")

	buffer.WriteString(`<g transform="translate(790 40)" font-size="11" font-family="ui-sans-serif,sans-serif">`)
	buffer.WriteString(`<line x1="0" y1="0" x2="24" y2="0" stroke="#d65a3a" stroke-width="3"/><text x="31" y="4" fill="#3d4853">LBS</text>`)
	buffer.WriteString(`<line x1="78" y1="0" x2="102" y2="0" stroke="#227f82" stroke-width="3"/><text x="109" y="4" fill="#3d4853">双四元数</text></g>`)
	buffer.WriteString(`</svg>`)
	return buffer.String()
}

func drawExtremum(buffer *bytes.Buffer, x, y float64, color, label string, dx, dy float64, anchor string) {
	fmt.Fprintf(buffer, `<circle cx="%s" cy="%s" r="5" fill="#f7f5ef" stroke="%s" stroke-width="3"/>`, svgFloat(x), svgFloat(y), color)
	fmt.Fprintf(buffer, `<text x="%s" y="%s" text-anchor="%s" fill="%s" font-size="11" font-weight="700">%s</text>`, svgFloat(x+dx), svgFloat(y+dy), anchor, color, svgText(label))
}

func RenderTrajectorySVG(samples []skin.TrajectorySample, markers []int) string {
	const (
		width       = 1000
		height      = 380
		leftMargin  = 66.0
		rightMargin = 32.0
		topMargin   = 76.0
		bottom      = 54.0
	)
	plotWidth := width - leftMargin - rightMargin
	plotHeight := height - topMargin - bottom

	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := math.Inf(1), math.Inf(-1)
	for _, sample := range samples {
		minX = math.Min(minX, sample.Position.X)
		maxX = math.Max(maxX, sample.Position.X)
		minY = math.Min(minY, sample.Position.Y)
		maxY = math.Max(maxY, sample.Position.Y)
	}
	paddingX := math.Max((maxX-minX)*0.1, 0.12)
	paddingY := math.Max((maxY-minY)*0.1, 0.12)
	minX -= paddingX
	maxX += paddingX
	minY -= paddingY
	maxY += paddingY
	mapX := func(x float64) float64 { return leftMargin + (x-minX)/(maxX-minX)*plotWidth }
	mapY := func(y float64) float64 { return topMargin + (maxY-y)/(maxY-minY)*plotHeight }

	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, `<svg class="chart" viewBox="0 0 %d %d" role="img" aria-label="顶点轨迹">`, width, height)
	buffer.WriteString(`<rect width="1000" height="380" rx="18" fill="#111820"/>`)
	buffer.WriteString(`<text x="66" y="28" fill="#f1f4f6" font-size="14" font-weight="700">关键顶点轨迹</text>`)
	buffer.WriteString(`<text x="968" y="28" text-anchor="end" fill="#91a0ae" font-size="11">X–Y 侧视投影 · 完整 XYZ 见 CSV</text>`)

	for index := 0; index <= 4; index++ {
		x := leftMargin + plotWidth*float64(index)/4
		y := topMargin + plotHeight*float64(index)/4
		fmt.Fprintf(&buffer, `<line x1="%s" y1="%s" x2="%s" y2="%s" stroke="#2c3946" stroke-width=".7"/>`, svgFloat(x), svgFloat(topMargin), svgFloat(x), svgFloat(topMargin+plotHeight))
		fmt.Fprintf(&buffer, `<line x1="%s" y1="%s" x2="%s" y2="%s" stroke="#2c3946" stroke-width=".7"/>`, svgFloat(leftMargin), svgFloat(y), svgFloat(leftMargin+plotWidth), svgFloat(y))
	}
	buffer.WriteString(`<text x="500" y="368" text-anchor="middle" fill="#91a0ae" font-size="10">绑定姿态世界坐标 X</text>`)
	fmt.Fprintf(&buffer, `<text transform="translate(18 %s) rotate(-90)" text-anchor="middle" fill="#91a0ae" font-size="10">绑定姿态世界坐标 Y</text>`, svgFloat(topMargin+plotHeight/2))

	colors := []string{"#e7a33e", "#64c7df", "#c992e8"}
	dashStyle := map[string]string{
		"LBS": "6 5",
		"DQ":  "",
	}

	for markerIndex, vertex := range markers {
		for _, method := range []string{"LBS", "DQ"} {
			points := make([][2]float64, 0)
			for _, sample := range samples {
				if sample.Vertex == vertex && sample.Method == method {
					points = append(points, [2]float64{mapX(sample.Position.X), mapY(sample.Position.Y)})
				}
			}
			if len(points) == 0 {
				continue
			}
			fmt.Fprintf(
				&buffer,
				`<g opacity=".9">%s</g>`,
				lineSegments(points, colors[markerIndex%len(colors)], 2, dashStyle[method]),
			)
			end := points[len(points)-1]
			fill := colors[markerIndex%len(colors)]
			if method == "LBS" {
				fill = "#111820"
			}
			fmt.Fprintf(&buffer, `<circle cx="%s" cy="%s" r="4" fill="%s" stroke="%s" stroke-width="2"/>`, svgFloat(end[0]), svgFloat(end[1]), fill, colors[markerIndex%len(colors)])
			start := points[0]
			fmt.Fprintf(&buffer, `<circle cx="%s" cy="%s" r="3" fill="#111820" stroke="%s" stroke-width="2"/>`, svgFloat(start[0]), svgFloat(start[1]), colors[markerIndex%len(colors)])
		}
	}

	for index, vertex := range markers {
		ring := 0
		segment := 0
		foreWeight := 0.0
		for _, sample := range samples {
			if sample.Vertex == vertex {
				ring, segment = sample.Ring, sample.Segment
				foreWeight = sample.Weights[1]
				break
			}
		}
		x := leftMargin + float64(index)*222
		fmt.Fprintf(&buffer, `<line x1="%s" y1="50" x2="%s" y2="50" stroke="%s" stroke-width="3"/>`, svgFloat(x), svgFloat(x+22), colors[index%len(colors)])
		fmt.Fprintf(&buffer, `<text x="%s" y="54" fill="#dce3e9" font-size="10">V%d · 环 %d / 段 %d · w前臂 %.2f</text>`, svgFloat(x+28), vertex, ring, segment, foreWeight)
	}
	buffer.WriteString(`<g transform="translate(780 64)" fill="#dce3e9" font-size="10"><line x1="0" y1="0" x2="22" y2="0" stroke="#91a0ae" stroke-width="2" stroke-dasharray="6 5"/><text x="28" y="4">LBS</text><line x1="56" y1="0" x2="78" y2="0" stroke="#91a0ae" stroke-width="2"/><text x="84" y="4">DQ</text></g>`)
	buffer.WriteString(`</svg>`)
	return buffer.String()
}

func RenderHeatmapSVG(mesh skin.Mesh, quads []skin.QuadSample) string {
	const (
		width      = 1000
		height     = 404
		top        = 78.0
		cellWidth  = 7.0
		cellHeight = 4.15
	)
	panels := []struct {
		title string
		left  float64
	}{
		{title: "LBS · 局部面积变形", left: 74},
		{title: "双四元数 · 局部面积变形", left: 552},
	}

	var buffer bytes.Buffer
	fmt.Fprintf(&buffer, `<svg class="chart" viewBox="0 0 %d %d" role="img" aria-label="展开表面变形热图">`, width, height)
	buffer.WriteString(`<rect width="1000" height="404" rx="18" fill="#f7f5ef"/>`)
	buffer.WriteString(`<text x="66" y="28" fill="#202b36" font-size="14" font-weight="700">展开表面变形热图</text>`)
	buffer.WriteString(`<text x="968" y="28" text-anchor="end" fill="#667483" font-size="11">每一格为四边形面积比 · 蓝=压缩 · 红=鼓包</text>`)

	cellRings := mesh.RingCount - 1
	for panelIndex, panel := range panels {
		minRatio, maxRatio := 1.0, 1.0
		for _, quad := range quads {
			ratio := quad.LBSRatio
			if panelIndex == 1 {
				ratio = quad.DQRatio
			}
			minRatio = math.Min(minRatio, ratio)
			maxRatio = math.Max(maxRatio, ratio)
		}
		fmt.Fprintf(&buffer, `<text x="%s" y="58" fill="#202b36" font-size="12" font-weight="700">%s</text>`, svgFloat(panel.left), svgText(panel.title))
		fmt.Fprintf(&buffer, `<text x="%s" y="58" text-anchor="end" fill="#667483" font-size="10">%.1f%%–%.1f%%</text>`, svgFloat(panel.left+cellWidth*float64(cellRings)), minRatio*100, maxRatio*100)

		for ring := 0; ring < cellRings; ring++ {
			for segment := 0; segment < mesh.Segments; segment++ {
				quad := quads[ring*mesh.Segments+segment]
				ratio := quad.LBSRatio
				if panelIndex == 1 {
					ratio = quad.DQRatio
				}
				x := panel.left + float64(ring)*cellWidth
				y := top + float64(segment)*cellHeight
				fmt.Fprintf(&buffer, `<rect x="%s" y="%s" width="%s" height="%s" fill="%s"/>`, svgFloat(x), svgFloat(y), svgFloat(cellWidth+.15), svgFloat(cellHeight+.15), heatColor(ratio))
			}
		}
		plotRight := panel.left + cellWidth*float64(cellRings)
		plotBottom := top + cellHeight*float64(mesh.Segments)
		fmt.Fprintf(&buffer, `<rect x="%s" y="%s" width="%s" height="%s" fill="none" stroke="#b9b5aa" stroke-width=".8"/>`, svgFloat(panel.left), svgFloat(top), svgFloat(plotRight-panel.left), svgFloat(plotBottom-top))

		if panelIndex == 0 {
			for _, tick := range []int{0, 12, 24, 36, 48} {
				y := top + float64(tick)*cellHeight
				label := fmt.Sprintf("%d°", tick*360/mesh.Segments)
				fmt.Fprintf(&buffer, `<text x="%s" y="%s" text-anchor="end" fill="#667483" font-size="9">%s</text>`, svgFloat(panel.left-8), svgFloat(y+3), label)
			}
			buffer.WriteString(`<text transform="translate(22 184) rotate(-90)" text-anchor="middle" fill="#667483" font-size="9">圆周角</text>`)
		}
		for _, tick := range []float64{-1.5, -1, -0.5, 0, 0.5, 1, 1.5} {
			x := panel.left + mapRange(tick, mesh.RingX[0], mesh.RingX[len(mesh.RingX)-1])*cellWidth*float64(cellRings)
			fmt.Fprintf(&buffer, `<text x="%s" y="%s" text-anchor="middle" fill="#667483" font-size="9">%s</text>`, svgFloat(x), svgFloat(plotBottom+16), svgFloat(tick))
		}
		buffer.WriteString(`<text x="` + svgFloat(panel.left+cellWidth*float64(cellRings)/2) + `" y="392" text-anchor="middle" fill="#667483" font-size="9">轴向坐标 X</text>`)
	}

	buffer.WriteString(`<g transform="translate(452 365)">`)
	writeHeatLegend(&buffer, 0, 0, 96, 8)
	buffer.WriteString(`<text x="0" y="22" fill="#667483" font-size="9">压缩</text><text x="48" y="22" text-anchor="middle" fill="#667483" font-size="9">100%</text><text x="96" y="22" text-anchor="end" fill="#667483" font-size="9">鼓包</text></g>`)
	buffer.WriteString(`</svg>`)
	return buffer.String()
}

func mapRange(value, minimum, maximum float64) float64 {
	if maximum == minimum {
		return 0
	}
	return (value - minimum) / (maximum - minimum)
}
