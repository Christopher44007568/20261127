package main

import "math"

// EnergyLUT stores the directional single-scattering albedo E(alpha, mu) of
// the isotropic Smith GGX BRDF, plus the hemispherical average Eavg(alpha).
// mu = cos(theta), the view-direction elevation cosine.
type EnergyLUT struct {
	na, nm int
	alphas []float64
	mus    []float64
	e      [][]float64 // e[alphaIndex][muIndex]
	eAvg   []float64
}

const lutSamples = 1024

func buildEnergyLUT() *EnergyLUT {
	na, nm := 64, 96
	lut := &EnergyLUT{
		na:     na,
		nm:     nm,
		alphas: make([]float64, na),
		mus:    make([]float64, nm),
		e:      make([][]float64, na),
		eAvg:   make([]float64, na),
	}
	for i := 0; i < na; i++ {
		// Linear alpha grid covers the full roughness range.
		lut.alphas[i] = mix(1e-3, 1, float64(i)/float64(na-1))
		lut.e[i] = make([]float64, nm)
	}
	for j := 0; j < nm; j++ {
		// Denser sampling near grazing angles where E changes fastest.
		t := float64(j) / float64(nm-1)
		lut.mus[j] = math.Max(1e-4, t*t)
	}

	rng := newRng(0xC0FFEE)
	for i, alpha := range lut.alphas {
		for j, mu := range lut.mus {
			lut.e[i][j] = integrateE(alpha, mu, rng, lutSamples)
		}
		// Eavg = 2 * integral_0^1 E(mu) mu dmu (trapezoid on mu grid).
		integral := 0.0
		for j := 1; j < nm; j++ {
			m0, m1 := lut.mus[j-1], lut.mus[j]
			v0, v1 := lut.e[i][j-1]*m0, lut.e[i][j]*m1
			integral += 0.5 * (v0 + v1) * (m1 - m0)
		}
		lut.eAvg[i] = clamp01(2 * integral)
	}
	return lut
}

// integrateE estimates E(alpha, mu) by Monte Carlo integration over the
// hemisphere of incoming directions using a cosine-weighted sampler.
func integrateE(alpha, mu float64, rng *Rng, samples int) float64 {
	// View direction with elevation cosine mu along +Z.
	v := Vec3{math.Sqrt(math.Max(0, 1-mu*mu)), 0, mu}
	a2 := alpha * alpha
	lambdaV := 0.5 * (math.Sqrt(1+a2*(1-mu*mu)/(mu*mu)) - 1)
	sum := 0.0
	for k := 0; k < samples; k++ {
		// GGX-importance-sampled half vector.
		u1, u2 := rng.float(), rng.float()
		thetaH := math.Atan(alpha * math.Sqrt(u1) / math.Sqrt(math.Max(0, 1-u1)))
		phiH := 2 * math.Pi * u2
		h := Vec3{
			math.Sin(thetaH) * math.Cos(phiH),
			math.Sin(thetaH) * math.Sin(phiH),
			math.Cos(thetaH),
		}
		// Reflect v around h to get the light direction.
		wi := h.Scale(2 * v.Dot(h)).Sub(v)
		noL := wi.Z
		if noL <= 1e-5 {
			continue
		}
		noH := clamp01(h.Z)
		voh := clamp01(v.Dot(h))
		den := math.Pi * math.Pow(noH*noH*(a2-1)+1, 2)
		D := a2 / math.Max(1e-30, den)
		lambdaL := 0.5 * (math.Sqrt(1+a2*(1-noL*noL)/(noL*noL)) - 1)
		G := 1 / (1 + lambdaV + lambdaL)
		// GGX sampling pdf = D * noH / (4 voh).
		pdf := D * noH / math.Max(1e-30, 4*voh)
		// E = integral f cosL dw; estimator f cosL / pdf.
		f := D * G / (4 * mu * noL)
		sum += f * noL / math.Max(1e-30, pdf)
	}
	return clamp01(sum / float64(samples))
}

// eval returns E(alpha, mu) with bilinear grid interpolation.
func (lut *EnergyLUT) eval(alpha, mu float64) float64 {
	mu = clampf(mu, 1e-4, 1)
	ai := gridIndex(lut.alphas, clampf(alpha, lut.alphas[0], lut.alphas[lut.na-1]))
	mi := gridIndex(lut.mus, mu)
	a0, a1 := ai, min(ai+1, lut.na-1)
	m0, m1 := mi, min(mi+1, lut.nm-1)
	ta := gridT(lut.alphas, alpha, a0)
	tm := gridT(lut.mus, mu, m0)
	e00 := lut.e[a0][m0]
	e10 := lut.e[a1][m0]
	e01 := lut.e[a0][m1]
	e11 := lut.e[a1][m1]
	e0 := e00 + (e10-e00)*ta
	e1 := e01 + (e11-e01)*ta
	return clamp01(e0 + (e1-e0)*tm)
}

// avg returns Eavg(alpha) with linear interpolation.
func (lut *EnergyLUT) avg(alpha float64) float64 {
	alpha = clampf(alpha, lut.alphas[0], lut.alphas[lut.na-1])
	i := gridIndex(lut.alphas, alpha)
	j := min(i+1, lut.na-1)
	t := gridT(lut.alphas, alpha, i)
	return clamp01(lut.eAvg[i] + (lut.eAvg[j]-lut.eAvg[i])*t)
}

func gridIndex(grid []float64, x float64) int {
	for i := 0; i < len(grid)-1; i++ {
		if x <= grid[i+1] {
			return i
		}
	}
	return len(grid) - 2
}

func gridT(grid []float64, x float64, i int) float64 {
	span := grid[i+1] - grid[i]
	if math.Abs(span) < 1e-12 {
		return 0
	}
	return clamp01((x - grid[i]) / span)
}
