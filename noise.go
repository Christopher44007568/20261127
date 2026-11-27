package main

import "math"

// hash1 returns a deterministic pseudo-random value in [0,1) for integer i.
func hash1(i int) float64 {
	x := uint64(int64(i))
	x ^= x >> 33
	x *= 0xff51afd7ed558ccd
	x ^= x >> 33
	x *= 0xc4ceb9fe1a85ec53
	x ^= x >> 33
	return float64(x>>11) * 0x1.0p-53
}

func smooth(t float64) float64 { return t * t * (3 - 2*t) }

// valueNoise1D is a smooth periodic-free 1D value noise.
func valueNoise1D(x float64) float64 {
	i := int(math.Floor(x))
	f := x - float64(i)
	a := hash1(i)
	b := hash1(i + 1)
	return a + (b-a)*smooth(f)
}

// fbm stacks two noise octaves into a band-limited streaking signal.
func fbm(x float64) float64 {
	return valueNoise1D(x)*0.65 + valueNoise1D(x*2.3+17.0)*0.35
}
