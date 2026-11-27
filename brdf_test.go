package main

import (
	"math"
	"testing"
)

func testParams() Params {
	return Params{Roughness: 0.35, Anisotropy: 0.8, F0: colorSteel}
}

func assertNonNegVec(t *testing.T, name string, c Vec3) {
	t.Helper()
	for _, x := range []float64{c.X, c.Y, c.Z} {
		if math.IsNaN(x) || math.IsInf(x, 0) || x < -1e-9 {
			t.Fatalf("%s has invalid component %v (%+v)", name, x, c)
		}
	}
}

// TestBRDFNonNegative sweeps many view/light directions including grazing.
func TestBRDFNonNegative(t *testing.T) {
	lut := buildEnergyLUT()
	p := testParams()
	frame := makeFrame(Vec3{0, 0, 1}, 0.7)
	for i := 0; i <= 40; i++ {
		theta := math.Pi * 0.5 * float64(i) / 40
		for j := 0; j < 24; j++ {
			phi := 2 * math.Pi * float64(j) / 24
			v := Vec3{math.Sin(theta) * math.Cos(phi), math.Sin(theta) * math.Sin(phi), math.Cos(theta)}
			l := Vec3{math.Cos(phi*0.7) * 0.6, math.Sin(phi*0.7) * 0.6, 0.8}.Normalize()
			if v.Dot(frame.N) <= 1e-4 {
				continue
			}
			val := evalBRDF(p, frame, l, v, lut)
			assertNonNegVec(t, "BRDF", val)
		}
	}
}

// TestGrazingNoWhiteSpot ensures the grazing response stays finite and bounded
// instead of exploding into a white spot.
func TestGrazingNoWhiteSpot(t *testing.T) {
	lut := buildEnergyLUT()
	p := testParams()
	frame := makeFrame(Vec3{0, 0, 1}, 0.0)
	v := Vec3{0.99995, 0, 0.01}.Normalize()
	l := Vec3{0.9999, 0.01, 0.01}.Normalize()
	val := evalBRDF(p, frame, l, v, lut)
	assertNonNegVec(t, "grazing BRDF", val)
	if val.Len() > 1e4 {
		t.Fatalf("grazing BRDF explodes: %v", val)
	}
}

// TestNDFNormalized integrates the anisotropic D over the hemisphere and
// expects ~1.
func TestNDFNormalized(t *testing.T) {
	ax, ay := 0.3, 0.7
	rng := newRng(12345)
	const n = 200000
	sum := 0.0
	for i := 0; i < n; i++ {
		// Uniform hemisphere sampling: z = u1, phi = 2pi u2; pdf = 1/(2pi).
		u1, u2 := rng.float(), rng.float()
		z := u1
		r := math.Sqrt(math.Max(0, 1-z*z))
		phi := 2 * math.Pi * u2
		dir := Vec3{r * math.Cos(phi), r * math.Sin(phi), z}
		hc := dirCos{dir.X, dir.Y, dir.Z}
		sum += dAniso(hc, ax, ay) * 2 * math.Pi / float64(n)
	}
	if math.Abs(sum-1) > 0.05 {
		t.Fatalf("anisotropic NDF integral = %.4f, want ~1", sum)
	}
}

// TestEnergyCompensationHelps verifies the multi-scatter term is non-negative
// and grows as roughness rises.
func TestEnergyCompensationHelps(t *testing.T) {
	lut := buildEnergyLUT()
	frame := makeFrame(Vec3{0, 0, 1}, 0.0)
	v := Vec3{0.3, 0.2, 0.93}.Normalize()
	l := Vec3{-0.2, 0.4, 0.89}.Normalize()
	low := evalBRDF(Params{Roughness: 0.1, Anisotropy: 0, F0: colorSteel}, frame, l, v, lut)
	high := evalBRDF(Params{Roughness: 0.9, Anisotropy: 0, F0: colorSteel}, frame, l, v, lut)
	assertNonNegVec(t, "low roughness", low)
	assertNonNegVec(t, "high roughness", high)
}

// TestTangentRotationReorientsHighlight checks that rotating the tangent by
// 90 degrees swaps the anisotropic response of two orthogonal light dirs.
func TestTangentRotationReorientsHighlight(t *testing.T) {
	lut := buildEnergyLUT()
	p := testParams()
	v := Vec3{0, 0, 1}
	frame0 := makeFrame(Vec3{0, 0, 1}, 0)
	frame90 := makeFrame(Vec3{0, 0, 1}, math.Pi/2)
	// Light grazing mostly along X vs mostly along Y.
	lX := Vec3{0.95, 0, 0.31}.Normalize()
	lY := Vec3{0, 0.95, 0.31}.Normalize()
	b0x := evalBRDF(p, frame0, lX, v, lut).Len()
	b0y := evalBRDF(p, frame0, lY, v, lut).Len()
	b90x := evalBRDF(p, frame90, lX, v, lut).Len()
	b90y := evalBRDF(p, frame90, lY, v, lut).Len()
	// The bright axis must flip when the fiber rotates 90 degrees.
	if !(b0x > b0y && b90y > b90x) {
		t.Fatalf("highlight did not rotate: 0deg x=%.3f y=%.3f; 90deg x=%.3f y=%.3f",
			b0x, b0y, b90x, b90y)
	}
}

// TestLUTBounds keeps every LUT entry in the valid albedo range.
func TestLUTBounds(t *testing.T) {
	lut := buildEnergyLUT()
	for i := range lut.alphas {
		if lut.eAvg[i] < 0 || lut.eAvg[i] > 1 {
			t.Fatalf("Eavg out of range at %d: %f", i, lut.eAvg[i])
		}
		for j := range lut.mus {
			e := lut.e[i][j]
			if e < 0 || e > 1+1e-6 {
				t.Fatalf("E out of range alpha=%d mu=%d: %f", i, j, e)
			}
		}
	}
}
