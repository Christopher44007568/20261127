package main

import (
	"image"
	"image/color"
	"math"
)

// HeatMap is the tile-period repetitiveness field.
type HeatMap struct {
	W, H      int
	PeriodX   int
	PeriodY   int
	Radius    int
	Tau       float64
	Delta     []float64 // window-averaged period residual (0..~441 distance units)
	Score     []float64 // exp(-Delta/Tau), 1 = perfectly repeating
	MeanDelta float64
	MeanScore float64
}

// RepetitionHeat measures how identical each region is to the region one
// source-tile period away. Exact tiling gives Delta ~ 0 everywhere and
// Score ~ 1 (hot). A box window makes the result a local patch metric
// rather than a grainy per-pixel one.
func RepetitionHeat(img *image.RGBA, periodX, periodY, radius int, tau float64) *HeatMap {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	px, py := periodX, periodY

	res := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c0 := img.RGBAAt(x, y)
			r0 := rgba8ToRGB(c0.R, c0.G, c0.B)

			x1 := (x + px) % w
			y1s := y
			d1 := distAt(img, r0, x1, y1s)

			x2 := x
			y2 := (y + py) % h
			d2 := distAt(img, r0, x2, y2)

			x3 := (x + px) % w
			y3 := (y + py) % h
			d3 := distAt(img, r0, x3, y3)

			res[y*w+x] = (d1 + d2 + d3) / 3
		}
	}

	avg := boxMean(res, w, h, radius)
	hm := &HeatMap{W: w, H: h, PeriodX: px, PeriodY: py, Radius: radius, Tau: tau, Delta: avg}
	hm.Score = make([]float64, w*h)
	var sd, ss float64
	for i, d := range avg {
		s := math.Exp(-d / tau)
		hm.Score[i] = s
		sd += d
		ss += s
	}
	n := float64(w * h)
	hm.MeanDelta = sd / n
	hm.MeanScore = ss / n
	return hm
}

func distAt(img *image.RGBA, from RGB, x, y int) float64 {
	c := img.RGBAAt(x, y)
	to := rgba8ToRGB(c.R, c.G, c.B)
	return colorDist255(from, to)
}

// boxMean returns the mean of src over a (2r+1)^2 window using an
// integral image; borders clamp to the available overlap.
func boxMean(src []float64, w, h, r int) []float64 {
	stride := w + 1
	integral := make([]float64, (w+1)*(h+1))
	for y := 0; y < h; y++ {
		var rowSum float64
		for x := 0; x < w; x++ {
			rowSum += src[y*w+x]
			integral[(y+1)*stride+(x+1)] = integral[y*stride+(x+1)] + rowSum
		}
	}
	out := make([]float64, w*h)
	for y := 0; y < h; y++ {
		y0 := maxInt(0, y-r)
		y1 := minInt(h-1, y+r)
		for x := 0; x < w; x++ {
			x0 := maxInt(0, x-r)
			x1 := minInt(w-1, x+r)
			a := integral[y0*stride+x0]
			bb := integral[y0*stride+(x1+1)]
			c := integral[(y1+1)*stride+x0]
			d := integral[(y1+1)*stride+(x1+1)]
			sum := d - bb - c + a
			n := float64((y1 - y0 + 1) * (x1 - x0 + 1))
			out[y*w+x] = sum / n
		}
	}
	return out
}

// Colorize renders the score field with a cold-to-hot heat LUT.
func (hm *HeatMap) Colorize() *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, hm.W, hm.H))
	for y := 0; y < hm.H; y++ {
		for x := 0; x < hm.W; x++ {
			t := hm.Score[y*hm.W+x]
			r, g, b := heatLUT(t)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

// ColorizeScaled block-averages the score field by f then applies the
// LUT, giving cleaner mini panels than averaging already-LUTed colors.
func (hm *HeatMap) ColorizeScaled(f int) *image.RGBA {
	w, h := hm.W/f, hm.H/f
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var s float64
			for dy := 0; dy < f; dy++ {
				for dx := 0; dx < f; dx++ {
					s += hm.Score[(y*f+dy)*hm.W+(x*f+dx)]
				}
			}
			s /= float64(f * f)
			r, g, b := heatLUT(s)
			img.SetRGBA(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	return img
}

func heatLUT(t float64) (uint8, uint8, uint8) {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	// blue -> cyan -> green -> yellow -> red -> dark red
	stops := []struct {
		at      float64
		r, g, b float64
	}{
		{0.00, 0.05, 0.10, 0.45},
		{0.20, 0.00, 0.45, 0.95},
		{0.40, 0.00, 0.85, 0.80},
		{0.60, 0.35, 0.90, 0.25},
		{0.78, 1.00, 0.90, 0.10},
		{0.92, 1.00, 0.42, 0.05},
		{1.00, 0.75, 0.05, 0.05},
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].at {
			a := stops[i-1]
			b := stops[i]
			k := (t - a.at) / (b.at - a.at)
			return clamp255(lerp(a.r, b.r, k) * 255),
				clamp255(lerp(a.g, b.g, k) * 255),
				clamp255(lerp(a.b, b.b, k) * 255)
		}
	}
	return 190, 12, 12
}

// BoxDownscale averages f x f blocks. f must divide width and height.
func BoxDownscale(src *image.RGBA, f int) *image.RGBA {
	b := src.Bounds()
	w, h := b.Dx()/f, b.Dy()/f
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var rs, gs, bs float64
			for dy := 0; dy < f; dy++ {
				for dx := 0; dx < f; dx++ {
					c := src.RGBAAt(x*f+dx, y*f+dy)
					rs += float64(c.R)
					gs += float64(c.G)
					bs += float64(c.B)
				}
			}
			n := float64(f * f)
			dst.SetRGBA(x, y, color.RGBA{R: uint8(rs / n), G: uint8(gs / n), B: uint8(bs / n), A: 255})
		}
	}
	return dst
}
