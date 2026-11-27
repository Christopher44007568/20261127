package main

import "math"

// splitmix64 is a tiny deterministic mixing function used for all
// coordinate-keyed randomness (brick tints, lattice values, grain).
func splitmix64(x uint64) uint64 {
	x += 0x9E3779B97F4A7C15
	z := x
	z = (z ^ (z >> 30)) * 0xBF58476D1CE4E5B9
	z = (z ^ (z >> 27)) * 0x94D049BB133111EB
	return z ^ (z >> 31)
}

// hash64 maps an integer lattice point and seed to a stable uint64.
func hash64(x, y, seed int64) uint64 {
	h := uint64(x)*0x9E3779B97F4A7C15 ^
		uint64(y)*0xC2B2AE3D27D4EB4F ^
		uint64(seed)*0x165667B19E3779F9
	return splitmix64(h)
}

// hashFloat returns a deterministic float in [0,1).
func hashFloat(x, y, seed int64) float64 {
	return float64(hash64(x, y, seed)>>11) / float64(uint64(1)<<53)
}

// frand derives a repeatable random float from an object seed + salt.
func frand(seed uint64, salt int64) float64 {
	return float64(splitmix64(seed+uint64(salt)*0x9E3779B97F4A7C15)>>11) /
		float64(uint64(1)<<53)
}

func smoothStep01(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

func smoother(t float64) float64 {
	return t * t * t * (t*(t*6-15) + 10)
}

func lerp(a, b, t float64) float64 { return a + (b-a)*t }

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// ValueNoise2D is smooth quintic value noise. Coordinates are in
// arbitrary lattice units; negative coordinates are supported.
func ValueNoise2D(x, y float64, seed int64) float64 {
	xi := int64(math.Floor(x))
	yi := int64(math.Floor(y))
	xf := x - float64(xi)
	yf := y - float64(yi)
	u := smoother(xf)
	v := smoother(yf)

	a := hashFloat(xi, yi, seed)
	b := hashFloat(xi+1, yi, seed)
	c := hashFloat(xi, yi+1, seed)
	d := hashFloat(xi+1, yi+1, seed)
	return lerp(lerp(a, b, u), lerp(c, d, u), v)
}

// FBM is fractal Brownian motion over value noise, normalized to [0,1].
func FBM(x, y float64, seed int64, octaves int) float64 {
	sum, amp, norm := 0.0, 0.5, 0.0
	freq := 1.0
	for i := 0; i < octaves; i++ {
		sum += amp * ValueNoise2D(x*freq, y*freq, seed+int64(i)*1013)
		norm += amp
		amp *= 0.5
		freq *= 2.03
	}
	return sum / norm
}

// TiledValueNoise2D evaluates value noise whose integer lattice wraps
// every period cells, so a texture baked over an exact period is seamless.
func TiledValueNoise2D(x, y float64, period int, seed int64) float64 {
	xi := int64(math.Floor(x))
	yi := int64(math.Floor(y))
	xf := x - float64(xi)
	yf := y - float64(yi)
	u := smoother(xf)
	v := smoother(yf)
	p := int64(period)

	wx := func(v int64) int64 {
		m := v % p
		if m < 0 {
			m += p
		}
		return m
	}
	x0, x1 := wx(xi), wx(xi+1)
	y0, y1 := wx(yi), wx(yi+1)

	a := hashFloat(x0, y0, seed)
	b := hashFloat(x1, y0, seed)
	c := hashFloat(x0, y1, seed)
	d := hashFloat(x1, y1, seed)
	return lerp(lerp(a, b, u), lerp(c, d, u), v)
}

// TiledFBMOverTile sums periodic octaves over TileW x TileH pixels.
// baseCell is the first-octave lattice cell size in pixels; each octave
// doubles the cell and halves the lattice period. Every resulting cell
// size must divide TileW/TileH exactly (use powers of two), otherwise
// the lattice wrap would introduce a seam at the tile border.
func TiledFBMOverTile(px, py int, seed int64, octaves, baseCell int) float64 {
	sum, amp, norm := 0.0, 0.5, 0.0
	cell := baseCell
	for i := 0; i < octaves; i++ {
		periodX := TileW / cell
		periodY := TileH / cell
		nx := float64(px) / float64(cell)
		ny := float64(py) / float64(cell)
		n := tiledNoiseRect(nx, ny, periodX, periodY, seed+int64(i)*2027)
		sum += amp * n
		norm += amp
		amp *= 0.5
		cell *= 2
	}
	return sum / norm
}

func tiledNoiseRect(x, y float64, periodX, periodY int, seed int64) float64 {
	xi := int64(math.Floor(x))
	yi := int64(math.Floor(y))
	xf := x - float64(xi)
	yf := y - float64(yi)
	u := smoother(xf)
	v := smoother(yf)

	wx := func(v int64, p int) int64 {
		m := v % int64(p)
		if m < 0 {
			m += int64(p)
		}
		return m
	}
	x0, x1 := wx(xi, periodX), wx(xi+1, periodX)
	y0, y1 := wx(yi, periodY), wx(yi+1, periodY)

	a := hashFloat(x0, y0, seed)
	b := hashFloat(x1, y0, seed)
	c := hashFloat(x0, y1, seed)
	d := hashFloat(x1, y1, seed)
	return lerp(lerp(a, b, u), lerp(c, d, u), v)
}
