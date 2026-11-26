package render

import (
	"fmt"
	"image/color"
	"math"

	"meshdiagnostics/internal/diag"
	"meshdiagnostics/internal/remesh"
)

type chartFrame struct {
	x, y, w, h int
}

// RenderHistogram writes a two-panel diagnostic chart: edge-length
// distributions for triangles/quads and the quad corner-angle distribution.
func RenderHistogram(path string, triEdges, quadEdges []float64, a *diag.Analysis, rs remesh.Stats, collapseDeg float64) error {
	c := NewCanvas(1400, 820, color.RGBA{246, 244, 237, 255})
	backgroundGradient(c, color.RGBA{252, 250, 244, 255}, color.RGBA{230, 231, 224, 255})
	ink := color.RGBA{28, 44, 48, 255}
	muted := color.RGBA{86, 102, 103, 255}
	quadColor := color.RGBA{37, 130, 166, 255}
	triColor := color.RGBA{235, 139, 50, 255}
	red := color.RGBA{219, 70, 53, 255}

	c.DrawText(48, 28, "MESH DIAGNOSTICS - EDGE LENGTH AND QUAD ANGLE", 3, ink)
	c.DrawText(50, 68, fmt.Sprintf("TRI %d   QUAD %d   FEATURES %d   FEATURE LENGTH KEPT %.2f%%",
		len(triEdges), len(quadEdges), rs.QuadFeatures, rs.FeatureLengthRatio*100), 2, muted)

	left := chartFrame{x: 62, y: 140, w: 610, h: 500}
	right := chartFrame{x: 750, y: 140, w: 590, h: 500}
	drawChartFrame(c, left, "EDGE LENGTH DENSITY", "EDGES PER UNIT LENGTH", ink, muted)
	drawChartFrame(c, right, "QUAD CORNER ANGLE DENSITY", "CORNERS PER DEGREE", ink, muted)

	edgeLo, edgeHi := combinedRange(triEdges, quadEdges)
	edgeHistTri := diag.Histogram(triEdges, 30)
	edgeHistQuad := diag.Histogram(quadEdges, 30)
	edgeHistTri.Lo, edgeHistQuad.Lo = edgeLo, edgeLo
	edgeHistTri.Hi, edgeHistQuad.Hi = edgeHi, edgeHi
	edgeHistTri.Width, edgeHistQuad.Width = (edgeHi-edgeLo)/30, (edgeHi-edgeLo)/30
	for i := range edgeHistTri.Counts {
		edgeHistTri.Counts[i] = 0
	}
	for _, v := range triEdges {
		idx := int((v - edgeLo) / ((edgeHi - edgeLo) / 30))
		if idx < 0 {
			idx = 0
		}
		if idx >= 30 {
			idx = 29
		}
		edgeHistTri.Counts[idx]++
	}
	drawEdgePlot(c, left, edgeLo, edgeHi, triEdges, quadEdges, edgeHistTri, edgeHistQuad,
		triColor, quadColor, ink, muted)

	angleLo, angleHi := 0.0, 180.0
	angleHist := diag.Histogram(a.CornerAngles, 36)
	drawAnglePlot(c, right, angleLo, angleHi, a, angleHist, collapseDeg, quadColor, red, ink, muted)

	drawEdgeStats(c, left, a.EdgeStats, triEdges, quadColor, triColor, ink, muted)
	drawAngleStats(c, right, a, collapseDeg, red, ink, muted)
	if err := SavePNG(path, c.Downsample(1)); err != nil {
		return err
	}
	return nil
}

