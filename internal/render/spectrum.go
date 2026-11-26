package render

import (
	"thinfilm/internal/thinfilm"
)

// SpectrumConfig controls the reflectance-vs-wavelength chart.
type SpectrumConfig struct {
	Width, Height int
	FilmN         float64
	Thicknesses   []float64 // nm, one curve each
	Title         string
}

const (
	specMarginLeft   = 70
	specMarginRight  = 30
	specMarginTop    = 74
	specMarginBottom = 78
	specLambdaMin    = 380.0
	specLambdaMax    = 780.0
)

// curve colors cycle through high-contrast, color-vision-safe hues.
var curveColors = []RGB{
	{31, 119, 180},  // blue
	{255, 127, 14},  // orange
	{44, 160, 44},   // green
	{214, 39, 40},   // red
	{148, 103, 189}, // purple
	{140, 86, 75},   // brown
	{227, 119, 194}, // pink
}

// DrawSpectrum plots reflectance curves at normal incidence for the given
// film thicknesses, with a spectral-color band under the wavelength axis.
func DrawSpectrum(cfg SpectrumConfig) *Canvas {
	c := NewCanvas(cfg.Width, cfg.Height, RGB{250, 250, 248})
	x0, x1 := specMarginLeft, cfg.Width-specMarginRight
	y0, y1 := specMarginTop, cfg.Height-specMarginBottom
	axisCol := RGB{40, 40, 40}
	gridCol := RGB{220, 220, 215}

	// Horizontal grid + reflectance ticks (0..1).
	for r := 0; r <= 10; r++ {
		fraction := float64(r) / 10
		y := y1 - int(fraction*float64(y1-y0))
		c.Line(x0, y, x1, y, gridCol)
		c.Line(x0-5, y, x0, y, axisCol)
		label := formatPercent(fraction)
		c.Text(x0-10-TextWidth(label, 1), y-3, 1, label, axisCol)
	}

	// Wavelength ticks every 50 nm plus the spectral color band.
	bandTop := y1 + 22
	bandBottom := y1 + 38
	for lambda := specLambdaMin; lambda <= specLambdaMax; lambda += 10 {
		x := lambdaToX(lambda, x0, x1)
		col := spectralColor(lambda)
		c.Line(x, bandTop, x, bandBottom, col)
		if int(lambda)%50 == 0 {
			c.Line(x, y1, x, y1+5, axisCol)
			label := formatWavelength(int(lambda))
			c.Text(x-TextWidth(label, 1)/2, y1+8, 1, label, axisCol)
		}
	}

	// Frame.
	c.Line(x0, y0, x1, y0, axisCol)
	c.Line(x1, y0, x1, y1, axisCol)
	c.Line(x1, y1, x0, y1, axisCol)
	c.Line(x0, y1, x0, y0, axisCol)

	// One polyline per thickness.
	for i, d := range cfg.Thicknesses {
		stack := thinfilm.NewStack(cfg.FilmN, d)
		col := curveColors[i%len(curveColors)]
		var pts [][2]int
		for lambda := specLambdaMin; lambda <= specLambdaMax; lambda += 2 {
			refl := stack.Reflectance(lambda, 0)
			x := lambdaToX(lambda, x0, x1)
			y := y1 - int(refl*float64(y1-y0))
			pts = append(pts, [2]int{x, y})
		}
		c.Polyline(pts, col)
	}

	// Legend.
	legendY := 12
	legendX := x0
	for i, d := range cfg.Thicknesses {
		label := legendLabel(d)
		col := curveColors[i%len(curveColors)]
		c.FillRect(legendX, legendY, legendX+18, legendY+10, col)
		c.Text(legendX+22, legendY+1, 1, label, axisCol)
		legendX += 22 + TextWidth(label, 1) + 24
	}

	// Axis names.
	xName := "WAVELENGTH (NM)"
	c.Text(x0+(x1-x0-TextWidth(xName, 2))/2, bandBottom+10, 2, xName, axisCol)
	drawVertical(c, 14, y0+(y1-y0)/2-40, "REFLECTANCE", axisCol)
	if cfg.Title != "" {
		c.Text((cfg.Width-TextWidth(cfg.Title, 2))/2, 36, 2, cfg.Title, axisCol)
	}
	return c
}

func lambdaToX(lambda float64, x0, x1 int) int {
	fraction := (lambda - specLambdaMin) / (specLambdaMax - specLambdaMin)
	return x0 + int(fraction*float64(x1-x0))
}

func formatPercent(f float64) string {
	v := int(f*100 + 0.5)
	if v == 100 {
		return "100"
	}
	if v < 10 {
		return "  " + string([]byte{byte('0' + v)})
	}
	return string([]byte{byte('0' + v/10), byte('0' + v%10)})
}

func formatWavelength(lambda int) string {
	out := []byte{}
	if lambda == 0 {
		return "0"
	}
	for lambda > 0 {
		out = append([]byte{byte('0' + lambda%10)}, out...)
		lambda /= 10
	}
	return string(out)
}

func legendLabel(d float64) string {
	if d == 0 {
		return "D=0NM (BARE)"
	}
	return "D=" + formatWavelength(int(d)) + "NM"
}
