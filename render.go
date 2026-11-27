package main

import (
	"image"
	"image/color"
	"math"
)

type PixelInfo struct {
	X   int
	Y   int
	Hit Hit
	FP  Footprint
}

type FootprintCache struct {
	Pixels []PixelInfo
	At     map[int]int
}

func BuildFootprints(scene *Scene) *FootprintCache {
	cache := &FootprintCache{
		Pixels: make([]PixelInfo, 0, scene.Width*scene.Height/2),
		At:     make(map[int]int, scene.Width*scene.Height/2),
	}
	for y := 0; y < scene.Height; y++ {
		for x := 0; x < scene.Width; x++ {
			h := scene.hit(float64(x), float64(y))
			if !h.OK {
				continue
			}
			fp := scene.footprint(x, y, h)
			cache.At[y*scene.Width+x] = len(cache.Pixels)
			cache.Pixels = append(cache.Pixels, PixelInfo{X: x, Y: y, Hit: h, FP: fp})
		}
	}
	return cache
}

type RenderMode int

const (
	ModeFixed RenderMode = iota
	ModeAniso
	ModeReference
)

type RenderStats struct {
	Mode              RenderMode
	Budget            int
	BoardPixels       int
	TotalTaps         int
	MaxTaps           int
	MaxProbes         int
	DegradedPixels    int
	DirectionPixels   int
	MaxAnisotropy     float64
	MaxMajorTexels    float64
	MaxMinorTexels    float64
	LongAxisVariation float64
	CrossAxisEnergy   float64
}

func backgroundImage(scene *Scene) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, scene.Width, scene.Height))
	for y := 0; y < scene.Height; y++ {
		t := float64(y) / float64(scene.Height-1)
		top := RGB{0.855, 0.885, 0.920}
		bottom := RGB{0.925, 0.940, 0.955}
		c := top.Mul(1 - t).Add(bottom.Mul(t)).Clamp()
		for x := 0; x < scene.Width; x++ {
			setRGB(img, x, y, c)
		}
	}
	return img
}

func renderBoard(scene *Scene, tex *Texture, pattern *StripePattern, fps *FootprintCache, mode RenderMode, budget int) (*image.RGBA, *RenderStats) {
	img := backgroundImage(scene)
	stats := &RenderStats{Mode: mode, Budget: budget, BoardPixels: len(fps.Pixels)}
	for _, p := range fps.Pixels {
		var c RGB
		var fs FilterStats
		switch mode {
		case ModeFixed:
			c, fs = fixedFour(tex, p.Hit.UV)
		case ModeAniso:
			c, fs = anisotropic(tex, p.Hit.UV, p.FP, budget)
		case ModeReference:
			c = referenceColor(pattern, p.Hit.UV, p.FP)
		}
		setRGB(img, p.X, p.Y, c.Clamp())
		stats.TotalTaps += fs.Taps
		if fs.Taps > stats.MaxTaps {
			stats.MaxTaps = fs.Taps
		}
		if fs.Probes > stats.MaxProbes {
			stats.MaxProbes = fs.Probes
		}
		if fs.Degraded {
			stats.DegradedPixels++
		}
		ratio := p.FP.Major / math.Max(p.FP.Minor, 1e-6)
		if ratio > stats.MaxAnisotropy {
			stats.MaxAnisotropy = ratio
		}
		if p.FP.Major > stats.MaxMajorTexels {
			stats.MaxMajorTexels = p.FP.Major
		}
		if p.FP.Minor > stats.MaxMinorTexels {
			stats.MaxMinorTexels = p.FP.Minor
		}
		if ratio >= 1.25 {
			stats.DirectionPixels++
		}
	}
	return img, stats
}

func renderFootprintOverlay(scene *Scene, fps *FootprintCache, base *image.RGBA) *image.RGBA {
	img := cloneImage(base)
	cyan := RGB{0.0, 0.95, 0.95}
	drawText(img, 20, 18, "SCREEN DERIVATIVE FOOTPRINTS 10X", RGB{0.08, 0.14, 0.18}, 2)
	for y := 36; y < scene.Height; y += 46 {
		for x := 30; x < scene.Width; x += 52 {
			index, ok := fps.At[y*scene.Width+x]
			if !ok {
				continue
			}
			p := fps.Pixels[index]
			drawFootprintEllipse(img, p, cyan)
			setRGB(img, p.X, p.Y, RGB{0, 0.35, 0.35})
		}
	}
	return img
}

func drawFootprintEllipse(img *image.RGBA, p PixelInfo, c RGB) {
	const segments = 28
	const displayScale = 10.0
	majorDir, okMajor := fpScreenDirection(p.FP, p.FP.AxisMajor)
	minorDir, okMinor := fpScreenDirection(p.FP, p.FP.AxisMinor)
	if !okMajor || !okMinor {
		return
	}
	center := Vec2{float64(p.X), float64(p.Y)}
	semiMajor := 0.5 * displayScale * p.FP.Major
	semiMinor := 0.5 * displayScale * p.FP.Minor
	var prev Vec2
	havePrev := false
	for i := 0; i <= segments; i++ {
		a := 2 * math.Pi * float64(i) / segments
		screen := center.
			Add(majorDir.Mul(semiMajor * math.Cos(a))).
			Add(minorDir.Mul(semiMinor * math.Sin(a)))
		if havePrev {
			drawLine(img, prev, screen, c)
		}
		prev = screen
		havePrev = true
	}
}

func cloneImage(src *image.RGBA) *image.RGBA {
	dst := image.NewRGBA(src.Rect)
	copy(dst.Pix, src.Pix)
	return dst
}

func drawLine(img *image.RGBA, a, b Vec2, c RGB) {
	dx := b.X - a.X
	dy := b.Y - a.Y
	steps := int(math.Ceil(math.Max(math.Abs(dx), math.Abs(dy))))
	if steps < 1 {
		steps = 1
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := int(math.Round(mix(a.X, b.X, t)))
		y := int(math.Round(mix(a.Y, b.Y, t)))
		if image.Pt(x, y).In(img.Rect) {
			setRGB(img, x, y, c)
		}
	}
}

func setRGB(img *image.RGBA, x, y int, c RGB) {
	c = c.Clamp()
	img.SetRGBA(x, y, color.RGBA{
		R: uint8(math.Round(c.R * 255)),
		G: uint8(math.Round(c.G * 255)),
		B: uint8(math.Round(c.B * 255)),
		A: 255,
	})
}

func getRGB(img *image.RGBA, x, y int) RGB {
	i := img.PixOffset(x, y)
	return RGB{
		R: float64(img.Pix[i+0]) / 255,
		G: float64(img.Pix[i+1]) / 255,
		B: float64(img.Pix[i+2]) / 255,
	}
}
