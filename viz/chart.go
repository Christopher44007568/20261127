package viz

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

// Series is one curve in LineChart.
type Series struct {
	Name   string
	Color  color.NRGBA
	Pts    [][2]float64
	Step   bool // draw post-step staircase plus markers
	Square bool // square markers instead of circles
	Width  int
}

// LineChart renders a labeled axes chart for x,y in the given ranges.
func LineChart(title, xLabel, yLabel string, xMin, xMax, yMin, yMax float64, series []Series) *image.NRGBA {
	const W, H = 860, 460
	img := image.NewNRGBA(image.Rect(0, 0, W, H))
	Fill(img, color.NRGBA{R: 255, G: 255, B: 255, A: 255})

	ml, mr, mt, mb := 78, 30, 64, 64
	px0, py0 := ml, mt
	px1, py1 := W-mr, H-mb
	pw, ph := px1-px0, py1-py0

	X := func(x float64) int { return px0 + int(math.Round((x-xMin)/(xMax-xMin)*float64(pw))) }
	Y := func(y float64) int { return py1 - int(math.Round((y-yMin)/(yMax-yMin)*float64(ph))) }

	DrawTextFit(img, 20, 14, 3, W-40, title, color.NRGBA{R: 18, G: 18, B: 22, A: 255})
	if yLabel != "" {
		DrawText(img, ml, 42, 2, yLabel, color.NRGBA{R: 50, G: 52, B: 60, A: 255})
	}

	// Horizontal gridlines + y tick labels at quarters.
	for i := 0; i <= 4; i++ {
		v := yMin + (yMax-yMin)*float64(i)/4
		y := Y(v)
		drawHLine(img, px0, px1, y, color.NRGBA{R: 225, G: 228, B: 235, A: 255})
		s := fmt.Sprintf("%.2f", v)
		DrawText(img, px0-12-TextWidth(s, 2), y-6, 2, s, color.NRGBA{R: 70, G: 74, B: 84, A: 255})
	}
	// Vertical gridlines + x ticks.
	xTicks := []float64{-1, -0.5, 0, 0.5, 1}
	for _, v := range xTicks {
		if v < xMin || v > xMax {
			continue
		}
		x := X(v)
		drawVLine(img, x, py0, py1, color.NRGBA{R: 232, G: 234, B: 240, A: 255})
		s := fmt.Sprintf("%.1f", v)
		DrawText(img, x-TextWidth(s, 2)/2, py1+10, 2, s, color.NRGBA{R: 70, G: 74, B: 84, A: 255})
	}

	// Axes.
	drawHLine(img, px0, px1, py1, color.NRGBA{R: 40, G: 42, B: 50, A: 255})
	drawVLine(img, px0, py0, py1, color.NRGBA{R: 40, G: 42, B: 50, A: 255})
	DrawText(img, px0+pw/2-TextWidth(xLabel, 2)/2, H-26, 2, xLabel, color.NRGBA{R: 50, G: 52, B: 60, A: 255})

	// Draw series.
	for _, s := range series {
		width := s.Width
		if width < 1 {
			width = 2
		}
		for i := 1; i < len(s.Pts); i++ {
			a := s.Pts[i-1]
			b := s.Pts[i]
			if s.Step {
				// Post-step: horizontal at previous value, then vertical.
				drawThickLine(img, X(a[0]), Y(a[1]), X(b[0]), Y(a[1]), width, s.Color)
				drawThickLine(img, X(b[0]), Y(a[1]), X(b[0]), Y(b[1]), width, s.Color)
			} else {
				drawThickLine(img, X(a[0]), Y(a[1]), X(b[0]), Y(b[1]), width, s.Color)
			}
		}
		if s.Step {
			for _, p := range s.Pts {
				mx, my := X(p[0]), Y(p[1])
				if s.Square {
					fillRect(img, mx-4, my-4, 9, 9, s.Color)
				} else {
					fillCircle(img, mx, my, 4, s.Color)
				}
			}
		}
	}

	// Legend.
	lx, ly := px1-210, py0+12
	fillRect(img, lx-12, ly-8, 210, 26*len(series)+14, color.NRGBA{R: 255, G: 255, B: 255, A: 235})
	for i, s := range series {
		y := ly + i*26
		drawThickLine(img, lx, y+8, lx+34, y+8, 3, s.Color)
		DrawText(img, lx+44, y+2, 2, s.Name, color.NRGBA{R: 30, G: 32, B: 40, A: 255})
	}

	return img
}

func drawThickLine(img *image.NRGBA, x0, y0, x1, y1, thick int, c color.NRGBA) {
	dx := absInt(x1 - x0)
	dy := absInt(y1 - y0)
	steps := dx + dy
	if steps == 0 {
		fillRect(img, x0-thick/2, y0-thick/2, thick, thick, c)
		return
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := x0 + int(math.Round(float64(x1-x0)*t))
		y := y0 + int(math.Round(float64(y1-y0)*t))
		r := thick / 2
		fillRect(img, x-r, y-r, thick, thick, c)
	}
}

func fillCircle(img *image.NRGBA, cx, cy, r int, c color.NRGBA) {
	for y := -r; y <= r; y++ {
		for x := -r; x <= r; x++ {
			if x*x+y*y <= r*r {
				img.SetNRGBA(cx+x, cy+y, c)
			}
		}
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
