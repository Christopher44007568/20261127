package main

import (
	"image"
	"math"
)

func lumRGBA(img *image.RGBA, x, y int) float64 {
	c := img.RGBAAt(x, y)
	return 0.2126*float64(c.R) + 0.7152*float64(c.G) + 0.0722*float64(c.B)
}

// BoundaryContinuity compares color jumps across tile seams against a
// matched control line one geometric period (BrickW / 2*BrickH) away,
// which has the exact same mortar/brick edge composition. A ratio near 1
// means the seam adds no discontinuity beyond natural texture edges;
// random per-tile rotation produces a ratio well above 1.
func BoundaryContinuity(img *image.RGBA, tile int) (seam, control, ratio float64) {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()

	var seamSum, seamN float64
	var ctrlSum, ctrlN float64

	geoX := BrickW
	geoY := 2 * BrickH
	for y := 0; y < h; y++ {
		for x := 1; x < w; x++ {
			if x%tile != 0 {
				continue
			}
			c0 := img.RGBAAt(x-1, y)
			c1 := img.RGBAAt(x, y)
			seamSum += colorDist255(rgba8ToRGB(c0.R, c0.G, c0.B), rgba8ToRGB(c1.R, c1.G, c1.B))
			seamN++
			cx := x + geoX
			if cx < w {
				a0 := img.RGBAAt(cx-1, y)
				a1 := img.RGBAAt(cx, y)
				ctrlSum += colorDist255(rgba8ToRGB(a0.R, a0.G, a0.B), rgba8ToRGB(a1.R, a1.G, a1.B))
				ctrlN++
			}
		}
	}
	for y := 1; y < h; y++ {
		for x := 0; x < w; x++ {
			if y%tile != 0 {
				continue
			}
			c0 := img.RGBAAt(x, y-1)
			c1 := img.RGBAAt(x, y)
			seamSum += colorDist255(rgba8ToRGB(c0.R, c0.G, c0.B), rgba8ToRGB(c1.R, c1.G, c1.B))
			seamN++
			cy := y + geoY
			if cy < h {
				a0 := img.RGBAAt(x, cy-1)
				a1 := img.RGBAAt(x, cy)
				ctrlSum += colorDist255(rgba8ToRGB(a0.R, a0.G, a0.B), rgba8ToRGB(a1.R, a1.G, a1.B))
				ctrlN++
			}
		}
	}

	seam = seamSum / seamN
	control = ctrlSum / ctrlN
	ratio = seam / control
	return
}

// JointContrast measures luminance separation between mortar centers and
// neighboring brick faces. Comparing processed vs original quantifies
// how well close-up joints survive.
type JointContrast struct {
	MeanContrast float64 `json:"mean_contrast"`
	StrongFrac   float64 `json:"strong_fraction"`
	Samples      int     `json:"samples"`
}

func MeasureJointContrast(img *image.RGBA) JointContrast {
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	var sum float64
	var strong, n int

	add := func(xm, ym, xf, yf int) {
		if xf < 0 || yf < 0 || xf >= w || yf >= h || xm < 0 || ym < 0 || xm >= w || ym >= h {
			return
		}
		d := math.Abs(lumRGBA(img, xm, ym) - lumRGBA(img, xf, yf))
		sum += d
		n++
		if d >= 10 {
			strong++
		}
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			info := BrickAt(x, y)
			if !info.Mortar {
				continue
			}
			// Horizontal joint strip: sample vertically across it.
			if info.Ly < Joint {
				cy := y - info.Ly + Joint/2
				add(x, cy, x, cy+Joint/2+3)
			}
			// Vertical joint strip: sample horizontally across it.
			if info.Lx < Joint {
				cx := x - info.Lx + Joint/2
				add(cx, y, cx+Joint/2+3, y)
			}
		}
	}
	if n == 0 {
		return JointContrast{}
	}
	return JointContrast{MeanContrast: sum / float64(n), StrongFrac: float64(strong) / float64(n), Samples: n}
}
