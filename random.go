package flakepaint

import "math"

func mix64(x uint64) uint64 {
	x += 0x9e3779b97f4a7c15
	x = (x ^ (x >> 30)) * 0xbf58476d1ce4e5b9
	x = (x ^ (x >> 27)) * 0x94d049bb133111eb
	return x ^ (x >> 31)
}

func hash2i(x, y int64, seed uint64) uint64 {
	ux := uint64(x) * 0x9e3779b185ebca87
	uy := uint64(y) * 0xc2b2ae3d27d4eb4f
	return mix64(ux ^ uy ^ mix64(seed))
}

func unitFromHash(h uint64) float64 {
	return float64(h>>11) * (1.0 / (1 << 53))
}

func hashUnit(x, y int64, seed uint64, stream uint64) float64 {
	return unitFromHash(hash2i(x, y, seed^mix64(stream+0x517cc1b727220a95)))
}

func hashSigned(x, y int64, seed uint64, stream uint64) float64 {
	return hashUnit(x, y, seed, stream)*2 - 1
}

func valueNoise2D(x, y float64, seed uint64) float64 {
	x0 := int64(math.Floor(x))
	y0 := int64(math.Floor(y))
	fx := x - float64(x0)
	fy := y - float64(y0)
	ux := fx * fx * (3 - 2*fx)
	uy := fy * fy * (3 - 2*fy)

	a := hashUnit(x0, y0, seed, 101)
	b := hashUnit(x0+1, y0, seed, 101)
	c := hashUnit(x0, y0+1, seed, 101)
	d := hashUnit(x0+1, y0+1, seed, 101)
	top := a + (b-a)*ux
	bottom := c + (d-c)*ux
	return top + (bottom-top)*uy
}

func fbm2D(x, y float64, seed uint64) float64 {
	sum := 0.0
	amp := 0.55
	freq := 1.0
	norm := 0.0
	for i := uint64(0); i < 4; i++ {
		sum += valueNoise2D(x*freq, y*freq, seed+i*0x9e3779b9) * amp
		norm += amp
		amp *= 0.48
		freq *= 2.03
	}
	return sum / norm
}

func macroDensityAt(x, y float64, seed uint64) float64 {
	n := fbm2D(x, y, seed+0x6a09e667f3bcc909)
	clusters := valueNoise2D(x*0.37+13.2, y*0.37-7.8, seed+0xbb67ae8584caa73b)
	return clamp(0.22+1.36*n*n*0.72+0.28*clusters, 0.08, 1.0)
}

// macroDensityAtLOD removes octaves whose world-space wavelength is too small
// for the current pixel footprint. This is the key anti-flicker prefilter: a
// distant pixel receives a stable average, never a freshly chosen cell.
func macroDensityAtLOD(x, y, footprintMM float64, seed uint64) float64 {
	sum := 0.0
	amp := 0.55
	freq := 1.0
	norm := 0.0
	cutoffMM := math.Max(footprintMM*6.0, 0.05)
	for i := uint64(0); i < 4; i++ {
		wavelength := 52.0 / freq
		if wavelength >= cutoffMM {
			sum += valueNoise2D(x*freq, y*freq, seed+i*0x9e3779b9) * amp
			norm += amp
		}
		amp *= 0.48
		freq *= 2.03
	}
	var n float64
	if norm > 0 {
		n = sum / norm
	}
	clusters := valueNoise2D(x*0.37+13.2, y*0.37-7.8, seed+0xbb67ae8584caa73b)
	return clamp(0.22+1.36*n*n*0.72+0.28*clusters, 0.08, 1.0)
}
