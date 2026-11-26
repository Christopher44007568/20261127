package viz

import (
	"fmt"
	"image"
	"image/color"

	"coverlab/cover"
	"coverlab/geom"
)

const panelPad = 16

func cropDims(scene cover.Scene) (x0, y0, x1, y1, cw, ch int) {
	c := scene.Crop
	x0, y0, x1, y1 = c[0], c[1], c[2], c[3]
	return x0, y0, x1, y1, x1 - x0, y1 - y0
}

func panelCanvas(w, h int, title string) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	Fill(img, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	DrawTextFit(img, panelPad, 14, 3, w-2*panelPad, title, color.NRGBA{R: 20, G: 20, B: 24, A: 255})
	return img
}

// CoveragePanel draws the crop as a zoomed grayscale coverage matrix with
// per-pixel numeric coverage annotations.
func CoveragePanel(scene cover.Scene, cov []float64, title string) *image.NRGBA {
	const cell = 56
	x0, y0, _, _, cw, ch := cropDims(scene)
	gridX, gridY := panelPad, 52
	w := gridX*2 + cw*cell
	h := gridY + ch*cell + panelPad
	img := panelCanvas(w, h, title)

	for jy := 0; jy < ch; jy++ {
		for jx := 0; jx < cw; jx++ {
			sx, sy := x0+jx, y0+jy
			v := cov[sy*scene.W+sx]
			px := gridX + jx*cell
			py := gridY + jy*cell
			fillRect(img, px, py, cell, cell, GrayCoverage(v))
			label := fmt.Sprintf("%.2f", v)
			tw := TextWidth(label, 2)
			tc := color.NRGBA{R: 245, G: 245, B: 245, A: 255}
			if v > 0.55 {
				tc = color.NRGBA{R: 15, G: 15, B: 18, A: 255}
			}
			DrawText(img, px+(cell-tw)/2, py+cell/2-6, 2, label, tc)
		}
	}
	DrawCellGrid(img, gridX, gridY, cw, ch, cell, color.NRGBA{R: 0, G: 0, B: 0, A: 70})
	return img
}

// CloseupPanel draws the zoomed color render of the crop with a pixel grid.
func CloseupPanel(scene cover.Scene, rgb []cover.RGB, title string) *image.NRGBA {
	const cell = 46
	x0, y0, _, _, cw, ch := cropDims(scene)
	gridX, gridY := panelPad, 52
	w := gridX*2 + cw*cell
	h := gridY + ch*cell + panelPad
	img := panelCanvas(w, h, title)

	for jy := 0; jy < ch; jy++ {
		for jx := 0; jx < cw; jx++ {
			sx, sy := x0+jx, y0+jy
			c := rgb[sy*scene.W+sx]
			px := gridX + jx*cell
			py := gridY + jy*cell
			fillRect(img, px, py, cell, cell, color.NRGBA{R: clamp8(c.R), G: clamp8(c.G), B: clamp8(c.B), A: 255})
		}
	}
	DrawCellGrid(img, gridX, gridY, cw, ch, cell, color.NRGBA{R: 0, G: 0, B: 0, A: 80})
	return img
}

// HeatmapPanel draws |got-ref| over the crop using Viridis, plus a colorbar
// and a stats line. vmax > 0 fixes the scale; otherwise the panel auto-scales.
func HeatmapPanel(scene cover.Scene, got, ref []float64, vmax float64, title, stats string) *image.NRGBA {
	const cell = 40
	x0, y0, _, _, cw, ch := cropDims(scene)
	gridX, gridY := panelPad, 58
	gridW, gridH := cw*cell, ch*cell
	barX := gridX + gridW + 22
	barW := 20
	w := barX + barW + 92
	h := gridY + gridH + panelPad + 26
	img := panelCanvas(w, h, title)
	if stats != "" {
		DrawText(img, panelPad, 36, 2, stats, color.NRGBA{R: 70, G: 70, B: 80, A: 255})
	}

	if vmax <= 0 {
		for i := range ref {
			if e := absf(got[i] - ref[i]); e > vmax {
				vmax = e
			}
		}
	}
	if vmax < 1e-9 {
		vmax = 1e-9
	}

	for jy := 0; jy < ch; jy++ {
		for jx := 0; jx < cw; jx++ {
			sx, sy := x0+jx, y0+jy
			e := absf(got[sy*scene.W+sx] - ref[sy*scene.W+sx])
			px := gridX + jx*cell
			py := gridY + jy*cell
			fillRect(img, px, py, cell, cell, Viridis(e/vmax))
		}
	}
	DrawCellGrid(img, gridX, gridY, cw, ch, cell, color.NRGBA{R: 255, G: 255, B: 255, A: 45})

	// Colorbar: top = vmax, bottom = 0.
	barH := gridH
	for yy := 0; yy < barH; yy++ {
		t := 1 - float64(yy)/float64(barH)
		fillRect(img, barX, gridY+yy, barW, 1, Viridis(t))
	}
	DrawRectOutline(img, image.Rect(barX, gridY, barX+barW, gridY+barH), 1, color.NRGBA{R: 40, G: 40, B: 48, A: 255})
	labels := []struct {
		y float64
		s string
	}{
		{0, fmt.Sprintf("%.3f", vmax)},
		{0.5, fmt.Sprintf("%.3f", vmax/2)},
		{1, "0"},
	}
	for _, lb := range labels {
		ly := gridY + int(lb.y*float64(barH)) - 6
		if lb.y == 1 {
			ly = gridY + barH - 12
		}
		DrawText(img, barX+barW+8, ly, 2, lb.s, color.NRGBA{R: 30, G: 30, B: 36, A: 255})
	}
	return img
}

