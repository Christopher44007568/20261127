package field

import (
	"glyphsdf/internal/geom"
)

// CorrectSigns performs the offline equivalent of MSDF error correction. The
// vector fill is available here, so dense bilinear/median reconstruction
// errors can be pushed back into the three channel grids. It preserves sharp
// corners because the target is the exact vector sign rather than a blur.
func CorrectSigns(f *Field, samplesPerCell, iterations int) int {
	if samplesPerCell < 4 {
		samplesPerCell = 8
	}
	if iterations < 1 {
		iterations = 6
	}
	fixed := 0
	for iter := 0; iter < iterations; iter++ {
		for y0 := 0; y0 < f.H-1; y0++ {
			for x0 := 0; x0 < f.W-1; x0++ {
				for sy := 0; sy < samplesPerCell; sy++ {
					ty := (float64(sy) + 0.5) / float64(samplesPerCell)
					y := float64(y0) + 0.5 + ty
					for sx := 0; sx < samplesPerCell; sx++ {
						tx := (float64(sx) + 0.5) / float64(samplesPerCell)
						x := float64(x0) + 0.5 + tx
						if f.correctSample(x, y, x0, y0, tx, ty) {
							fixed++
						}
					}
				}
			}
		}
		// Texel centers are used when magnification goes outside the
		// bilinear center domain and are cheap to make consistent.
		for y := 0; y < f.H; y++ {
			for x := 0; x < f.W; x++ {
				if f.correctCenter(x, y) {
					fixed++
				}
			}
		}
	}
	for i, sd := range f.SDF {
		if sd < -0.9 {
			for ch := 0; ch < 3; ch++ {
				if f.MSDF[ch][i] > -0.9 {
					f.MSDF[ch][i] = -0.9
				}
			}
		}
	}
	return fixed
}

func (f *Field) correctCenter(x, y int) bool {
	idx := y*f.W + x
	rgb := [3]float64{f.MSDF[0][idx], f.MSDF[1][idx], f.MSDF[2][idx]}
	med, ch := medianChannel(rgb)
	p := geom.V{X: float64(x) + 0.5, Y: float64(y) + 0.5}
	inside := geom.Inside(p, f.Shape.Contours)
	margin := 0.05
	var delta float64
	if inside && med < margin {
		delta = margin - med
	} else if !inside && med > -margin {
		delta = -margin - med
	} else {
		return false
	}
	v := clampDist(f.MSDF[ch][idx]+delta, f.Range)
	f.MSDF[ch][idx] = v
	return true
}

func (f *Field) correctSample(x, y float64, x0, y0 int, tx, ty float64) bool {
	i00 := y0*f.W + x0
	i10 := y0*f.W + x0 + 1
	i01 := (y0+1)*f.W + x0
	i11 := (y0+1)*f.W + x0 + 1
	indices := [4]int{i00, i10, i01, i11}
	weights := [4]float64{
		(1 - tx) * (1 - ty),
		tx * (1 - ty),
		(1 - tx) * ty,
		tx * ty,
	}
	var rgb [3]float64
	for ch := 0; ch < 3; ch++ {
		buf := f.MSDF[ch]
		rgb[ch] = buf[i00]*weights[0] + buf[i10]*weights[1] + buf[i01]*weights[2] + buf[i11]*weights[3]
	}
	med, ch := medianChannel(rgb)
	inside := geom.Inside(geom.V{X: x, Y: y}, f.Shape.Contours)
	margin := 0.05
	var delta float64
	if inside && med < margin {
		delta = margin - med
	} else if !inside && med > -margin {
		delta = -margin - med
	} else {
		return false
	}

	// z = sum(w_i v_i). Choose increments proportional to bilinear weights
	// so the requested correction is applied with minimum squared change.
	sumSq := 0.0
	for _, w := range weights {
		sumSq += w * w
	}
	if sumSq < 1e-8 {
		return false
	}
	buf := f.MSDF[ch]
	for k, idx := range indices {
		inc := 0.65 * delta * weights[k] / sumSq
		buf[idx] = clampDist(buf[idx]+inc, f.Range)
	}
	return true
}

func medianChannel(v [3]float64) (float64, int) {
	if v[0] <= v[1] && v[1] <= v[2] {
		return v[1], 1
	}
	if v[2] <= v[1] && v[1] <= v[0] {
		return v[1], 1
	}
	if v[1] <= v[0] && v[0] <= v[2] {
		return v[0], 0
	}
	if v[2] <= v[0] && v[0] <= v[1] {
		return v[0], 0
	}
	return v[2], 2
}
