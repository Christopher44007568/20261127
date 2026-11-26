package render

import (
	"fmt"
	"html"
	"math"
	"strings"
)

const (
	colorInk    = "#0f172a"
	colorMuted  = "#64748b"
	colorGrid   = "#e2e8f0"
	colorPanel  = "#ffffff"
	colorBG     = "#f8fafc"
	colorIdle   = "#94a3b8"
	colorWalk   = "#16a34a"
	colorRun    = "#dc2626"
	colorSpeed  = "#7c3aed"
	colorNaive  = "#f97316"
	colorRaw    = "#f59e0b"
	colorLocked = "#2563eb"
	colorLeft   = "#2563eb"
	colorRight  = "#ea580c"
)

type svgBuilder struct {
	strings.Builder
}

func newSVG(width, height int) *svgBuilder {
	s := &svgBuilder{}
	fmt.Fprintf(s, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="0 0 %d %d" role="img">`, width, height, width, height)
	s.WriteString(`<style>text{font-family:"Noto Sans CJK SC","Microsoft YaHei",system-ui,sans-serif}.ns{vector-effect:non-scaling-stroke}</style>`)
	fmt.Fprintf(s, `<rect width="%d" height="%d" fill="%s"/>`, width, height, colorBG)
	return s
}

func (s *svgBuilder) end() string {
	s.WriteString(`</svg>`)
	return s.String()
}

func num(v float64) string {
	v = math.Round(v*100) / 100
	return fmt.Sprintf("%.2f", v)
}

func (s *svgBuilder) rect(x, y, w, h, rx float64, attrs string) {
	fmt.Fprintf(s, `<rect x="%s" y="%s" width="%s" height="%s" rx="%s" %s/>`, num(x), num(y), num(w), num(h), num(rx), attrs)
}

func (s *svgBuilder) line(x1, y1, x2, y2 float64, attrs string) {
	fmt.Fprintf(s, `<line x1="%s" y1="%s" x2="%s" y2="%s" %s/>`, num(x1), num(y1), num(x2), num(y2), attrs)
}

func (s *svgBuilder) circle(cx, cy, r float64, attrs string) {
	fmt.Fprintf(s, `<circle cx="%s" cy="%s" r="%s" %s/>`, num(cx), num(cy), num(r), attrs)
}

func (s *svgBuilder) text(x, y float64, value, attrs string) {
	fmt.Fprintf(s, `<text x="%s" y="%s" %s>%s</text>`, num(x), num(y), attrs, html.EscapeString(value))
}

func (s *svgBuilder) path(d, attrs string) {
	fmt.Fprintf(s, `<path d="%s" %s/>`, d, attrs)
}

func (s *svgBuilder) raw(value string) {
	s.WriteString(value)
}

func lineAttrs(color string, width float64, extra string) string {
	return fmt.Sprintf(`stroke="%s" stroke-width="%.1f" stroke-linecap="round" stroke-linejoin="round" fill="none" class="ns" vector-effect="non-scaling-stroke" %s`, color, width, extra)
}
