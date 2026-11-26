package report

import (
	"fmt"
	"math"
	"strings"

	"characterbounds/hierbox"
)

func VolumeChartSVG(results []*hierbox.PoseResult) string {
	width, height := 1120, 500
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" font-family="ui-sans-serif,system-ui,'Noto Sans CJK SC','Microsoft YaHei','Droid Sans Fallback',sans-serif">`, width, height, width, height)
	b.WriteString(`<rect width="100%" height="100%" fill="#f8fafc"/>`)
	b.WriteString(`<text x="32" y="38" font-size="22" font-weight="700" fill="#0f172a">Bounding volume by bone depth / 逐层体积图</text>`)
	b.WriteString(`<text x="32" y="62" font-size="13" fill="#475569">Log10(m3): bars sum all node AABB volumes at one topology depth. Layers are not expected to be monotonic.</text>`)

	x0, y0, pw, ph := 92.0, 92.0, 970.0, 310.0
	fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" fill="#fff" stroke="#cbd5e1" rx="10"/>`, x0-18, y0-18, pw+36, ph+58)

	minLog, maxLog := volumeDomain(results)
	yFor := func(v float64) float64 {
		lv := minLog
		if v > 0 {
			lv = math.Max(minLog, math.Min(maxLog, math.Log10(v)))
		}
		return y0 + ph - (lv-minLog)/(maxLog-minLog)*ph
	}

	for k := int(minLog); k <= int(maxLog); k++ {
		y := yFor(math.Pow(10, float64(k)))
		fmt.Fprintf(&b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#e2e8f0"/>`, x0, y, x0+pw, y)
		label := fmt.Sprintf("10^%d", k)
		if k == 0 {
			label = "1 m³"
		}
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" text-anchor="end" font-size="11" fill="#64748b">%s</text>`, x0-10, y+4, label)
	}

	depthCount := 0
	if len(results) > 0 {
		depthCount = len(results[0].LayerVolumes)
	}
	groupW := pw / float64(maxInt(depthCount, 1))
	barW := math.Min(34, groupW/5.2)
	poseColors := []string{"#7c3aed", "#ea580c", "#0891b2"}

	for d := 0; d < depthCount; d++ {
		gx := x0 + groupW*float64(d) + groupW/2
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" text-anchor="middle" font-size="12" font-weight="700" fill="#334155">depth %d</text>`, gx, y0+ph+25, d)
		if len(results) > 0 && d < len(results[0].LayerVolumes) {
			fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" text-anchor="middle" font-size="10" fill="#94a3b8">%d nodes</text>`, gx, y0+ph+41, results[0].LayerVolumes[d].Nodes)
		}
		for pi, res := range results {
			if d >= len(res.LayerVolumes) {
				continue
			}
			v := res.LayerVolumes[d].VolumeSum
			y := yFor(v)
			bh := y0 + ph - y
			x := gx - barW*1.6 + float64(pi)*barW*1.15
			color := poseColors[pi%len(poseColors)]
			fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="%.1f" height="%.1f" rx="4" fill="%s"><title>%s depth=%d sum=%.6f m³ union=%.6f m³</title></rect>`,
				x, y, barW, bh, color, englishPoseTitle(res.Name), d, v, res.LayerVolumes[d].UnionVolume)
			if v >= 0.01 {
				fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" text-anchor="middle" font-size="9" fill="#334155">%.3f</text>`, x+barW/2, y-5, v)
			}
		}
	}

	lx := 760.0
	ly := 470.0
	for i, res := range results {
		x := lx + float64(i)*120
		fmt.Fprintf(&b, `<rect x="%.1f" y="%.1f" width="14" height="14" rx="3" fill="%s"/><text x="%.1f" y="%.1f" font-size="12" fill="#334155">%s</text>`,
			x, ly-12, poseColors[i%len(poseColors)], x+20, ly, englishPoseTitle(res.Name))
	}
	b.WriteString(`</svg>`)
	return b.String()
}

func volumeDomain(results []*hierbox.PoseResult) (float64, float64) {
	minV := math.Inf(1)
	maxV := 0.0
	for _, res := range results {
		for _, layer := range res.LayerVolumes {
			if layer.VolumeSum > 1e-12 && layer.VolumeSum < minV {
				minV = layer.VolumeSum
			}
			if layer.VolumeSum > maxV {
				maxV = layer.VolumeSum
			}
		}
	}
	if math.IsInf(minV, 1) {
		minV = 1e-6
	}
	minLog := math.Floor(math.Log10(minV))
	maxLog := math.Ceil(math.Log10(maxV))
	if maxLog < 0 {
		maxLog = 0
	}
	if maxLog-minLog < 3 {
		minLog = maxLog - 3
	}
	return minLog, maxLog
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func VolumeCSV(results []*hierbox.PoseResult) string {
	var b strings.Builder
	b.WriteString("pose,depth,nodes,volume_sum_m3,union_volume_m3\n")
	for _, res := range results {
		for _, layer := range res.LayerVolumes {
			fmt.Fprintf(&b, "%s,%d,%d,%.9f,%.9f\n", res.Name, layer.Depth, layer.Nodes, layer.VolumeSum, layer.UnionVolume)
		}
	}
	return b.String()
}
