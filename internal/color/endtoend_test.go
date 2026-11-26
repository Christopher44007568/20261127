package color

import (
	"math"
	"testing"

	"thinfilm/internal/thinfilm"
)

// TestThicknessRampNoColorJump drives the full pipeline (film physics ->
// spectral integration -> sRGB) over the 200-800 nm range from the brief and
// asserts the displayed color migrates smoothly rather than jumping.
func TestThicknessRampNoColorJump(t *testing.T) {
	var prev [3]float64
	havePrev := false
	for d := 200.0; d <= 800.0; d += 1.0 {
		stack := thinfilm.NewStack(1.5, d)
		r, g, b := RGBFloat(func(lambda float64) float64 {
			return stack.Reflectance(lambda, 0)
		})
		cur := [3]float64{r, g, b}
		if havePrev {
			for ch := 0; ch < 3; ch++ {
				if math.Abs(cur[ch]-prev[ch]) > 0.03 {
					t.Fatalf("channel %d jumps at d=%.0f: %.4f -> %.4f",
						ch, d, prev[ch], cur[ch])
				}
			}
		}
		prev, havePrev = cur, true
	}
}

// TestZeroFilmIsNeutral: the zero-thickness reference must be achromatic
// (no film-introduced hue). Iron has a weak warm tint from its own n,k
// dispersion, so the channels are allowed a small spread.
func TestZeroFilmIsNeutral(t *testing.T) {
	bare := thinfilm.NewStack(1.5, 0)
	r, g, b := RGBFloat(func(lambda float64) float64 {
		return bare.Reflectance(lambda, 0)
	})
	spread := math.Max(r, math.Max(g, b)) - math.Min(r, math.Min(g, b))
	if spread > 0.05 {
		t.Fatalf("bare metal not neutral: rgb=(%.4f,%.4f,%.4f) spread=%.4f",
			r, g, b, spread)
	}
}

// TestAngledColorContinuity exercises the same color pipeline through a full
// normal-to-grazing angular sweep on a coated stack.
func TestAngledColorContinuity(t *testing.T) {
	stack := thinfilm.NewStack(1.5, 500)
	var prev [3]float64
	havePrev := false
	for theta := 0.5; theta <= 85.0; theta += 0.5 {
		r, g, b := RGBFloat(func(lambda float64) float64 {
			return stack.Reflectance(lambda, theta)
		})
		cur := [3]float64{r, g, b}
		if havePrev {
			for ch := 0; ch < 3; ch++ {
				if math.Abs(cur[ch]-prev[ch]) > 0.05 {
					t.Fatalf("angular color jump ch=%d at %.1f deg",
						ch, theta)
				}
			}
		}
		prev, havePrev = cur, true
	}
}
