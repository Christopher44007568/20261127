package main

import "math"

// RGB is a straight (non-premultiplied) color in [0,1].
type RGB struct{ R, G, B float64 }

func clamp255(v float64) uint8 {
	if v <= 0 {
		return 0
	}
	if v >= 255 {
		return 255
	}
	return uint8(math.Round(v))
}

func rgbToRGBA8(c RGB) (uint8, uint8, uint8, uint8) {
	return clamp255(c.R * 255), clamp255(c.G * 255), clamp255(c.B * 255), 255
}

func rgba8ToRGB(r, g, b uint8) RGB {
	return RGB{float64(r) / 255, float64(g) / 255, float64(b) / 255}
}

// hslToRGB converts HSL (h in [0,1), s,l in [0,1]) to RGB.
func hslToRGB(h, s, l float64) RGB {
	h -= math.Floor(h)
	if s < 0 {
		s = 0
	}
	if s > 1 {
		s = 1
	}
	if l < 0 {
		l = 0
	}
	if l > 1 {
		l = 1
	}
	if s == 0 {
		return RGB{l, l, l}
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	hue := func(t float64) float64 {
		if t < 0 {
			t++
		}
		if t > 1 {
			t--
		}
		switch {
		case t < 1.0/6.0:
			return p + (q-p)*6*t
		case t < 0.5:
			return q
		case t < 2.0/3.0:
			return p + (q-p)*(2.0/3.0-t)*6
		default:
			return p
		}
	}
	return RGB{hue(h + 1.0/3.0), hue(h), hue(h - 1.0/3.0)}
}

// rgbToHSL returns h,s,l with h in [0,1).
func rgbToHSL(c RGB) (float64, float64, float64) {
	mx := math.Max(c.R, math.Max(c.G, c.B))
	mn := math.Min(c.R, math.Min(c.G, c.B))
	l := (mx + mn) / 2
	if mx == mn {
		return 0, 0, l
	}
	d := mx - mn
	s := d / (1 - math.Abs(2*l-1))
	var h float64
	switch mx {
	case c.R:
		h = math.Mod((c.G-c.B)/d, 6)
	case c.G:
		h = (c.B-c.R)/d + 2
	default:
		h = (c.R-c.G)/d + 4
	}
	h /= 6
	if h < 0 {
		h++
	}
	return h, s, l
}

// shiftHSL applies additive hue/saturation/lightness shifts and returns RGB.
func shiftHSL(c RGB, dh, ds, dl float64) RGB {
	h, s, l := rgbToHSL(c)
	return hslToRGB(h+dh, clamp01(s+ds), clamp01(l+dl))
}

func mulRGB(c RGB, k float64) RGB {
	return RGB{clamp01(c.R * k), clamp01(c.G * k), clamp01(c.B * k)}
}

func mixRGB(a, b RGB, t float64) RGB {
	t = clamp01(t)
	return RGB{
		a.R + (b.R-a.R)*t,
		a.G + (b.G-a.G)*t,
		a.B + (b.B-a.B)*t,
	}
}

// colorDist returns a perceptual-ish squared-ish distance on 0..255 scale.
// A cheap red-weighted mean gives heatmaps slightly better discrimination
// than plain Euclidean RGB.
func colorDist255(a, b RGB) float64 {
	dr := (a.R - b.R) * 255
	dg := (a.G - b.G) * 255
	db := (a.B - b.B) * 255
	rmean := (a.R + b.R) * 0.5 * 255
	rw := 2 + rmean/256
	return math.Sqrt(rw*dr*dr + 4*dg*dg + (2+(255-rmean)/256)*db*db)
}
