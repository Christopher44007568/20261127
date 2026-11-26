package main

import "math"

// Visible spectrum sampled at 5 nm steps.
const (
	LambdaMin  = 380.0
	LambdaMax  = 780.0
	LambdaStep = 5.0
	Samples    = int((LambdaMax-LambdaMin)/LambdaStep) + 1 // 81
)

// Spectrum holds spectral power/reflectance at 5 nm intervals.
type Spectrum [Samples]float64

func lambdaAt(i int) float64 { return LambdaMin + float64(i)*LambdaStep }

// piecewiseGauss is the asymmetric Gaussian used by the analytic
// CIE 1931 color-matching approximation (Wyman, Sloan, Shirley 2013).
func piecewiseGauss(x, mu, sigma1, sigma2 float64) float64 {
	s := sigma1
	if x >= mu {
		s = sigma2
	}
	t := (x - mu) / s
	return math.Exp(-0.5 * t * t)
}

// cieXYZ returns the CIE 1931 2-degree color matching functions at lambda (nm).
func cieXYZ(lambda float64) (x, y, z float64) {
	x = 1.056*piecewiseGauss(lambda, 599.8, 37.9, 31.0) +
		0.362*piecewiseGauss(lambda, 442.0, 16.0, 26.7) -
		0.065*piecewiseGauss(lambda, 501.1, 20.4, 26.2)
	y = 0.821*piecewiseGauss(lambda, 568.8, 46.9, 40.5) +
		0.286*piecewiseGauss(lambda, 530.9, 16.3, 31.1)
	z = 1.217*piecewiseGauss(lambda, 437.0, 11.8, 36.0) +
		0.681*piecewiseGauss(lambda, 459.0, 26.0, 13.8)
	return
}

// ToXYZ integrates the spectrum against the CMFs (equal-energy illuminant,
// white-point normalized so a flat unit spectrum yields X = Y = Z = 1).
func (s *Spectrum) ToXYZ() (X, Y, Z float64) {
	var nx, ny, nz float64
	for i := 0; i < Samples; i++ {
		cx, cy, cz := cieXYZ(lambdaAt(i))
		X += s[i] * cx
		Y += s[i] * cy
		Z += s[i] * cz
		nx += cx
		ny += cy
		nz += cz
	}
	return X / nx, Y / ny, Z / nz
}

// PeakLambda returns the wavelength of the spectral maximum.
func (s *Spectrum) PeakLambda() float64 {
	best, bi := s[0], 0
	for i := 1; i < Samples; i++ {
		if s[i] > best {
			best, bi = s[i], i
		}
	}
	return lambdaAt(bi)
}

// XYZToLinearRGB applies the sRGB D65 conversion matrix.
func XYZToLinearRGB(X, Y, Z float64) (r, g, b float64) {
	r = 3.2406*X - 1.5372*Y - 0.4986*Z
	g = -0.9689*X + 1.8758*Y + 0.0415*Z
	b = 0.0557*X - 0.2040*Y + 1.0570*Z
	return
}

func srgbGamma(c float64) float64 {
	if c <= 0.0031308 {
		return 12.92 * c
	}
	return 1.055*math.Pow(c, 1.0/2.4) - 0.055
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

// Linear RGB of the equal-energy white spectrum, used as the display
// white point so a flat reflectance renders as neutral white.
var whiteR, whiteG, whiteB = func() (r, g, b float64) {
	var w Spectrum
	for i := range w {
		w[i] = 1
	}
	X, Y, Z := w.ToXYZ()
	return XYZToLinearRGB(X, Y, Z)
}()

// ToSRGB converts the spectrum to gamma-encoded sRGB in [0,1].
// scale is an exposure factor applied in linear space.
func (s *Spectrum) ToSRGB(scale float64) (r, g, b float64) {
	X, Y, Z := s.ToXYZ()
	lr, lg, lb := XYZToLinearRGB(X*scale, Y*scale, Z*scale)
	lr, lg, lb = lr/whiteR, lg/whiteG, lb/whiteB
	return clamp01(srgbGamma(clamp01(lr))),
		clamp01(srgbGamma(clamp01(lg))),
		clamp01(srgbGamma(clamp01(lb)))
}

// Hue returns the HSV hue in degrees of the spectrum's sRGB color.
func (s *Spectrum) Hue() float64 {
	r, g, b := s.ToSRGB(1.0)
	max := math.Max(r, math.Max(g, b))
	min := math.Min(r, math.Min(g, b))
	d := max - min
	if d < 1e-9 {
		return 0
	}
	var h float64
	switch max {
	case r:
		h = 60 * math.Mod((g-b)/d, 6)
	case g:
		h = 60 * ((b-r)/d + 2)
	default:
		h = 60 * ((r-g)/d + 4)
	}
	if h < 0 {
		h += 360
	}
	return h
}
