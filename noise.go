package main

import "math"

func hashUint(x, y, seed uint32) uint32 {
	h := x*0x9e3779b1 ^ y*0x85ebca77 ^ seed*0xc2b2ae35
	h ^= h >> 16
	h *= 0x7feb352d
	h ^= h >> 15
	h *= 0x846ca68b
	h ^= h >> 16
	return h
}

func hash01(x, y, seed uint32) float64 {
	return float64(hashUint(x, y, seed)) / 4294967295.0
}

func valueNoise(x, y float64, seed uint32) float64 {
	ix := int64(math.Floor(x))
	iy := int64(math.Floor(y))
	fx := x - math.Floor(x)
	fy := y - math.Floor(y)
	ux := fx * fx * (3 - 2*fx)
	uy := fy * fy * (3 - 2*fy)

	x0 := uint32(ix)
	y0 := uint32(iy)
	a := hash01(x0, y0, seed)
	b := hash01(x0+1, y0, seed)
	c := hash01(x0, y0+1, seed)
	d := hash01(x0+1, y0+1, seed)
	return lerpFloat(lerpFloat(a, b, ux), lerpFloat(c, d, ux), uy)
}

func fbm(x, y float64, seed uint32, octaves int) float64 {
	sum, amp, norm := 0.0, 0.5, 0.0
	for i := 0; i < octaves; i++ {
		sum += amp * valueNoise(x, y, seed+uint32(i)*0x9e3779b9)
		norm += amp
		amp *= 0.5
		x = x*2.03 + 17.17
		y = y*2.01 - 11.31
	}
	return sum / norm
}

// cellSpeckle returns a stable field with isolated bright cells. The second
// return is a soft cell border used for brushed micro-grooves.
func cellSpeckle(u, v float64, cells float64, seed uint32) (spot, line float64) {
	x := u * cells
	y := v * cells
	ix := uint32(int64(math.Floor(x)))
	iy := uint32(int64(math.Floor(y)))
	fx := x - math.Floor(x)
	fy := y - math.Floor(y)

	best := 2.0
	for oy := int32(-1); oy <= 1; oy++ {
		for ox := int32(-1); ox <= 1; ox++ {
			cx := ix + uint32(ox)
			cy := iy + uint32(oy)
			px := float64(ox) + 0.18 + 0.64*hash01(cx, cy, seed+17)
			py := float64(oy) + 0.18 + 0.64*hash01(cx, cy, seed+29)
			dx := fx - px
			dy := fy - py
			d := math.Hypot(dx, dy)
			if d < best {
				best = d
			}
		}
	}
	spot = 1 - smoothstepFloat(0.03, 0.20, best)

	row := math.Abs(math.Sin((v*cells*0.72 + fbm(u*9, v*2, seed+3, 3)*0.35) * math.Pi))
	line = math.Pow(1-row, 18)
	return spot, line
}
