package main

import (
	"image"
	"image/color"
	"math"
	"sort"
	"strconv"
)

func RenderSeedMap(r *Result) *image.RGBA {
	g := r.Grid
	c := newCanvas(g, renderScale)
	c.clear(blockedColor())
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			id := g.ID(x, y)
			if g.Blocked[id] {
				c.fillCell(x, y, blockedColor())
				continue
			}
			base := color.RGBA{R: 218, G: 224, B: 230, A: 255}
			c.fillCell(x, y, blendShade(base, terrainShade(g, x, y)))
		}
	}
	for i, seed := range r.Seeds {
		drawSeedMarker(c, seed, i)
		cx := c.mapX(float64(seed.X)+0.5) + int(1.5*float64(c.scale))
		cy := c.mapY(float64(seed.Y)+0.5) - 5
		drawLabelHalo(c, strconv.Itoa(i), cx, cy, 2)
	}
	return c.img
}

func RenderRegionMap(r *Result) *image.RGBA {
	g := r.Grid
	c := newCanvas(g, renderScale)
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			id := g.ID(x, y)
			if g.Blocked[id] {
				c.fillCell(x, y, blockedColor())
				continue
			}
			col := labelColor(r.Labels[id])
			shade := 0.76 + 0.24*terrainShade(g, x, y)
			c.fillCell(x, y, blendShade(col, shade))
		}
	}
	segments := ExtractBoundarySegments(g, r.Labels)
	drawSegments(c, segments, c.scale/2+1, color.RGBA{R: 10, G: 12, B: 18, A: 230})
	for i, seed := range r.Seeds {
		drawSeedMarker(c, seed, i)
	}
	return c.img
}

func RenderBoundaryMap(r *Result) *image.RGBA {
	g := r.Grid
	c := newCanvas(g, renderScale)
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			id := g.ID(x, y)
			if g.Blocked[id] {
				c.fillCell(x, y, color.RGBA{R: 4, G: 6, B: 10, A: 255})
				continue
			}
			shade := terrainShade(g, x, y)
			col := color.RGBA{
				R: uint8(clampUint8(34*shade + 42*r.Feature.Feature[id])),
				G: uint8(clampUint8(43*shade + 18*r.Feature.Feature[id])),
				B: uint8(clampUint8(58*shade + 58*r.Feature.Feature[id])),
				A: 255,
			}
			c.fillCell(x, y, col)
		}
	}
	segments := ExtractBoundarySegments(g, r.Labels)
	drawSegments(c, segments, c.scale+2, color.RGBA{R: 0, G: 0, B: 0, A: 255})
	drawSegments(c, segments, maxInt(2, c.scale/2), color.RGBA{R: 236, G: 246, B: 255, A: 255})
	for i, seed := range r.Seeds {
		drawSeedMarker(c, seed, i)
	}
	return c.img
}

func RenderHeatmap(r *Result) *image.RGBA {
	g := r.Grid
	c := newCanvas(g, renderScale)
	maxDistance := finiteMax(r.Distance)
	if maxDistance <= 0 {
		maxDistance = 1
	}
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			id := g.ID(x, y)
			if g.Blocked[id] {
				c.fillCell(x, y, color.RGBA{R: 5, G: 7, B: 10, A: 255})
				continue
			}
			col := heatColor(r.Distance[id] / maxDistance)
			shade := 0.90 + 0.10*terrainShade(g, x, y)
			c.fillCell(x, y, blendShade(col, shade))
		}
	}
	segments := ExtractBoundarySegments(g, r.Labels)
	drawSegments(c, segments, maxInt(2, c.scale/2+1), color.RGBA{R: 0, G: 0, B: 0, A: 210})
	drawSeedMarkersSmall(c, r.Seeds)
	drawHeatLegend(c, maxDistance)
	return c.img
}

func drawSeedMarkersSmall(c *canvas, seeds []Seed) {
	for i, seed := range seeds {
		s := c.scale
		cx := c.mapX(float64(seed.X) + .5)
		cy := c.mapY(float64(seed.Y) + .5)
		drawCircle(c.img, cx, cy, s/2+2, color.RGBA{R: 0, G: 0, B: 0, A: 230})
		drawCircle(c.img, cx, cy, s/2, color.RGBA{R: 255, G: 255, B: 255, A: 255})
		_ = i
	}
}

func drawSegments(c *canvas, segments []BoundarySegment, width int, col color.RGBA) {
	if width < 1 {
		width = 1
	}
	for _, seg := range segments {
		if seg.X1 == seg.X2 {
			cx := c.mapX(seg.X1)
			y1 := c.mapY(seg.Y1)
			y2 := c.mapY(seg.Y2)
			fillRect(c.img, image.Rect(cx-width/2, y1, cx-width/2+width, y2), col)
		} else {
			cy := c.mapY(seg.Y1)
			x1 := c.mapX(seg.X1)
			x2 := c.mapX(seg.X2)
			fillRect(c.img, image.Rect(x1, cy-width/2, x2, cy-width/2+width), col)
		}
	}
}

func drawHeatLegend(c *canvas, maxDistance float64) {
	img := c.img
	x0, y0 := 26, 26
	width, height := 180, 14
	for x := 0; x < width; x++ {
		col := heatColor(float64(x) / float64(width-1))
		fillRect(img, image.Rect(x0+x, y0, x0+x+1, y0+height), col)
	}
	border := color.RGBA{R: 255, G: 255, B: 255, A: 230}
	for x := x0 - 1; x <= x0+width; x++ {
		img.SetRGBA(x, y0-1, border)
		img.SetRGBA(x, y0+height, border)
	}
	for y := y0 - 1; y <= y0+height; y++ {
		img.SetRGBA(x0-1, y, border)
		img.SetRGBA(x0+width, y, border)
	}
	drawLabelHalo(c, "0", x0+2, y0+3, 2)
	label := strconv.Itoa(int(math.Ceil(maxDistance)))
	textWidth := len(label) * 8
	drawLabelHalo(c, label, x0+width-textWidth-2, y0+3, 2)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// quantile is currently used by diagnostics and tests.
func quantile(values []float64, q float64) float64 {
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	if len(sorted) == 0 {
		return 0
	}
	idx := int(q * float64(len(sorted)-1))
	return sorted[idx]
}
