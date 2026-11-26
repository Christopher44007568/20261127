package studio

import "math"

type Stop struct {
	Pos   float32
	Color RGBA
}

func sampleStops(stops []Stop, t float32) RGBA {
	if len(stops) == 0 {
		return RGBA{}
	}
	t = clamp01f(t)
	if t <= stops[0].Pos {
		return stops[0].Color
	}
	last := len(stops) - 1
	if t >= stops[last].Pos {
		return stops[last].Color
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].Pos {
			span := stops[i].Pos - stops[i-1].Pos
			u := float32(0)
			if span > 0 {
				u = (t - stops[i-1].Pos) / span
			}
			return lerpRGBA(stops[i-1].Color, stops[i].Color, u)
		}
	}
	return stops[last].Color
}

// LinearGradient paints an opaque directional gradient. Alpha comes only from
// the colour stops; a clip mask is applied later by the clip stack.
func LinearGradient(w, h int, ax, ay, bx, by float32, stops []Stop) *FloatImage {
	im := NewImage(w, h)
	dx, dy := bx-ax, by-ay
	denom := dx*dx + dy*dy
	if denom == 0 {
		denom = 1
	}
	for y := 0; y < h; y++ {
		py := float32(y) + 0.5
		for x := 0; x < w; x++ {
			px := float32(x) + 0.5
			t := ((px-ax)*dx + (py-ay)*dy) / denom
			im.Set(x, y, Premultiply(sampleStops(stops, t)))
		}
	}
	return im
}

// RadialGradient interpolates straight colour and alpha, then premultiplies
// once at the end of sampling.
func RadialGradient(w, h int, cx, cy, radius float32, stops []Stop) *FloatImage {
	im := NewImage(w, h)
	if radius <= 0 {
		return im
	}
	for y := 0; y < h; y++ {
		dy := float32(y) + 0.5 - cy
		for x := 0; x < w; x++ {
			dx := float32(x) + 0.5 - cx
			t := sqrt32(dx*dx+dy*dy) / radius
			im.Set(x, y, Premultiply(sampleStops(stops, t)))
		}
	}
	return im
}

// AlphaMask is the soft moving curtain used by the stack. Values stay well
// away from zero so the mask reads as a translucent modulation, not a hard
// eraser. The diagonal period is continuous at any tiled boundary because it
// is derived from a phase wrapped with cosine.
func AlphaMask(w, h int) []float32 {
	mask := make([]float32, w*h)
	const period = 230.0
	for y := 0; y < h; y++ {
		py := float64(y) + 0.5
		for x := 0; x < w; x++ {
			px := float64(x) + 0.5
			phase := (px*0.72 + py*0.36) * 2 * math.Pi / period
			wave := 0.5 + 0.5*math.Cos(phase)
			edge := 0.5 + 0.5*math.Sin(phase*0.5+0.8)
			v := 0.34 + 0.48*wave + 0.18*edge
			mask[y*w+x] = float32(clamp01(v))
		}
	}
	return mask
}

// PeriodicTileColor is exactly 1-periodic in u and v. Repeating this tile
// therefore has matching values on both sides of every seam.
func PeriodicTileColor(u, v float64) RGBA {
	tau := 2 * math.Pi
	a := 0.5 + 0.5*math.Sin(tau*u+0.55*math.Sin(tau*v))
	b := 0.5 + 0.5*math.Cos(tau*v-0.45*math.Sin(tau*u))
	dx := math.Sin(tau * (u - 0.5))
	dy := math.Sin(tau * (v - 0.5))
	radial := clamp01(1 - math.Sqrt(dx*dx+dy*dy)/math.Sqrt2)

	palette := []Stop{
		{Pos: 0.00, Color: Hex("07101e")},
		{Pos: 0.34, Color: Hex("126b8e")},
		{Pos: 0.63, Color: Hex("2ed5c4")},
		{Pos: 0.84, Color: Hex("ffb45c")},
		{Pos: 1.00, Color: Hex("ff4f89")},
	}
	base := sampleStops(palette, float32(a))
	glow := Hex("ffe0a3")
	base = lerpRGBA(base, glow, float32(0.42*radial*radial))
	base = lerpRGBA(base, Hex("3e71ff"), float32(0.16*b))
	base.A = 1
	return base
}

func TileImage(tileW, tileH, cols, rows int) *FloatImage {
	im := NewImage(tileW*cols, tileH*rows)
	for y := 0; y < im.H; y++ {
		v := float64(y%tileH) / float64(tileH)
		for x := 0; x < im.W; x++ {
			u := float64(x%tileW) / float64(tileW)
			c := PeriodicTileColor(u, v)
			im.Set(x, y, Premultiply(c))
		}
	}
	return im
}
