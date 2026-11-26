package main

import (
	"math"
	"math/cmplx"
	"runtime"
	"sync"
)

// Layer is one isotropic film: refractive index N and thickness D in nm.
type Layer struct {
	N float64
	D float64
}

// Jewel beetle (Chrysochroa-like) elytron stack: alternating chitin and
// hydrated chitin quarter-wave layers over an absorbing cuticle substrate.
func BeetleStack() []Layer {
	const pairs = 8
	layers := make([]Layer, 0, 2*pairs)
	for i := 0; i < pairs; i++ {
		layers = append(layers,
			Layer{N: 1.56, D: 88},  // chitin
			Layer{N: 1.34, D: 102}, // hydrated chitin
		)
	}
	return layers
}

// admittance returns the tilted optical admittance for s- and p-polarization.
func admittances(n, cosT complex128) (etaS, etaP complex128) {
	return n * cosT, n / cosT
}

// reflectance computes |r|^2 for one polarization of the stack
// between incident medium n0 and substrate nSub.
func reflectancePol(layers []Layer, n0, nSub, lambda, cosT0 float64, sPol bool) float64 {
	sinT0 := math.Sqrt(math.Max(0, 1-cosT0*cosT0))
	pick := func(es, ep complex128) complex128 {
		if sPol {
			return es
		}
		return ep
	}

	// Characteristic matrix of the whole stack.
	m11, m12 := complex(1, 0), complex(0, 0)
	m21, m22 := complex(0, 0), complex(1, 0)
	for _, l := range layers {
		nj := complex(l.N, 0)
		sinTj := complex(n0*sinT0/l.N, 0)
		cosTj := cmplx.Sqrt(1 - sinTj*sinTj)
		delta := 2 * math.Pi * nj * complex(l.D, 0) * cosTj / complex(lambda, 0)
		eta := pick(admittances(nj, cosTj))
		cd, sd := cmplx.Cos(delta), cmplx.Sin(delta)
		a, b := cd, sd*complex(0, 1)/eta
		c, d := eta*complex(0, 1)*sd, cd
		m11, m12, m21, m22 =
			m11*a+m12*c, m11*b+m12*d,
			m21*a+m22*c, m21*b+m22*d
	}

	cosT0c := complex(cosT0, 0)
	eta0 := pick(admittances(complex(n0, 0), cosT0c))
	sinTs := complex(n0*sinT0/nSub, 0)
	cosTs := cmplx.Sqrt(1 - sinTs*sinTs)
	etaSub := pick(admittances(complex(nSub, 0), cosTs))

	B := m11 + m12*etaSub
	C := m21 + m22*etaSub
	r := (eta0*B - C) / (eta0*B + C)
	return real(r)*real(r) + imag(r)*imag(r)
}

// Reflectance averages s and p polarizations (unpolarized light).
func Reflectance(layers []Layer, n0, nSub, lambda, cosT0 float64) float64 {
	rs := reflectancePol(layers, n0, nSub, lambda, cosT0, true)
	rp := reflectancePol(layers, n0, nSub, lambda, cosT0, false)
	return 0.5 * (rs + rp)
}

// FilmSpectrum returns the angle-dependent reflectance spectrum of the stack.
func FilmSpectrum(layers []Layer, n0, nSub, cosT0 float64) Spectrum {
	var s Spectrum
	for i := 0; i < Samples; i++ {
		s[i] = Reflectance(layers, n0, nSub, lambdaAt(i), cosT0)
	}
	return s
}

// BaseScatterSpectrum is the diffuse pigment reflectance of the cuticle:
// dark with a broad green absorption remnant (melanin-like).
func BaseScatterSpectrum() Spectrum {
	var s Spectrum
	for i := 0; i < Samples; i++ {
		l := lambdaAt(i)
		green := 0.10 * math.Exp(-0.5*math.Pow((l-545)/70, 2))
		s[i] = 0.035 + green
	}
	return s
}

// Shade combines specular thin-film reflectance with Lambertian base
// scattering: film term modulated by Fresnel-like view weighting plus
// diffuse pigment term weighted by cos of the light incidence.
func Shade(layers []Layer, n0, nSub, cosFilm, cosLight float64) Spectrum {
	film := FilmSpectrum(layers, n0, nSub, cosFilm)
	base := BaseScatterSpectrum()
	cosLight = clamp01(cosLight)
	var out Spectrum
	for i := 0; i < Samples; i++ {
		out[i] = film[i] + base[i]*cosLight
	}
	return out
}

// FilmLUT caches film reflectance spectra over quantized cos(theta) so
// per-pixel shading reduces to a table lookup plus a diffuse add.
type FilmLUT struct {
	bins  int
	table []Spectrum
}

func NewFilmLUT(layers []Layer, n0, nSub float64, bins int) *FilmLUT {
	lut := &FilmLUT{bins: bins, table: make([]Spectrum, bins+1)}
	var wg sync.WaitGroup
	jobs := make(chan int, bins+1)
	for w := 0; w < runtime.NumCPU(); w++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				cosT := float64(i) / float64(bins)
				lut.table[i] = FilmSpectrum(layers, n0, nSub, cosT)
			}
		}()
	}
	for i := 0; i <= bins; i++ {
		jobs <- i
	}
	close(jobs)
	wg.Wait()
	return lut
}

// Shade looks up the film spectrum and adds the Lambertian base term.
func (l *FilmLUT) Shade(cosFilm, cosLight float64) Spectrum {
	idx := int(clamp01(cosFilm)*float64(l.bins) + 0.5)
	if idx > l.bins {
		idx = l.bins
	}
	base := BaseScatterSpectrum()
	cosLight = clamp01(cosLight)
	out := l.table[idx]
	for i := 0; i < Samples; i++ {
		out[i] += base[i] * cosLight
	}
	return out
}
