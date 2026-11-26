package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteAreaChartSVG renders the continuous vector counter-area curve. SVG is
// used only as a tiny dependency-free vector chart format.
func WriteAreaChartSVG(path string, metrics []SizeMetric) error {
	if len(metrics) == 0 {
		return fmt.Errorf("no metrics to chart")
	}
	const width, height = 920, 560
	const left, right, top, bottom = 82.0, 34.0, 54.0, 70.0
	plotW := float64(width) - left - right
	plotH := float64(height) - top - bottom

	minSize := metrics[0].FontSize
	maxSize := metrics[len(metrics)-1].FontSize
	maxArea := 0.0
	for _, m := range metrics {
		if m.CounterAreaPixels > maxArea {
			maxArea = m.CounterAreaPixels
		}
	}
	if maxArea <= 0 || maxSize == minSize {
		return fmt.Errorf("invalid chart ranges: maxArea=%v sizeRange=%v", maxArea, maxSize-minSize)
	}

	x := func(size float64) float64 {
		return left + (size-minSize)/(maxSize-minSize)*plotW
	}
	y := func(area float64) float64 {
		return top + plotH - area/maxArea*plotH
	}

	var line strings.Builder
	for i, m := range metrics {
		if i > 0 {
			line.WriteByte(' ')
		}
		fmt.Fprintf(&line, "%.1f,%.1f", x(m.FontSize), y(m.CounterAreaPixels))
	}

	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`, width, height, width, height)
	b.WriteString(`<rect width="100%" height="100%" fill="#f8f5ef"/>`)
	b.WriteString(`<text x="460" y="32" text-anchor="middle" font-family="sans-serif" font-size="22" font-weight="700" fill="#172a3a">Counter area is continuous across font sizes</text>`)

	for i := 0; i <= 5; i++ {
		area := maxArea * float64(i) / 5
		yy := y(area)
		fmt.Fprintf(&b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#d8d1c4" stroke-width="1"/>`, left, yy, left+plotW, yy)
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" text-anchor="end" font-family="sans-serif" font-size="13" fill="#5b6470">%.1f</text>`, left-10, yy+4, area)
	}
	for _, size := range []float64{12, 24, 36, 48, 72, 96, 128} {
		if size < minSize || size > maxSize {
			continue
		}
		xx := x(size)
		fmt.Fprintf(&b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#e5ded1" stroke-width="1"/>`, xx, top, xx, top+plotH)
		fmt.Fprintf(&b, `<text x="%.1f" y="%.1f" text-anchor="middle" font-family="sans-serif" font-size="13" fill="#5b6470">%.0f</text>`, xx, top+plotH+24, size)
	}

	fmt.Fprintf(&b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#172a3a" stroke-width="2"/>`, left, top, left, top+plotH)
	fmt.Fprintf(&b, `<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="#172a3a" stroke-width="2"/>`, left, top+plotH, left+plotW, top+plotH)
	fmt.Fprintf(&b, `<polyline points="%s" fill="none" stroke="#0f766e" stroke-width="4" stroke-linejoin="round" stroke-linecap="round"/>`, line.String())
	for _, m := range metrics {
		fmt.Fprintf(&b, `<circle cx="%.1f" cy="%.1f" r="3.2" fill="#f59e0b" stroke="#0f766e" stroke-width="1.5"/>`, x(m.FontSize), y(m.CounterAreaPixels))
	}
	b.WriteString(`<text x="460" y="548" text-anchor="middle" font-family="sans-serif" font-size="16" fill="#172a3a">font size (px/em)</text>`)
	fmt.Fprintf(&b, `<text x="22" y="280" text-anchor="middle" transform="rotate(-90 22 280)" font-family="sans-serif" font-size="16" fill="#172a3a">counter area (px²), wght=%d</text>`, metrics[0].Weight)
	b.WriteString(`</svg>`)

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
