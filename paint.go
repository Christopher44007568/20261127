package main

import (
	"image/color"
	"math"
)

func lerpByte(a, b uint8, t float64) uint8 {
	value := float64(a) + (float64(b)-float64(a))*t
	return uint8(math.Round(math.Max(0, math.Min(255, value))))
}

func lerpColor(a, b color.NRGBA, t float64) color.NRGBA {
	return color.NRGBA{
		R: lerpByte(a.R, b.R, t),
		G: lerpByte(a.G, b.G, t),
		B: lerpByte(a.B, b.B, t),
		A: 255,
	}
}

func multiGradient(t float64, colors []color.NRGBA) color.NRGBA {
	if t <= 0 {
		return colors[0]
	}
	if t >= 1 {
		return colors[len(colors)-1]
	}
	scaled := t * float64(len(colors)-1)
	index := int(scaled)
	if index >= len(colors)-1 {
		return colors[len(colors)-1]
	}
	return lerpColor(colors[index], colors[index+1], scaled-float64(index))
}

func DiagonalGradient(colors ...color.NRGBA) PaintFunc {
	return func(x, y int) color.NRGBA {
		t := (float64(x)*0.75 + float64(y)*1.25) / 1500
		t = math.Max(0, math.Min(1, t))
		return multiGradient(t, colors)
	}
}

func RadialPop(centerX, centerY, radius float64, colors ...color.NRGBA) PaintFunc {
	return func(x, y int) color.NRGBA {
		dx := float64(x) + 0.5 - centerX
		dy := float64(y) + 0.5 - centerY
		t := math.Sqrt(dx*dx+dy*dy) / radius
		return multiGradient(t, colors)
	}
}

func DiagonalStripes(width int, colors ...color.NRGBA) PaintFunc {
	return func(x, y int) color.NRGBA {
		band := (x + y) / width
		return colors[band%len(colors)]
	}
}

func Dots(centerSpacing, radius float64, base, dot color.NRGBA) PaintFunc {
	return func(x, y int) color.NRGBA {
		gx := math.Mod(float64(x), centerSpacing)
		gy := math.Mod(float64(y), centerSpacing)
		dx := gx - centerSpacing/2
		dy := gy - centerSpacing/2
		if dx*dx+dy*dy <= radius*radius {
			return dot
		}
		return base
	}
}
