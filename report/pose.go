package report

import (
	"fmt"
	"io"

	"orientation-interp/interp"
	"orientation-interp/rotmath"
)

var axisColors = [3]string{"#d62728", "#2ca02c", "#1f77b4"}

// WritePoseStrip renders a row of RGB orientation tripods per method.
// Each tripod is an orthographic projection of the body axes onto the
// viewing plane; depth cues shorten the projected vectors automatically.
func WritePoseStrip(w io.Writer, all []Series, keyTimes []float64) error {
	const (
		width     = 960
		rowHeight = 110
		top       = 50
		left      = 150
		right     = 30
	)
	height := top + rowHeight*len(all) + 20
	c := newCanvas(width, float64(height), margin{left: left, right: right, top: float64(top), bottom: 20})
	c.title("Intermediate pose sequence (body axes X=red, Y=green, Z=blue)")

	step := 0.2
	slots := int((interp.ScenarioEnd-interp.ScenarioStart)/step) + 1
	usable := c.plotW() / float64(slots)

	for row, s := range all {
		centerY := float64(top) + float64(row)*float64(rowHeight) + float64(rowHeight)/2
		c.b.WriteString(fmt.Sprintf(
			`<text x="%.1f" y="%.1f" font-family="sans-serif" font-size="12" fill="%s" text-anchor="end">%s</text>`,
			float64(left-10), centerY+4, textColor, escape(s.Label)))

		for slot := 0; slot < slots; slot++ {
			t := interp.ScenarioStart + float64(slot)*step
			centerX := float64(left) + (float64(slot)+0.5)*usable
			idx := nearestSample(s.Samples, t)
			drawTripod(c, s.Samples[idx].Matrix, centerX, centerY, usable*0.32)
			if row == 0 {
				c.b.WriteString(fmt.Sprintf(
					`<text x="%.1f" y="%.1f" font-family="sans-serif" font-size="9" fill="%s" text-anchor="middle">%.1f</text>`,
					centerX, float64(top)+float64(len(all))*float64(rowHeight)+12, textColor, t))
			}
		}
		c.b.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s"/>`,
			float64(left), centerY+float64(rowHeight)/2-6,
			float64(width-right), centerY+float64(rowHeight)/2-6, gridColor))
	}
	return c.writeTo(w)
}

func nearestSample(samples []interp.Sample, t float64) int {
	idx := int((t - interp.ScenarioStart) / interp.ScenarioStep)
	if idx < 0 {
		return 0
	}
	if idx >= len(samples) {
		return len(samples) - 1
	}
	return idx
}

// drawTripod projects body axes onto the XY viewing plane (camera looks
// along +Z). A shaft drawn from the projected origin shows each axis.
func drawTripod(c *svgCanvas, m rotmath.Mat3, cx, cy, scale float64) {
	for axis := 0; axis < 3; axis++ {
		dx := m[0][axis] * scale
		dy := m[1][axis] * scale
		c.b.WriteString(fmt.Sprintf(
			`<line x1="%.1f" y1="%.1f" x2="%.1f" y2="%.1f" stroke="%s" stroke-width="2"/>`,
			cx, cy, cx+dx, cy+dy, axisColors[axis]))
		c.b.WriteString(fmt.Sprintf(
			`<circle cx="%.1f" cy="%.1f" r="1.6" fill="%s"/>`,
			cx+dx, cy+dy, axisColors[axis]))
	}
	c.b.WriteString(fmt.Sprintf(`<circle cx="%.1f" cy="%.1f" r="2.2" fill="%s"/>`, cx, cy, axisColor))
}
