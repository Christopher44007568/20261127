package main

import "math"

func hexLinear(hex uint32) Vec3 {
	r := float64((hex>>16)&255) / 255
	g := float64((hex>>8)&255) / 255
	b := float64(hex&255) / 255
	return v3(srgbToLinear(r), srgbToLinear(g), srgbToLinear(b))
}

var (
	colInk       = hexLinear(0x07101f)
	colPanel     = hexLinear(0x101c31)
	colPaper     = hexLinear(0xe8f1ff)
	colGridDark  = hexLinear(0x61738e)
	colGridLight = hexLinear(0xffffff)
	colCyan      = hexLinear(0x62e6ff)
	colRed       = hexLinear(0xff5a52)
	colViolet    = hexLinear(0x9b6cff)
	colAmber     = hexLinear(0xffc857)
	colGreen     = hexLinear(0x55e38a)
)

func roundedBoxSDF(p, center Vec2, half Vec2, radius float64) float64 {
	q := v2(math.Abs(p.X-center.X)-half.X+radius, math.Abs(p.Y-center.Y)-half.Y+radius)
	outside := v2(math.Max(q.X, 0), math.Max(q.Y, 0)).Len()
	inside := math.Min(math.Max(q.X, q.Y), 0)
	return outside + inside - radius
}

func benchColor(p Vec2) Vec3 {
	t := clamp01((p.Y + 5.0) / 10.0)
	base := hexLinear(0x0a1326).Mul(1 - t).Add(hexLinear(0x243653).Mul(t))
	glow := math.Exp(-((p.X+1.1)*(p.X+1.1)/38 + (p.Y-0.2)*(p.Y-0.2)/16))
	base = base.Add(hexLinear(0x2f78a8).Mul(0.16 * glow))

	cardCenter := v2(-1.55, 0.0)
	cardHalf := v2(4.85, 4.28)
	d := roundedBoxSDF(p, cardCenter, cardHalf, 0.28)
	aa := 0.012
	cover := 1 - smoothstep(-aa, aa, d)
	if cover <= 0 {
		return base
	}

	q := p.Add(v2(20.3, 11.7))
	cell := 0.46
	ix := int(math.Floor(q.X / cell))
	iy := int(math.Floor(q.Y / cell))
	lightSquare := (ix+iy)&1 == 0
	card := hexLinear(0xd5e1f2)
	if lightSquare {
		card = hexLinear(0xf3f8ff)
	} else {
		card = hexLinear(0xaebed8)
	}

	fx := math.Mod(q.X, cell) / cell
	fy := math.Mod(q.Y, cell) / cell
	line := math.Max(
		1-smoothstep(0.015, 0.055, math.Min(fx, 1-fx)),
		1-smoothstep(0.015, 0.055, math.Min(fy, 1-fy)),
	)
	card = card.Mul(1 - 0.20*line).Add(colGridDark.Mul(0.20 * line))

	ringDist := math.Abs(p.Sub(v2(-1.15, 0.15)).Len() - 1.18)
	ring := 1 - smoothstep(0.018, 0.055, ringDist)
	card = card.Mul(1 - 0.32*ring).Add(hexLinear(0x244b7a).Mul(0.32 * ring))

	ringDist2 := math.Abs(p.Sub(v2(-3.55, 2.55)).Len() - 0.62)
	ring2 := 1 - smoothstep(0.018, 0.05, ringDist2)
	card = card.Mul(1 - 0.26*ring2).Add(colAmber.Mul(0.26 * ring2))

	if p.X > -5.7 && p.X < -3.25 && p.Y > -3.72 && p.Y < -3.18 {
		bars := []uint32{0xe84d5b, 0xff8a3d, 0xf5d547, 0x55c96b, 0x45b7e8, 0x5968e8, 0x9b55e8}
		idx := int(clamp((p.X+5.7)/2.45*float64(len(bars)), 0, float64(len(bars)-1)))
		bar := hexLinear(bars[idx])
		card = card.Mul(0.42).Add(bar.Mul(0.58))
	}

	return base.Mul(1 - cover).Add(card.Mul(cover))
}

func paintBench(c *Canvas) {
	for y := 0; y < c.H; y++ {
		for x := 0; x < c.W; x++ {
			p := c.PixelToWorld(float64(x)+0.5, float64(y)+0.5)
			c.Set(x, y, benchColor(p))
		}
	}
}

func strokeWorldPolygon(c *Canvas, p Polygon2, width float64, col Vec3, alpha float64) {
	for i := range p.Verts {
		a := p.Verts[i]
		b := p.Verts[(i+1)%len(p.Verts)]
		c.LineWorld(a, b, LineStyle{Color: col, Width: width, Alpha: alpha})
	}
}
