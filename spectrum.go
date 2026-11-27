package main

import "math"

const (
	lambdaMin = 400.0 // nanometres
	lambdaMax = 700.0
)

// WavelengthSamples returns deterministic stratified samples of the visible
// spectrum. The renderer uses the same physical samples for refraction,
// Beer-Lambert absorption and CIE XYZ integration.
func WavelengthSamples(count int) []float64 {
	if count < 3 {
		count = 3
	}
	out := make([]float64, count)
	step := (lambdaMax - lambdaMin) / float64(count)
	for i := range out {
		out[i] = lambdaMin + (float64(i)+0.5)*step
	}
	return out
}

// RefractiveIndex is a Cauchy dispersion model for a mildly dense crown/flint
// glass. lambda is in nanometres. Blue therefore has a slightly larger index
// than red, which fixes the sign of every dispersion result.
func RefractiveIndex(lambda float64) float64 {
	um := lambda / 1000.0
	return 1.600 + 0.0180/(um*um)
}

func gaussian(x, center, width float64) float64 {
	d := (x - center) / width
	return math.Exp(-d * d)
}

// AbsorptionCoefficient is in inverse world units. Two weak bands give the
// glass a pale sea-green body colour without making thick regions opaque.
func AbsorptionCoefficient(lambda float64) float64 {
	return 0.045 +
		0.205*gaussian(lambda, 655, 52) +
		0.115*gaussian(lambda, 424, 27) +
		0.030*gaussian(lambda, 505, 70)
}

func BeerTransmission(lambda, distance float64) float64 {
	if distance <= 0 {
		return 1
	}
	return math.Exp(-AbsorptionCoefficient(lambda) * distance)
}

// cieCMF returns analytic CIE 1931 2-degree colour-matching functions.
func cieCMF(lambda float64) (x, y, z float64) {
	t1 := (lambda - 442.0) * ternary(lambda < 442, 0.0624, 0.0374)
	t2 := (lambda - 599.8) * ternary(lambda < 599.8, 0.0264, 0.0323)
	t3 := (lambda - 501.1) * ternary(lambda < 501.1, 0.0490, 0.0382)
	x = 0.362*math.Exp(-0.5*t1*t1) +
		1.056*math.Exp(-0.5*t2*t2) -
		0.065*math.Exp(-0.5*t3*t3)

	u1 := (lambda - 568.8) * ternary(lambda < 568.8, 0.0213, 0.0247)
	u2 := (lambda - 530.9) * ternary(lambda < 530.9, 0.0613, 0.0322)
	y = 0.821*math.Exp(-0.5*u1*u1) + 0.286*math.Exp(-0.5*u2*u2)

	v1 := (lambda - 437.0) * ternary(lambda < 437, 0.0845, 0.0278)
	v2 := (lambda - 459.0) * ternary(lambda < 459, 0.0385, 0.0725)
	z = 1.217*math.Exp(-0.5*v1*v1) + 0.681*math.Exp(-0.5*v2*v2)
	if x < 0 {
		x = 0
	}
	return
}

func ternary(cond bool, a, b float64) float64 {
	if cond {
		return a
	}
	return b
}

func xyzToLinearRGB(c Vec3) Vec3 {
	return Vec3{
		X: 3.2404542*c.X - 1.5371385*c.Y - 0.4985314*c.Z,
		Y: -0.9692660*c.X + 1.8760108*c.Y + 0.0415560*c.Z,
		Z: 0.0556434*c.X - 0.2040259*c.Y + 1.0572252*c.Z,
	}
}

// WavelengthLinearRGB maps a spectral sample to display-referred linear RGB.
// A small gamut-compression term turns out-of-sRGB monochromatic stimuli into
// displayable colours without changing their hue ordering.
func WavelengthLinearRGB(lambda float64) Vec3 {
	x, y, z := cieCMF(lambda)
	rgb := xyzToLinearRGB(Vec3{x, y, z})
	minimum := math.Min(rgb.X, math.Min(rgb.Y, rgb.Z))
	if minimum < 0 {
		rgb = rgb.Add(v3(-minimum, -minimum, -minimum))
	}
	peak := math.Max(rgb.X, math.Max(rgb.Y, rgb.Z))
	if peak > 1e-6 {
		rgb = rgb.Mul(0.82 / peak)
	}
	return rgb
}

func absorptionRGB(distance float64, samples []float64) Vec3 {
	var xyz, white Vec3
	for _, lambda := range samples {
		cmfX, cmfY, cmfZ := cieCMF(lambda)
		t := BeerTransmission(lambda, distance)
		xyz = xyz.Add(v3(cmfX*t, cmfY*t, cmfZ*t))
		white = white.Add(v3(cmfX, cmfY, cmfZ))
	}
	if white.Y > 0 {
		xyz = xyz.Mul(1 / white.Y)
	}
	rgb := xyzToLinearRGB(xyz)
	rgb.X = clamp(rgb.X, 0, 1)
	rgb.Y = clamp(rgb.Y, 0, 1)
	rgb.Z = clamp(rgb.Z, 0, 1)
	return rgb
}

func srgbToLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func linearToSRGB(c float64) float64 {
	c = clamp01(c)
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1/2.4) - 0.055
}

func tonemap(c float64) float64 {
	// ACES-like compact curve; the simple rational form keeps highlights smooth.
	x := c * 1.12
	return clamp01((x * (2.51*x + 0.03)) / (x*(2.43*x+0.59) + 0.14))
}
