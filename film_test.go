package main

import (
	"math"
	"testing"
)

// The spectral peak must shift toward shorter wavelengths as the
// incidence angle grows — a wavelength-domain shift, not an RGB rephase.
func TestPeakShiftsToBlueWithTilt(t *testing.T) {
	layers := BeetleStack()
	prev := math.Inf(1)
	for deg := 0.0; deg <= 60; deg += 15 {
		cosT := math.Cos(deg * math.Pi / 180)
		s := FilmSpectrum(layers, 1.0, 1.50, cosT)
		peak := s.PeakLambda()
		if peak > prev {
			t.Errorf("tilt %.0f deg: peak moved to longer wavelength (%.0f > %.0f nm)", deg, peak, prev)
		}
		prev = peak
	}
	s0 := FilmSpectrum(layers, 1.0, 1.50, 1.0)
	s60 := FilmSpectrum(layers, 1.0, 1.50, 0.5)
	if s0.PeakLambda()-s60.PeakLambda() < 50 {
		t.Errorf("iridescent sweep too small: %.0f -> %.0f nm", s0.PeakLambda(), s60.PeakLambda())
	}
}

func TestReflectanceBounded(t *testing.T) {
	layers := BeetleStack()
	for _, cosT := range []float64{1.0, 0.7, 0.4, 0.1} {
		s := FilmSpectrum(layers, 1.0, 1.50, cosT)
		for i, v := range s {
			if v < 0 || v > 1 {
				t.Fatalf("cosT=%.2f lambda=%.0f: R=%v out of [0,1]", cosT, lambdaAt(i), v)
			}
		}
	}
}

// A flat unit spectrum must integrate to neutral white.
func TestFlatSpectrumIsWhite(t *testing.T) {
	var s Spectrum
	for i := range s {
		s[i] = 1
	}
	r, g, b := s.ToSRGB(1.0)
	if math.Abs(r-g) > 0.02 || math.Abs(g-b) > 0.02 {
		t.Errorf("flat spectrum not neutral: (%.3f, %.3f, %.3f)", r, g, b)
	}
}

// The LUT must agree with direct evaluation at the bin centers.
func TestLUTMatchesDirect(t *testing.T) {
	layers := BeetleStack()
	lut := NewFilmLUT(layers, 1.0, 1.50, 90)
	for _, cosT := range []float64{1.0, 0.5, 1.0 / 3.0} {
		direct := FilmSpectrum(layers, 1.0, 1.50, cosT)
		got := lut.Shade(cosT, 0)
		for i := range direct {
			if math.Abs(direct[i]-got[i]) > 1e-9 {
				t.Fatalf("cosT=%.3f lambda=%.0f: lut=%v direct=%v",
					cosT, lambdaAt(i), got[i], direct[i])
			}
		}
	}
}
