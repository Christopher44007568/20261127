package color

import (
	"math"
	"testing"
)

// TestWhitePoint: a perfect reflector must map to D65 white (Y=1) and
// neutral sRGB (255,255,255).
func TestWhitePoint(t *testing.T) {
	x, y, z := XYZ(func(float64) float64 { return 1 })
	if math.Abs(y-1) > 1e-9 {
		t.Fatalf("white Y = %.6f, want 1", y)
	}
	if math.Abs(x-1) > 1e-9 || math.Abs(z-1) > 1e-9 {
		t.Fatalf("white XYZ not normalized to 1: %.4f %.4f %.4f", x, y, z)
	}
	r, g, b := RGB(func(float64) float64 { return 1 })
	if r != 255 || g != 255 || b != 255 {
		t.Fatalf("white sRGB = (%d,%d,%d), want 255x3", r, g, b)
	}
}

// TestBlackPoint: zero reflectance maps to black.
func TestBlackPoint(t *testing.T) {
	r, g, b := RGB(func(float64) float64 { return 0 })
	if r != 0 || g != 0 || b != 0 {
		t.Fatalf("black sRGB = (%d,%d,%d), want 0", r, g, b)
	}
}

// TestWhitePointConstant: the hard-coded D65 white point must agree with the
// value obtained by directly integrating the embedded data.
func TestWhitePointConstant(t *testing.T) {
	var sx, sy, sz, wy float64
	for i := 0; i < sampleN; i++ {
		lambda := lambdaMin + float64(i)*stepNM
		w := d65Illuminant(lambda)
		xBar, yBar, zBar := cmf(lambda)
		sx += w * xBar
		sy += w * yBar
		sz += w * zBar
		wy += w * yBar
	}
	xn, zn := sx/sy, sz/sy
	if math.Abs(xn-d65White.x) > 0.005 || math.Abs(zn-d65White.z) > 0.005 {
		t.Fatalf("integrated white (%.4f,1,%.4f) != constant (%.4f,1,%.4f)",
			xn, zn, d65White.x, d65White.z)
	}
}

// TestSpectralHueOrder: long wavelengths look red, short look blue, and the
// middle of the band is brighter than either extreme (photopic weighting).
func TestSpectralHueOrder(t *testing.T) {
	band := func(center, width float64) Spectrum {
		return func(lambda float64) float64 {
			if math.Abs(lambda-center) <= width {
				return 1
			}
			return 0
		}
	}
	redR, redG, redB := RGB(band(650, 12))
	blueR, blueG, blueB := RGB(band(450, 12))
	if !(redR > redB && blueB > blueR) {
		t.Fatalf("hue order wrong: red=(%d,%d,%d) blue=(%d,%d,%d)",
			redR, redG, redB, blueR, blueG, blueB)
	}
	_, greenY, _ := XYZ(band(555, 12))
	_, blueY, _ := XYZ(band(450, 12))
	if greenY <= blueY {
		t.Fatalf("photopic weighting violated: green Y=%.3f blue Y=%.3f",
			greenY, blueY)
	}
}

// TestContinuity: a smoothly changing spectrum must not produce color jumps.
func TestContinuity(t *testing.T) {
	prev := -1.0
	for k := 0; k <= 200; k++ {
		level := 0.1 + 0.8*float64(k)/200.0
		spec := func(lv float64) Spectrum {
			return func(lambda float64) float64 {
				return lv * (0.8 + 0.2*math.Sin(lambda/100))
			}
		}(level)
		r, g, b := RGB(spec)
		lum := (float64(r) + float64(g) + float64(b)) / 3
		if prev >= 0 && math.Abs(lum-prev) > 3 {
			t.Fatalf("color jump at k=%d: %.2f -> %.2f", k, prev, lum)
		}
		prev = lum
	}
}
