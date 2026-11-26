// Package thinfilm computes coherent reflection from an air / dielectric
// film / absorbing metal stack, for s and p polarizations and arbitrary
// incidence angle.
//
// The film stack is solved with the Abeles characteristic-matrix formalism,
// which stays numerically stable for every thickness and angle, including
// the two important limiting cases:
//
//   - zero film thickness: the film matrix is the identity and the result
//     degenerates exactly to bare-substrate reflection (no color);
//   - grazing incidence: admittances vanish/explode only through cos(theta)
//     terms, so the angle is clamped just shy of 90 degrees and the
//     reflectance converges continuously to 1.
package thinfilm

import (
	"math"

	"thinfilm/internal/optconst"
)

// Stack describes the air (n0=1) / film / metal geometry.
type Stack struct {
	// FilmN is the (real, non-dispersive) film refractive index.
	FilmN float64
	// ThicknessNM is the film thickness d in nanometers.
	ThicknessNM float64
	// metalNK resolves the substrate's complex index at wavelength lambdaNM.
	metalNK func(lambdaNM float64) complex128
}

// NewStack builds a film on an iron substrate. Pass thickness 0 for the
// uncoated reference.
func NewStack(filmN, thicknessNM float64) Stack {
	return Stack{
		FilmN:       filmN,
		ThicknessNM: thicknessNM,
		metalNK:     optconst.FeNK,
	}
}

// c2x2 is a 2x2 complex matrix in row-major order.
type c2x2 [4]complex128

func matMul(a, b c2x2) c2x2 {
	return c2x2{
		a[0]*b[0] + a[1]*b[2], a[0]*b[1] + a[1]*b[3],
		a[2]*b[0] + a[3]*b[2], a[2]*b[1] + a[3]*b[3],
	}
}

// filmMatrix returns the characteristic matrix of the dielectric layer.
// delta = 2*pi*n*d*cos(theta)/lambda is the layer phase thickness; eta is
// the optical admittance for the requested polarization.
func filmMatrix(n, cosTheta, eta complex128, delta complex128) c2x2 {
	cosD := complex(math.Cos(real(delta)), 0)
	sinD := complex(math.Sin(real(delta)), 0)
	iSin := complex(0, real(sinD))
	return c2x2{
		cosD, iSin / eta,
		iSin * eta, cosD,
	}
}

// Reflectance returns the unpolarized intensity reflectance of the stack at
// wavelength lambdaNM for incidence angle thetaDeg measured from the normal.
func (s Stack) Reflectance(lambdaNM, thetaDeg float64) float64 {
	theta := clampTheta(thetaDeg)
	n0 := 1.0
	nFilm := complex(s.FilmN, 0)
	nSub := s.metalNK(lambdaNM)

	// Propagation angles from Snell's law with complex refractive indices.
	sin0 := math.Sin(theta)
	cos0 := math.Cos(theta)
	sinFilm2 := complex((sin0*sin0)/(s.FilmN*s.FilmN), 0)
	cosFilm := csqrt(1 - sinFilm2)
	sinSub2 := (complex(sin0, 0) / nSub) * (complex(sin0, 0) / nSub)
	cosSub := csqrt(1 - sinSub2)

	rs := reflectPolarization(complex(n0, 0), nFilm, nSub,
		complex(cos0, 0), cosFilm, cosSub, s.ThicknessNM, lambdaNM, false)
	rp := reflectPolarization(complex(n0, 0), nFilm, nSub,
		complex(cos0, 0), cosFilm, cosSub, s.ThicknessNM, lambdaNM, true)
	return 0.5 * (rs + rp)
}

// reflectPolarization applies the transfer-matrix boundary conditions for
// either s (pWave=false) or p (pWave=true) polarization.
func reflectPolarization(n0, nFilm, nSub, cos0, cosFilm, cosSub complex128,
	thicknessNM, lambdaNM float64, pWave bool) float64 {

	eta0 := admittance(n0, cos0, pWave)
	etaFilm := admittance(nFilm, cosFilm, pWave)
	etaSub := admittance(nSub, cosSub, pWave)

	// Phase thickness delta = 2*pi*n*d*cos(theta)/lambda.
	delta := 2 * math.Pi * nFilm * complex(thicknessNM/lambdaNM, 0) * cosFilm
	m := filmMatrix(nFilm, cosFilm, etaFilm, delta)

	// Input admittance Y = (m11*etaS + m12) / (m21*etaS + m22).
	num := m[0]*etaSub + m[1]
	den := m[2]*etaSub + m[3]
	yIn := num / den

	r := (eta0 - yIn) / (eta0 + yIn)
	return real(r * conj(r))
}

// admittance is the tilted optical admittance: n*cos(theta) for s and
// n/cos(theta) for p.
func admittance(n, cosTheta complex128, pWave bool) complex128 {
	if pWave {
		return n / cosTheta
	}
	return n * cosTheta
}

func csqrt(z complex128) complex128 {
	// math.Sqrt on complex numbers requires the cmplx package; keep the
	// explicit branch (non-negative real part) to avoid an extra import.
	re, im := real(z), imag(z)
	mag := math.Hypot(re, im)
	sr := math.Sqrt((mag + re) / 2)
	var si float64
	if mag-re > 0 {
		si = math.Sqrt((mag - re) / 2)
	}
	if im < 0 {
		si = -si
	}
	return complex(sr, si)
}

func conj(z complex128) complex128 { return complex(real(z), -imag(z)) }

// maxTheta keeps cos(theta) finite; 89.9 degrees is visually grazing while
// avoiding 0/0 at the exact 90-degree singularity.
const maxTheta = 89.9 * math.Pi / 180

func clampTheta(thetaDeg float64) float64 {
	t := thetaDeg * math.Pi / 180
	if t < 0 {
		t = -t
	}
	if t > maxTheta {
		t = maxTheta
	}
	return t
}
