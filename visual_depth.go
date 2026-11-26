package main

import "math"

var depthPalette = []RGB{
	{0.04, 0.08, 0.28},
	{0.05, 0.38, 0.72},
	{0.10, 0.72, 0.72},
	{0.55, 0.86, 0.35},
	{0.98, 0.82, 0.20},
	{0.85, 0.20, 0.18},
}

func depthColor(d, minD, maxD float64) RGB {
	t := (d - minD) / (maxD - minD)
	return paletteStops(clampFloat(t, 0, 1), depthPalette)
}

func DepthRange(frames []*DepthFrame) (float64, float64) {
	minD, maxD := math.MaxFloat64, -math.MaxFloat64
	for _, f := range frames {
		for _, d := range f.Depth {
			if d <= 0 {
				continue
			}
			minD = math.Min(minD, d)
			maxD = math.Max(maxD, d)
		}
	}
	if minD > maxD {
		return 1, 4
	}
	pad := (maxD - minD) * 0.04
	return minD - pad, maxD + pad
}

func WriteDepthMosaicPNG(path string, scene Scene, frames []*DepthFrame) error {
	const scale = 2
	const pad = 14
	const labelH = 24
	cols := 4
	rows := (len(frames) + cols - 1) / cols
	tileW := frames[0].Width * scale
	tileH := frames[0].Height * scale
	width := pad + cols*(tileW+pad)
	height := pad + rows*(tileH+labelH+pad) + 34

	cv := NewCanvas(width, height, RGB{0.93, 0.94, 0.96})
	cv.DrawText(pad, 10, scene.ID+" / raw depth mosaic + holes", 2, RGB{0.05, 0.07, 0.10})

	minD, maxD := DepthRange(frames)
	for i, f := range frames {
		col := i % cols
		row := i / cols
		x0 := pad + col*(tileW+pad)
		y0 := 42 + pad + row*(tileH+labelH+pad)
		cv.FillRect(x0, y0, tileW, tileH, RGB{0.08, 0.09, 0.12})

		for y := 0; y < f.Height; y++ {
			for x := 0; x < f.Width; x++ {
				d := f.Depth[y*f.Width+x]
				pixel := RGB{0.055, 0.065, 0.095}
				if f.HoleMask[y*f.Width+x] {
					pixel = RGB{0.95, 0.05, 0.75}
				}
				if d > 0 {
					pixel = depthColor(d, minD, maxD)
				}
				cv.FillRect(x0+x*scale, y0+y*scale, scale, scale, pixel)
			}
		}
		cv.FrameRect(x0, y0, tileW, tileH, 2, RGB{0.12, 0.14, 0.18})
		label := f.Label +
			" az=" + formatFloat(f.True.Azimuth, 0) +
			" err=" + formatFloat(radToDeg(f.Fusion.PoseErrorRadians), 2) + "deg"
		cv.DrawText(x0, y0+tileH+6, label, 1, RGB{0.08, 0.09, 0.12})
	}

	barW := 260
	barX := width - barW - pad
	barY := 12
	for x := 0; x < barW; x++ {
		cv.FillRect(barX+x, barY, 1, 12, depthColor(minD+float64(x)/float64(barW)*(maxD-minD), minD, maxD))
	}
	cv.FrameRect(barX, barY, barW, 12, 1, RGB{0.1, 0.1, 0.1})
	minLabel := formatFloat(minD, 2) + "m"
	maxLabel := formatFloat(maxD, 2) + "m"
	cv.DrawText(barX, barY+15, minLabel, 1, RGB{0.05, 0.05, 0.05})
	cv.DrawText(barX+barW-TextWidth(maxLabel, 1), barY+15, maxLabel, 1, RGB{0.05, 0.05, 0.05})
	cv.FillRect(barX-78, barY+1, 10, 10, RGB{0.95, 0.05, 0.75})
	cv.DrawText(barX-64, barY, "hole", 1, RGB{0.05, 0.05, 0.05})

	return SaveCanvasPNG(path, cv)
}

func formatFloat(v float64, digits int) string {
	if digits == 0 {
		return stringInt(int(math.Round(v)))
	}
	scale := math.Pow10(digits)
	n := int(math.Round(v * scale))
	neg := n < 0
	if neg {
		n = -n
	}
	whole := n / int(scale)
	frac := n % int(scale)
	s := stringInt(whole) + "."
	for d := digits - 1; d >= 0; d-- {
		p := int(math.Pow10(d))
		s += string(rune('0' + frac/p%10))
	}
	if neg {
		s = "-" + s
	}
	return s
}

func stringInt(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}
