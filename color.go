package main

import "image/color"

type RGB struct {
	R, G, B float64
}

func (c RGB) Clamp() RGB {
	return RGB{clampFloat(c.R, 0, 1), clampFloat(c.G, 0, 1), clampFloat(c.B, 0, 1)}
}

func (c RGB) RGBA() color.RGBA {
	q := c.Clamp()
	return color.RGBA{R: uint8(q.R*255 + 0.5), G: uint8(q.G*255 + 0.5), B: uint8(q.B*255 + 0.5), A: 255}
}

func lerpRGB(a, b RGB, t float64) RGB {
	return RGB{
		R: a.R + (b.R-a.R)*t,
		G: a.G + (b.G-a.G)*t,
		B: a.B + (b.B-a.B)*t,
	}
}

func paletteStops(t float64, stops []RGB) RGB {
	if t <= 0 {
		return stops[0]
	}
	if t >= 1 {
		return stops[len(stops)-1]
	}
	scaled := t * float64(len(stops)-1)
	i := int(scaled)
	return lerpRGB(stops[i], stops[i+1], scaled-float64(i))
}

var viridisStops = []RGB{
	{0.267, 0.005, 0.329},
	{0.283, 0.141, 0.458},
	{0.254, 0.265, 0.530},
	{0.207, 0.372, 0.553},
	{0.164, 0.471, 0.558},
	{0.128, 0.567, 0.551},
	{0.135, 0.659, 0.518},
	{0.267, 0.749, 0.441},
	{0.478, 0.821, 0.318},
	{0.741, 0.873, 0.150},
	{0.993, 0.906, 0.144},
}

func Viridis(t float64) RGB { return paletteStops(clampFloat(t, 0, 1), viridisStops) }

var heatStops = []RGB{
	{0.02, 0.05, 0.25},
	{0.00, 0.35, 0.85},
	{0.00, 0.85, 0.85},
	{0.95, 0.95, 0.20},
	{0.95, 0.25, 0.10},
	{0.55, 0.00, 0.25},
}

func HeatColor(t float64) RGB { return paletteStops(clampFloat(t, 0, 1), heatStops) }

var tsdfNegative = RGB{0.10, 0.32, 0.85}
var tsdfZero = RGB{0.98, 0.98, 0.96}
var tsdfPositive = RGB{0.95, 0.48, 0.12}

func TSDFColor(v float64) RGB {
	v = clampFloat(v, -1, 1)
	if v < 0 {
		return lerpRGB(tsdfZero, tsdfNegative, -v)
	}
	return lerpRGB(tsdfZero, tsdfPositive, v)
}
