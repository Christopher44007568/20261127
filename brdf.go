package main

import "math"

// Params describes an anisotropic metallic material.
// Roughness is the linear roughness (0..1), anisotropy in (-1..1):
// positive makes the tangent (fiber) direction smoother than bitangent.
type Params struct {
	Roughness  float64
	Anisotropy float64
	F0         Vec3
}

// alphaAxes converts roughness/anisotropy into GGX alphas.
// ax = alpha along tangent T, ay = alpha along bitangent B.
func (p Params) alphaAxes() (ax, ay, aeff float64) {
	r := clampf(p.Roughness, 0.045, 1)
	a := r * r
	aniso := clampf(p.Anisotropy, -0.98, 0.98)
	ax = a * (1 - aniso)
	ay = a * (1 + aniso)
	ax = math.Max(ax, 1e-4)
	ay = math.Max(ay, 1e-4)
	// Isotropic-equivalent roughness drives the energy LUT.
	aeff = math.Sqrt(ax * ay)
	aeff = clampf(aeff, 1e-4, 1)
	return ax, ay, aeff
}

// lambdaAniso is Smith's Lambda for the anisotropic GGX microfacet profile
// (Burley 2012 / Kulla-Conty 2017).
func lambdaAniso(vt, vb, vn, ax, ay float64) float64 {
	if vn <= 0 {
		return 0
	}
	// Smith Lambda for anisotropic GGX (Burley 2012):
	// Lambda = 0.5 * (sqrt(1 + (ax^2 vt^2 + ay^2 vb^2)/vn^2) - 1)
	tt := (ax * vt) * (ax * vt)
	bb := (ay * vb) * (ay * vb)
	s := math.Sqrt(1 + (tt+bb)/(vn*vn))
	return 0.5 * (s - 1)
}

// g2Aniso is the joint masking-shadowing Smith term for anisotropic GGX.
func g2Aniso(l, v dirCos, ax, ay float64) float64 {
	lv := lambdaAniso(l.td, l.bd, l.nd, ax, ay)
	vv := lambdaAniso(v.td, v.bd, v.nd, ax, ay)
	return 1 / (1 + lv + vv)
}

// dAniso is the anisotropic GGX normal distribution function.
func dAniso(h dirCos, ax, ay float64) float64 {
	if h.nd <= 0 {
		return 0
	}
	// Burley 2012 anisotropic GGX, normalized over the hemisphere:
	// D = 1 / (pi ax ay (1 + (ht/ax)^2 + (hb/ay)^2)^2).
	tt := (h.td * h.td) / (ax * ax)
	bb := (h.bd * h.bd) / (ay * ay)
	k := 1 + tt + bb
	denom := math.Pi * ax * ay * k * k
	return 1 / math.Max(1e-30, denom)
}

func fresnelSchlick(cosTheta float64, f0 Vec3) Vec3 {
	t := math.Pow(math.Max(0, 1-cosTheta), 5)
	return Vec3{
		f0.X + (1-f0.X)*t,
		f0.Y + (1-f0.Y)*t,
		f0.Z + (1-f0.Z)*t,
	}
}

// dirCos stores a direction expressed in tangent space.
type dirCos struct {
	td, bd, nd float64
}

// evalBRDF returns the full energy-compensated BRDF for outgoing v and
// incoming l (both pointing away from the surface, unit length).
func evalBRDF(p Params, f Frame, lDir, vDir Vec3, lut *EnergyLUT) Vec3 {
	ax, ay, aeff := p.alphaAxes()
	// Clamp below an ~0.08 degree elevation to keep the 1/(noL noV) term
	// finite at grazing angles without altering normal-range appearance.
	const minCos = 1.5e-3
	noL := clampf(lDir.Dot(f.N), minCos, 1)
	noV := clampf(vDir.Dot(f.N), minCos, 1)
	h := lDir.Add(vDir)
	if h.Len2() < 1e-20 {
		return Vec3{}
	}
	h = h.Normalize()

	hCos := dirCos{h.Dot(f.T), h.Dot(f.B), h.Dot(f.N)}
	lCos := dirCos{lDir.Dot(f.T), lDir.Dot(f.B), noL}
	vCos := dirCos{vDir.Dot(f.T), vDir.Dot(f.B), noV}

	D := safeMax0(dAniso(hCos, ax, ay))
	G := safeMax0(g2Aniso(lCos, vCos, ax, ay))
	denom := 4 * noL * noV
	spec := D * G / math.Max(1e-30, denom)

	noH := clamp01(h.Dot(f.N))
	F := fresnelSchlick(noH, p.F0)

	// Single-scattering microfacet term.
	single := F.Scale(spec)

	// Multi-scattering energy compensation (Kulla-Conty 2017, as used by
	// Filament). The anisotropic surface is mapped to its isotropic-equivalent
	// alpha for the 2D directional-albedo LUT.
	eV := lut.eval(aeff, noV)
	eL := lut.eval(aeff, noL)
	eAvg := lut.avg(aeff)
	fAvg := p.F0.Scale(20.0 / 21.0).Add(Vec3{1.0 / 21, 1.0 / 21, 1.0 / 21})
	num := p.F0.Sub(fAvg.Scale(eAvg))
	base := math.Max(1e-6, (1-eAvg)*(1-eAvg))
	fmsCoef := Vec3{
		safeMax0(num.X) / base,
		safeMax0(num.Y) / base,
		safeMax0(num.Z) / base,
	}
	ms := fmsCoef.Scale((1 - eV) * (1 - eL))

	return Vec3{
		safeMax0(single.X) + safeMax0(ms.X),
		safeMax0(single.Y) + safeMax0(ms.Y),
		safeMax0(single.Z) + safeMax0(ms.Z),
	}
}
