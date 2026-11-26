package optconst

import (
	"math"
	"testing"
)

// TestD65Reference checks the CIE 15 daylight reconstruction (normalized to
// 100 at 560 nm) against the values produced by the normative formula with
// 3-decimal M1/M2 coefficients, using the CIE basis tables.
func TestD65Reference(t *testing.T) {
	want := map[float64]float64{
		380: 50.0, 400: 82.8, 450: 117.0, 500: 109.4,
		550: 104.0, 560: 100.0, 600: 90.0, 650: 80.1,
		700: 71.7, 750: 63.7, 780: 63.4,
	}
	scale := 100.0 / D65(560)
	for lambda, expected := range want {
		got := D65(lambda) * scale
		if math.Abs(got-expected) > 1.0 {
			t.Errorf("D65(%g nm) = %.1f, want %.1f", lambda, got, expected)
		}
	}
}

func TestFeInterpolation(t *testing.T) {
	n := FeNK(550)
	if real(n) < 2.5 || real(n) > 3.2 || imag(n) < 2.5 || imag(n) > 3.3 {
		t.Fatalf("unexpected Fe n,k at 550 nm: %v", n)
	}
	// Clamping beyond the table edges must not panic.
	_ = FeNK(200)
	_ = FeNK(2000)
}

func TestCMF(t *testing.T) {
	_, y555, _ := CMF(555)
	if y555 < 0.95 || y555 > 1.05 {
		t.Fatalf("yBar(555) = %v, want ~1.0", y555)
	}
}
