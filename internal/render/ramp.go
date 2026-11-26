package render

import (
	"thinfilm/internal/color"
	"thinfilm/internal/thinfilm"
)

// RampConfig controls the thickness/incidence-angle color ramp.
type RampConfig struct {
	Width, Height int
	FilmN         float64
	ThicknessMax  float64 // nm, right edge of the horizontal axis
	Title         string
}

// Layout margins for the ramp figure.
const (
	rampMarginLeft   = 70
	rampMarginRight  = 30
	rampMarginTop    = 46
	rampMarginBottom = 56
)

// DrawRamp renders a 2D field: horizontal axis = film thickness, vertical
// axis = incidence angle (0 degrees at the bottom, grazing at the top).
// The zero-thickness column is the colorless bare-metal reference and the
// colors vary smoothly across both axes.
func DrawRamp(cfg RampConfig) *Canvas {
	c := NewCanvas(cfg.Width, cfg.Height, RGB{250, 250, 248})

	plotX0, plotX1 := rampMarginLeft, cfg.Width-rampMarginRight
	plotY0, plotY1 := rampMarginTop, cfg.Height-rampMarginBottom

	for py := plotY0; py < plotY1; py++ {
		// Angle: top row = grazing (89 deg), bottom row = normal (0 deg).
		fracY := float64(py-plotY0) / float64(plotY1-plotY0-1)
		theta := 89.0 * fracY
		for px := plotX0; px < plotX1; px++ {
			fracX := float64(px-plotX0) / float64(plotX1-plotX0-1)
			d := fracX * cfg.ThicknessMax
			stack := thinfilm.NewStack(cfg.FilmN, d)
			spec := func(lambda float64) float64 {
				return stack.Reflectance(lambda, theta)
			}
			r, g, b := color.RGB(spec)
			c.Set(px, py, RGB{r, g, b})
		}
	}

	axisCol := RGB{40, 40, 40}
	gridCol := RGB{210, 210, 205}

	// Plot frame.
	c.Line(plotX0, plotY0, plotX1, plotY0, axisCol)
	c.Line(plotX1, plotY0, plotX1, plotY1, axisCol)
	c.Line(plotX1, plotY1, plotX0, plotY1, axisCol)
	c.Line(plotX0, plotY1, plotX0, plotY0, axisCol)

	// Thickness ticks every 100 nm.
	for d := 0.0; d <= cfg.ThicknessMax+1; d += 100 {
		x := plotX0 + int(d/cfg.ThicknessMax*float64(plotX1-plotX0))
		c.Line(x, plotY1, x, plotY1+5, axisCol)
		c.Line(x, plotY0, x, plotY0+5, gridCol)
		label := formatThickness(d)
		w := TextWidth(label, 1)
		c.Text(x-w/2, plotY1+10, 1, label, axisCol)
	}

	// Angle ticks every 15 degrees.
	for theta := 0.0; theta <= 90; theta += 15 {
		y := plotY1 - int(theta/89.0*float64(plotY1-plotY0))
		c.Line(plotX0-5, y, plotX0, y, axisCol)
		c.Line(plotX1-5, y, plotX1, y, gridCol)
		label := formatAngle(theta)
		c.Text(plotX0-12-TextWidth(label, 1), y-3, 1, label, axisCol)
	}

	// Axis names and title.
	xName := "FILM THICKNESS (NM)"
	c.Text(plotX0+(plotX1-plotX0-TextWidth(xName, 2))/2,
		plotY1+34, 2, xName, axisCol)
	yName := "INCIDENCE ANGLE (DEG)"
	drawVertical(c, plotX0-58, plotY0+(plotY1-plotY0)/2, yName, axisCol)
	if cfg.Title != "" {
		c.Text((cfg.Width-TextWidth(cfg.Title, 2))/2, 12, 2, cfg.Title, axisCol)
	}

	return c
}

func drawVertical(c *Canvas, x, y int, s string, col RGB) {
	for i := 0; i < len(s); i++ {
		c.Text(x, y+i*(glyphH+2), 1, string(s[i]), col)
	}
}

func formatThickness(d float64) string {
	return formatFixed(int(d), 4)
}

func formatAngle(a float64) string {
	return formatFixed(int(a), 2)
}

// formatFixed renders a non-negative integer padded to at least width digits.
func formatFixed(v, width int) string {
	digits := []byte{}
	if v == 0 {
		digits = append(digits, '0')
	}
	for v > 0 {
		digits = append([]byte{byte('0' + v%10)}, digits...)
		v /= 10
	}
	for len(digits) < width {
		digits = append([]byte{' '}, digits...)
	}
	return string(digits)
}