func combinedRange(a, b []float64) (float64, float64) {
	if len(a) == 0 && len(b) == 0 {
		return 0, 1
	}
	lo, hi := math.Inf(1), math.Inf(-1)
	for _, v := range a {
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	for _, v := range b {
		lo, hi = math.Min(lo, v), math.Max(hi, v)
	}
	if hi-lo < 1e-12 {
		hi = lo + 1
	}
	return lo, hi
}

func drawChartFrame(c *Canvas, f chartFrame, title, subtitle string, ink, muted color.RGBA) {
	c.FillRect(f.x, f.y, f.w, f.h, color.RGBA{255, 255, 255, 210})
	c.StrokeRect(f.x, f.y, f.w, f.h, 1, color.RGBA{176, 184, 181, 255})
	c.DrawText(f.x+18, f.y+14, title, 3, ink)
	c.DrawText(f.x+18, f.y+42, subtitle, 2, muted)
}

func drawEdgePlot(c *Canvas, f chartFrame, lo, hi float64, tri, quad []float64,
	triBins, quadBins diag.Bins, triColor, quadColor, ink, muted color.RGBA) {
	plot := chartFrame{x: f.x + 58, y: f.y + 72, w: f.w - 90, h: f.h - 136}
	drawGrid(c, plot, 5, ink, muted)
	maxDensity := math.Max(maxDensity(tri, triBins), maxDensity(quad, quadBins))
	if maxDensity < 1e-14 {
		maxDensity = 1
	}
	bw := float64(plot.w) / float64(len(quadBins.Counts))
	for i, count := range quadBins.Counts {
		d := density(count, len(quad), quadBins.Width)
		height := int(float64(plot.h) * d / maxDensity)
		x := plot.x + int(float64(i)*bw)
		c.FillRect(x+1, plot.y+plot.h-height, int(bw)-1, height, quadColor)
	}
	// Triangle series as a stepped outline for an easy visual comparison.
	lastX := plot.x
	lastY := plot.y + plot.h
	for i, count := range triBins.Counts {
		d := density(count, len(tri), triBins.Width)
		x := plot.x + int(float64(i)*bw)
		y := plot.y + plot.h - int(float64(plot.h)*d/maxDensity)
		if i > 0 {
			c.DrawLine(float64(lastX), float64(lastY), float64(lastX), float64(y), 2, triColor)
			c.DrawLine(float64(lastX), float64(y), float64(x), float64(y), 2, triColor)
		}
		lastX, lastY = x, y
	}
	drawAxisLabels(c, plot, lo, hi, "LENGTH", ink, muted)
	c.FillRect(f.x+330, f.y+58, 18, 10, quadColor)
	c.DrawText(f.x+354, f.y+56, "QUAD", 2, ink)
	c.FillRect(f.x+418, f.y+60, 22, 4, triColor)
	c.DrawText(f.x+448, f.y+56, "TRIANGLE", 2, ink)
}

func maxDensity(values []float64, bins diag.Bins) float64 {
	if len(values) == 0 || bins.Width <= 0 {
		return 0
	}
	best := 0.0
	for _, count := range bins.Counts {
		best = math.Max(best, density(count, len(values), bins.Width))
	}
	return best
}

func density(count, total int, width float64) float64 {
	if total == 0 || width <= 0 {
		return 0
	}
	return float64(count) / (float64(total) * width)
}

func drawAnglePlot(c *Canvas, f chartFrame, lo, hi float64, a *diag.Analysis, bins diag.Bins,
	collapseDeg float64, bandColor, red, ink, muted color.RGBA) {
	plot := chartFrame{x: f.x + 58, y: f.y + 72, w: f.w - 90, h: f.h - 136}
	drawGrid(c, plot, 5, ink, muted)
	maxDensity := maxDensity(a.CornerAngles, bins)
	if maxDensity < 1e-14 {
		maxDensity = 1
	}
	bw := float64(plot.w) / float64(len(bins.Counts))
	for i, count := range bins.Counts {
		d := density(count, len(a.CornerAngles), bins.Width)
		height := int(float64(plot.h) * d / maxDensity)
		center := bins.Lo + (float64(i)+0.5)*bins.Width
		col := bandColor
		if center < collapseDeg {
			col = red
		}
		c.FillRect(plot.x+int(float64(i)*bw)+1, plot.y+plot.h-height, int(bw)-1, height, col)
	}
	x := plot.x + int((collapseDeg-lo)/(hi-lo)*float64(plot.w))
	c.DrawLine(float64(x), float64(plot.y), float64(x), float64(plot.y+plot.h), 3, red)
	c.DrawText(x+5, plot.y+4, "COLLAPSE", 2, red)
	drawAxisLabels(c, plot, lo, hi, "DEG", ink, muted)
	c.FillRect(f.x+250, f.y+58, 18, 10, bandColor)
	c.DrawText(f.x+274, f.y+56, "HEALTHY", 2, ink)
	c.FillRect(f.x+370, f.y+58, 18, 10, red)
	c.DrawText(f.x+394, f.y+56, "COLLAPSE ZONE", 2, ink)
}

func drawGrid(c *Canvas, f chartFrame, rows int, ink, muted color.RGBA) {
	for i := 0; i <= rows; i++ {
		y := f.y + int(float64(i)*float64(f.h)/float64(rows))
		col := color.RGBA{212, 217, 214, 255}
		c.DrawLine(float64(f.x), float64(y), float64(f.x+f.w), float64(y), 1, col)
	}
	c.DrawLine(float64(f.x), float64(f.y), float64(f.x), float64(f.y+f.h), 2, ink)
	c.DrawLine(float64(f.x), float64(f.y+f.h), float64(f.x+f.w), float64(f.y+f.h), 2, ink)
}

func drawAxisLabels(c *Canvas, f chartFrame, lo, hi float64, unit string, ink, muted color.RGBA) {
	for i := 0; i <= 5; i++ {
		v := lo + (hi-lo)*float64(i)/5
		x := f.x + int(float64(i)*float64(f.w)/5)
		c.DrawText(x-8, f.y+f.h+8, fmt.Sprintf("%.3f", v), 1, muted)
	}
	c.DrawText(f.x+f.w-TextWidth(unit, 1)-2, f.y+f.h+28, unit, 1, muted)
	c.DrawText(f.x, f.y-22, fmt.Sprintf("%.3f", lo), 1, muted)
	_ = ink
}

func drawEdgeStats(c *Canvas, f chartFrame, quad diag.Summary, triVals []float64,
	quadColor, triColor, ink, muted color.RGBA) {
	tri := diag.Summarize(triVals)
	y := f.y + f.h + 18
	c.DrawText(f.x, y, fmt.Sprintf("QUAD LEN MEAN %.4f  P50 %.4f  CV %.3f",
		quad.Mean, quad.Median, quad.CV), 2, quadColor)
	c.DrawText(f.x, y+18, fmt.Sprintf("QUAD LEN P5 %.4f  P95 %.4f  MAX %.4f",
		quad.P5, quad.P95, quad.Max), 2, quadColor)
	c.DrawText(f.x, y+42, fmt.Sprintf("TRI LEN  MEAN %.4f  P50 %.4f  CV %.3f",
		tri.Mean, tri.Median, tri.CV), 2, triColor)
	c.DrawText(f.x, y+60, fmt.Sprintf("TRI LEN  P5 %.4f  P95 %.4f  MAX %.4f",
		tri.P5, tri.P95, tri.Max), 2, triColor)
	_ = ink
	_ = muted
}

func drawAngleStats(c *Canvas, f chartFrame, a *diag.Analysis, collapseDeg float64,
	red, ink, muted color.RGBA) {
	y := f.y + f.h + 18
	c.DrawText(f.x, y, fmt.Sprintf("CORNER MIN %.2f  P50 %.2f  MAX %.2f",
		a.AngleStats.Min, a.AngleStats.Median, a.AngleStats.Max), 2, ink)
	c.DrawText(f.x, y+18, fmt.Sprintf("CORNER P5 %.2f  P95 %.2f", a.AngleStats.P5, a.AngleStats.P95), 2, ink)
	c.DrawText(f.x, y+42, fmt.Sprintf("COLLAPSED %d / %d  (%.2f%%)",
		len(a.Collapsed), a.FaceCount, float64(len(a.Collapsed))*100/float64(a.FaceCount)), 2, red)
	c.DrawText(f.x, y+60, fmt.Sprintf("COLLAPSE THRESHOLD %.1f DEG", collapseDeg), 2, red)
	_ = muted
}

func QuadPaletteBand(i int) color.RGBA {
	if i < 0 || i >= len(curvaturePalette) {
		return curvaturePalette[0]
	}
	return curvaturePalette[i]
}
