package report

import (
	"fmt"
	"io"
	"math"
	"strings"
)

// Plot colors shared across every chart so methods stay visually consistent.
var seriesColors = map[string]string{
	"euler-lerp":       "#d62728",
	"quat-nlerp-raw":   "#ff7f0e",
	"quat-slerp-short": "#2ca02c",
}

const (
	axisColor   = "#333333"
	gridColor   = "#e2e2e2"
	textColor   = "#222222"
	markerColor = "#555555"
)

type svgCanvas struct {
	w, h   float64
	b      strings.Builder
	margin margin
}

type margin struct{ left, right, top, bottom float64 }

func newCanvas(w, h float64, m margin) *svgCanvas {
	c := &svgCanvas{w: w, h: h, margin: m}
	c.b.WriteString(fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d">`,
		int(w), int(h), int(w), int(h)))
	c.b.WriteString(`<rect width="100%" height="100%" fill="white"/>`)
	return c
}

func (c *svgCanvas) plotW() float64 { return c.w - c.margin.left - c.margin.right }
func (c *svgCanvas) plotH() float64 { return c.h - c.margin.top - c.margin.bottom }

func (c *svgCanvas) xMap(t, t0, t1 float64) float64 {
	return c.margin.left + (t-t0)/(t1-t0)*c.plotW()
}

func (c *svgCanvas) yMap(v, v0, v1 float64) float64 {
	return c.margin.top + (1-(v-v0)/(v1-v0))*c.plotH()
}

func (c *svgCanvas) title(text string) {
	c.b.WriteString(fmt.Sprintf(
		`<text x="%.1f" y="%.1f" font-family="sans-serif" font-size="16" font-weight="bold" fill="%s" text-anchor="middle">%s</text>`,
		c.w/2, c.margin.top-14, textColor, escape(text)))
}

func (c *svgCanvas) axes(x0, x1, y0, y1 float64, yTicks int, yFormat func(float64) string) {
	left, top := c.margin.left, c.margin.top
	right, bottom := left+c.plotW(), top+c.plotH()
	for i := 0; i <= yTicks; i++ {
		v := y0 + (y1-y0)*float64(i)/float64(yTicks)
		y := c.yMap(v, y0, y1)
		c.b.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="1"/>`,
			left, y, right, y, gridColor))
		c.b.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" font-family="sans-serif" font-size="10" fill="%s" text-anchor="end">%s</text>`,
			left-5, y+3, textColor, escape(yFormat(v))))
	}
	for t := math.Ceil(x0); t <= x1; t++ {
		x := c.xMap(t, x0, x1)
		c.b.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="1"/>`,
			x, top, x, bottom, gridColor))
		c.b.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" font-family="sans-serif" font-size="10" fill="%s" text-anchor="middle">%.0fs</text>`,
			x, bottom+14, textColor, t))
	}
	c.b.WriteString(fmt.Sprintf(
		`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="1.5"/>`,
		left, top, left, bottom, axisColor))
	c.b.WriteString(fmt.Sprintf(
		`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="1.5"/>`,
		left, bottom, right, bottom, axisColor))
}

func (c *svgCanvas) keyframeMarkers(times []float64, x0, x1, y0, y1 float64) {
	for _, t := range times {
		x := c.xMap(t, x0, x1)
		c.b.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="1" stroke-dasharray="4 3"/>`,
			x, c.margin.top, x, c.margin.top+c.plotH(), markerColor))
	}
}

func (c *svgCanvas) line(xs, ys []float64, x0, x1, y0, y1 float64, color string, width float64) {
	c.b.WriteString(`<path d="`)
	pen := false
	for i := range xs {
		if math.IsNaN(ys[i]) {
			pen = false
			continue
		}
		cmd := "L"
		if !pen {
			cmd = "M"
			pen = true
		}
		c.b.WriteString(fmt.Sprintf("%s%.2f,%.2f ", cmd,
			c.xMap(xs[i], x0, x1), c.yMap(ys[i], y0, y1)))
	}
	c.b.WriteString(fmt.Sprintf(`" fill="none" stroke="%s" stroke-width="%.1f"/>`, color, width))
}

func (c *svgCanvas) legend(items []legendItem) {
	x := c.margin.left + 8
	y := c.margin.top + 12
	for _, item := range items {
		c.b.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="2.5"/>`,
			x, y, x+22, y, item.Color))
		c.b.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" font-family="sans-serif" font-size="11" fill="%s">%s</text>`,
			x+27, y+4, textColor, escape(item.Label)))
		y += 16
	}
}

type legendItem struct {
	Color, Label string
}

func (c *svgCanvas) writeTo(w io.Writer) error {
	c.b.WriteString("</svg>")
	_, err := io.WriteString(w, c.b.String())
	return err
}

func escape(s string) string {
	return strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;").Replace(s)
}
