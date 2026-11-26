package main

import "math"

type RGB struct {
	R, G, B uint8
}

func rgbf(r, g, b float64) RGB {
	return RGB{clampUint8(r * 255), clampUint8(g * 255), clampUint8(b * 255)}
}

func clampUint8(v float64) uint8 {
	if math.IsNaN(v) || v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(math.Round(v))
}

func lerpRGB(a, b RGB, t float64) RGB {
	t = clampFloat(t, 0, 1)
	return RGB{
		uint8(lerpFloat(float64(a.R), float64(b.R), t)),
		uint8(lerpFloat(float64(a.G), float64(b.G), t)),
		uint8(lerpFloat(float64(a.B), float64(b.B), t)),
	}
}

// HeatColor maps 0..1 through blue, cyan, green, yellow and red.
func HeatColor(t float64) RGB {
	t = clampFloat(t, 0, 1)
	stops := []RGB{
		{35, 52, 148},
		{34, 144, 203},
		{46, 196, 182},
		{171, 221, 164},
		{253, 219, 104},
		{239, 138, 48},
		{202, 56, 44},
	}
	x := t * float64(len(stops)-1)
	i := int(x)
	if i >= len(stops)-1 {
		return stops[len(stops)-1]
	}
	return lerpRGB(stops[i], stops[i+1], x-float64(i))
}

func NormalColor(n V3) RGB {
	n = Normalize(n)
	return rgbf(0.5+0.5*n.X, 0.5+0.5*n.Y, 0.5+0.5*n.Z)
}

func ElevationColor(t float64) RGB {
	return HeatColor(0.15 + 0.7*t)
}