func fillRect(img *image.NRGBA, x, y, w, h int, c color.NRGBA) {
	r := image.Rect(x, y, x+w, y+h).Intersect(img.Bounds())
	for yy := r.Min.Y; yy < r.Max.Y; yy++ {
		off := yy*img.Stride + r.Min.X*4
		for xx := r.Min.X; xx < r.Max.X; xx++ {
			i := off + (xx-r.Min.X)*4
			img.Pix[i+0] = c.R
			img.Pix[i+1] = c.G
			img.Pix[i+2] = c.B
			img.Pix[i+3] = 255
		}
		off += img.Stride
	}
}

func absf(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

// StainPanel heat-maps cross-contamination on pixels outside the source
// contour's own footprint. Pixels with ownCov > 0 are drawn as dark slate
// and the contour outline is overlaid for localization.
func StainPanel(scene cover.Scene, stain01, ownCov []float64, outline []geom.Vec, vmax float64, title, stats string) *image.NRGBA {
	const cell = 40
	x0, y0, _, _, cw, ch := cropDims(scene)
	gridX, gridY := panelPad, 58
	gridW, gridH := cw*cell, ch*cell
	barX := gridX + gridW + 22
	barW := 20
	w := barX + barW + 92
	h := gridY + gridH + panelPad + 26
	img := panelCanvas(w, h, title)
	if stats != "" {
		DrawText(img, panelPad, 36, 2, stats, color.NRGBA{R: 70, G: 70, B: 80, A: 255})
	}
	if vmax < 1e-9 {
		vmax = 1e-9
	}

	own := color.NRGBA{R: 48, G: 50, B: 62, A: 255}
	for jy := 0; jy < ch; jy++ {
		for jx := 0; jx < cw; jx++ {
			sx, sy := x0+jx, y0+jy
			idx := sy*scene.W + sx
			px := gridX + jx*cell
			py := gridY + jy*cell
			if ownCov[idx] > 1e-12 {
				fillRect(img, px, py, cell, cell, own)
			} else {
				fillRect(img, px, py, cell, cell, Viridis(stain01[idx]/vmax))
			}
		}
	}
	DrawCellGrid(img, gridX, gridY, cw, ch, cell, color.NRGBA{R: 255, G: 255, B: 255, A: 45})

	// Contour outline.
	if len(outline) >= 2 {
		toPX := func(p geom.Vec) (int, int) {
			return gridX + int((p.X-float64(x0))*cell), gridY + int((p.Y-float64(y0))*cell)
		}
		line := color.NRGBA{R: 255, G: 255, B: 255, A: 235}
		for i := 0; i < len(outline); i++ {
			a := outline[i]
			b := outline[(i+1)%len(outline)]
			ax, ay := toPX(a)
			bx, by := toPX(b)
			drawThickLine(img, ax, ay, bx, by, 2, line)
		}
	}

	barH := gridH
	for yy := 0; yy < barH; yy++ {
		t := 1 - float64(yy)/float64(barH)
		fillRect(img, barX, gridY+yy, barW, 1, Viridis(t))
	}
	DrawRectOutline(img, image.Rect(barX, gridY, barX+barW, gridY+barH), 1,
		color.NRGBA{R: 40, G: 40, B: 48, A: 255})
	for _, lb := range []struct {
		frac float64
		s    string
	}{
		{0, fmt.Sprintf("%.3f", vmax)},
		{0.5, fmt.Sprintf("%.3f", vmax/2)},
		{1, "0"},
	} {
		ly := gridY + int(lb.frac*float64(barH)) - 6
		if lb.frac == 1 {
			ly = gridY + barH - 12
		}
		DrawText(img, barX+barW+8, ly, 2, lb.s, color.NRGBA{R: 30, G: 30, B: 36, A: 255})
	}
	return img
}
