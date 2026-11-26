// Package optconst provides wavelength-resolved optical constants for the
// materials used by the thin-film renderer, plus the CIE 1931 standard
// observer color matching functions and a D65 daylight illuminant.
//
// All tabulated data is embedded at build time (see data_gen.go), so the
// program keeps zero runtime/external dependencies.
package optconst

import "math"

// table is a monotonically increasing wavelength (nm) table of vectors.
type table = []struct {
	Lambda float64
	Vals   []float64
}

// interp returns the linearly interpolated vector at lambdaNM. Outside the
// tabulated range it clamps to the nearest endpoint sample.
func interp(t table, lambdaNM float64) []float64 {
	n := len(t)
	if lambdaNM <= t[0].Lambda {
		return append([]float64(nil), t[0].Vals...)
	}
	if lambdaNM >= t[n-1].Lambda {
		return append([]float64(nil), t[n-1].Vals...)
	}
	lo, hi := 0, n-1
	for hi-lo > 1 {
		mid := (lo + hi) / 2
		if t[mid].Lambda <= lambdaNM {
			lo = mid
		} else {
			hi = mid
		}
	}
	w := (lambdaNM - t[lo].Lambda) / (t[hi].Lambda - t[lo].Lambda)
	out := make([]float64, len(t[lo].Vals))
	for i := range out {
		out[i] = t[lo].Vals[i] + w*(t[hi].Vals[i]-t[lo].Vals[i])
	}
	return out
}

// FeNK returns the complex refractive index n + i*k of iron at lambdaNM,
// from Johnson & Christy (1974), linearly interpolated on the visible grid.
func FeNK(lambdaNM float64) complex128 {
	v := interp(feJohnsonTable, lambdaNM)
	return complex(v[0], v[1])
}

// CMF returns the CIE 1931 2-degree standard-observer matching functions
// (xBar, yBar, zBar) at lambdaNM, interpolated from the 5 nm table.
func CMF(lambdaNM float64) (xBar, yBar, zBar float64) {
	v := interp(cmf1931Table, lambdaNM)
	return v[0], v[1], v[2]
}

// d65XY are the CIE 1931 chromaticity coordinates of CIE standard
// illuminant D65 (correlated color temperature 6504 K).
const (
	d65X = 0.31271
	d65Y = 0.32902
)

// D65 returns the relative spectral power of CIE illuminant D65 at lambdaNM
// using the normative CIE daylight-illuminant reconstruction (CIE 15).
// S0, S1, S2 are evaluated from the published mean and component tables at
// 10 nm resolution; D65 = S0 + M1*S1 + M2*S2.
func D65(lambdaNM float64) float64 {
	s0, s1, s2 := daylightComponents(lambdaNM)
	denom := 0.0241 + 0.2562*d65X - 0.7341*d65Y
	m1 := (-1.3515 - 1.7703*d65X + 5.9114*d65Y) / denom
	m1 = math.Round(m1*1000) / 1000
	m2 := (0.0300 - 31.4424*d65X + 30.0717*d65Y) / denom
	m2 = math.Round(m2*1000) / 1000
	return s0 + m1*s1 + m2*s2
}

// daylightComponents interpolates the CIE daylight basis tables (300-830 nm,
// 10 nm grid). Rows outside the grid fall back to the nearest endpoint, which
// is harmless because the visible integration starts at 380 nm.
func daylightComponents(lambdaNM float64) (s0, s1, s2 float64) {
	const step = 10.0
	n := len(daylightBase)
	if lambdaNM <= daylightBase[0].lambda {
		first := daylightBase[0]
		return first.s0, first.s1, first.s2
	}
	if lambdaNM >= daylightBase[n-1].lambda {
		last := daylightBase[len(daylightBase)-1]
		return last.s0, last.s1, last.s2
	}
	lo, hi := 0, n-1
	for hi-lo > 1 {
		mid := (lo + hi) / 2
		if daylightBase[mid].lambda <= lambdaNM {
			lo = mid
		} else {
			hi = mid
		}
	}
	a, b := daylightBase[lo], daylightBase[hi]
	w := (lambdaNM - a.lambda) / step
	return a.s0 + w*(b.s0-a.s0),
		a.s1 + w*(b.s1-a.s1),
		a.s2 + w*(b.s2-a.s2)
}
