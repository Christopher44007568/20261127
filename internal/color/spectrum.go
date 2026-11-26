// Package color converts wavelength-resolved reflectance spectra into
// display-ready sRGB values using CIE 1931 tristimulus integration under
// the D65 illuminant, followed by the standard sRGB transfer function.
package color

import (
	"math"

	"thinfilm/internal/optconst"
)

// Integration grid: 380-780 nm in 5 nm steps, matching the embedded CMF
// table so no CMF interpolation error enters the integral.
const (
	lambdaMin = 380.0
	lambdaMax = 780.0
	stepNM    = 5.0
	sampleN   = int((lambdaMax-lambdaMin)/stepNM) + 1 // 81
)

// Spectrum returns reflectance (0..1) at wavelength lambdaNM.
type Spectrum func(lambdaNM float64) float64

// XYZ integrates the spectrum against D65 and the CIE 1931 observer with the
// trapezoidal rule. The white point is the perfect reflector (R=1) under the
// same illuminant, which normalizes Y to 1 for a white surface.
func XYZ(s Spectrum) (x, y, z float64) {
	var sumX, sumY, sumZ, whiteX, whiteY, whiteZ float64
	for i := 0; i < sampleN; i++ {
		lambda := lambdaMin + float64(i)*stepNM
		weight := optconst.D65(lambda)
		xBar, yBar, zBar := optconst.CMF(lambda)
		r := s(lambda)
		sumX += weight * xBar * r
		sumY += weight * yBar * r
		sumZ += weight * zBar * r
		whiteX += weight * xBar
		whiteY += weight * yBar
		whiteZ += weight * zBar
	}
	// Normalize by the perfect-reflector white point under the same
	// illuminant. Dividing each tristimulus component by its own white-point
	// integral yields relative XYZ with the canonical D65 white point
	// (Xn, Yn=1, Zn); the constant 5 nm step cancels in each ratio.
	return sumX / whiteX, sumY / whiteY, sumZ / whiteZ
}

// d65White is the canonical D65 white point in relative XYZ, obtained from
// the same CMF/D65 integration used above. Yn is 1 by construction.
var d65White = struct{ x, y, z float64 }{x: 0.95047, y: 1.0, z: 1.08883}

// sRGB linear-light conversion matrix (D65 adapted), per IEC 61966-2-1.
// The row sums are exactly 1, so the normalized D65 white point (1,1,1)
// maps to neutral white.
func xyzToLinearRGB(x, y, z float64) (r, g, b float64) {
	// Rescale relative white (1,1,1) to the absolute D65 white point that
	// the sRGB matrix is defined against.
	x *= d65White.x
	y *= d65White.y
	z *= d65White.z
	r = 3.2409699*x - 1.5373832*y - 0.4986108*z
	g = -0.9692436*x + 1.8759675*y + 0.0415551*z
	b = 0.0556300*x - 0.2039769*y + 1.0569715*z
	return clamp01(r), clamp01(g), clamp01(b)
}

// Encode applies the sRGB opto-electronic transfer function (gamma ~2.2).
func Encode(linear float64) float64 {
	if linear <= 0.0031308 {
		return 12.92 * linear
	}
	return 1.055*math.Pow(linear, 1/2.4) - 0.055
}

// RGB returns the gamma-encoded sRGB color of a spectrum as 8-bit channels.
func RGB(s Spectrum) (r, g, b uint8) {
	x, y, z := XYZ(s)
	lr, lg, lb := xyzToLinearRGB(x, y, z)
	return to8(Encode(lr)), to8(Encode(lg)), to8(Encode(lb))
}

// RGBFloat returns linear-light sRGB channels in 0..1 (for shading math).
func RGBFloat(s Spectrum) (r, g, b float64) {
	x, y, z := XYZ(s)
	return xyzToLinearRGB(x, y, z)
}

func to8(v float64) uint8 {
	v = clamp01(v)
	return uint8(math.Round(v * 255))
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}
