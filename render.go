package main

import (
	"image"
	"image/color"
	"image/draw"
	"math"
)

const renderScale = 4

type canvas struct {
	img   *image.RGBA
	scale int
	pad   int
}

func newCanvas(g *Grid, scale int) *canvas {
	if scale <= 0 {
		scale = renderScale
	}
	pad := 5 * scale
	bounds := image.Rect(0, 0, g.W*scale+2*pad, g.H*scale+2*pad)
	return &canvas{img: image.NewRGBA(bounds), scale: scale, pad: pad}
}

func (c *canvas) clear(col color.RGBA) {
	draw.Draw(c.img, c.img.Bounds(), &image.Uniform{C: col}, image.Point{}, draw.Src)
}

func (c *canvas) cellRect(x, y int) image.Rectangle {
	s := c.scale
	return image.Rect(c.pad+x*s, c.pad+y*s, c.pad+(x+1)*s, c.pad+(y+1)*s)
}

func (c *canvas) mapX(x float64) int {
	return c.pad + int(math.Round(x*float64(c.scale)))
}

func (c *canvas) mapY(y float64) int {
	return c.pad + int(math.Round(y*float64(c.scale)))
}

func (c *canvas) fillCell(x, y int, col color.RGBA) {
	draw.Draw(c.img, c.cellRect(x, y), &image.Uniform{C: col}, image.Point{}, draw.Src)
}

func fillRect(img *image.RGBA, r image.Rectangle, col color.RGBA) {
	draw.Draw(img, r.Intersect(img.Bounds()), &image.Uniform{C: col}, image.Point{}, draw.Src)
}

func hslColor(h, s, l float64) color.RGBA {
	h = math.Mod(h, 1)
	if h < 0 {
		h += 1
	}
	hue2rgb := func(p, q, t float64) float64 {
		if t < 0 {
			t += 1
		}
		if t > 1 {
			t -= 1
		}
		if t < 1.0/6 {
			return p + (q-p)*6*t
		}
		if t < 1.0/2 {
			return q
		}
		if t < 2.0/3 {
			return p + (q-p)*(2.0/3-t)*6
		}
		return p
	}
	var r, gg, b float64
	if s == 0 {
		r, gg, b = l, l, l
	} else {
		q := l
		if l < 0.5 {
			q = l * (1 + s)
		} else {
			q = l + s - l*s
		}
		p := 2*l - q
		r = hue2rgb(p, q, h+1.0/3)
		gg = hue2rgb(p, q, h)
		b = hue2rgb(p, q, h-1.0/3)
	}
	return color.RGBA{R: uint8(r*255 + .5), G: uint8(gg*255 + .5), B: uint8(b*255 + .5), A: 255}
}

func labelColor(label int) color.RGBA {
	hue := math.Mod(float64(label)*0.618033988749895+0.04, 1)
	return hslColor(hue, 0.64, 0.54)
}

func heatColor(t float64) color.RGBA {
	t = clamp01(t)
	stops := []struct {
		t float64
		c [3]float64
	}{
		{0.00, [3]float64{68, 1, 84}},
		{0.25, [3]float64{59, 82, 139}},
		{0.50, [3]float64{33, 145, 140}},
		{0.75, [3]float64{94, 201, 98}},
		{1.00, [3]float64{253, 231, 37}},
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].t {
			a := stops[i-1]
			b := stops[i]
			u := (t - a.t) / (b.t - a.t)
			return color.RGBA{
				R: uint8(a.c[0] + (b.c[0]-a.c[0])*u + .5),
				G: uint8(a.c[1] + (b.c[1]-a.c[1])*u + .5),
				B: uint8(a.c[2] + (b.c[2]-a.c[2])*u + .5),
				A: 255,
			}
		}
	}
	return color.RGBA{R: 253, G: 231, B: 37, A: 255}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func blendShade(col color.RGBA, shade float64) color.RGBA {
	shade = math.Max(0.45, math.Min(1.25, shade))
	r := float64(col.R) * shade
	g := float64(col.G) * shade
	b := float64(col.B) * shade
	return color.RGBA{R: clampUint8(r), G: clampUint8(g), B: clampUint8(b), A: 255}
}

func clampUint8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + .5)
}

func terrainShade(g *Grid, x, y int) float64 {
	xl := g.ClampedElevation(x-1, y)
	xr := g.ClampedElevation(x+1, y)
	yu := g.ClampedElevation(x, y-1)
	yd := g.ClampedElevation(x, y+1)
	dx := xr - xl
	dy := yd - yu
	shade := 1.0 + 0.10*(-dx+dy)
	return math.Max(0.72, math.Min(1.18, shade))
}

func blockedColor() color.RGBA {
	return color.RGBA{R: 13, G: 16, B: 22, A: 255}
}

func drawCircle(img *image.RGBA, cx, cy, radius int, col color.RGBA) {
	r2 := radius * radius
	bounds := img.Bounds()
	for y := cy - radius; y <= cy+radius; y++ {
		for x := cx - radius; x <= cx+radius; x++ {
			dx := x - cx
			dy := y - cy
			if dx*dx+dy*dy <= r2 && x >= bounds.Min.X && x < bounds.Max.X && y >= bounds.Min.Y && y < bounds.Max.Y {
				img.SetRGBA(x, y, col)
			}
		}
	}
}

func drawSeedMarker(c *canvas, seed Seed, index int) {
	s := c.scale
	cx := c.mapX(float64(seed.X) + .5)
	cy := c.mapY(float64(seed.Y) + .5)
	outer := int(1.35*float64(s) + .5)
	inner := int(0.82*float64(s) + .5)
	drawCircle(c.img, cx, cy, outer+1, color.RGBA{R: 0, G: 0, B: 0, A: 220})
	drawCircle(c.img, cx, cy, outer, color.RGBA{R: 255, G: 255, B: 255, A: 255})
	drawCircle(c.img, cx, cy, inner, labelColor(index))
}
