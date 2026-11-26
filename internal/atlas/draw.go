package atlas

import (
	"image"
	"image/color"
	"math"

	"glyphsdf/internal/render"
)

func (a *Atlas) DrawLine(dst *image.RGBA, text string, method render.Method, x, baseline, fontSize float64, ink, paper color.RGBA) float64 {
	scale := fontSize / a.Em
	pen := x
	for _, r := range text {
		g, err := a.Get(r)
		if err != nil {
			pen += fontSize * 0.55
			continue
		}
		cov := render.Coverage(g.Field, method, scale, 4)
		dx := int(math.Round(pen - g.Pad*scale))
		dy := int(math.Round(baseline - g.Baseline*scale))
		render.DrawCoverageTinted(dst, cov, dx, dy, ink, paper)
		pen += g.Advance * scale
	}
	return pen
}
