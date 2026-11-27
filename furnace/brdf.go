package furnace

import "math"

// LambertBRDF is the perfectly diffuse lobe.
func LambertBRDF(albedo Vec3) Vec3 {
	return albedo.Mul(1 / math.Pi)
}

// FresnelSchlick returns the scalar Schlick Fresnel approximation.
func FresnelSchlick(cosTheta, f0 float64) float64 {
	return f0 + (1-f0)*pow5(1-cosTheta)
}

// FresnelSchlickV is the colored Schlick Fresnel approximation.
func FresnelSchlickV(cosTheta float64, f0 Vec3) Vec3 {
	u := pow5(1 - saturate(cosTheta))
	return f0.Add(V(1, 1, 1).Sub(f0).Mul(u))
}

// DistributionGGX is the isotropic Trowbridge-Reitz normal distribution.
func DistributionGGX(nDotH, alpha float64) float64 {
	if nDotH <= 0 {
		return 0
	}
	a2 := alpha * alpha
	d := nDotH*nDotH*(a2-1) + 1
	return a2 / (math.Pi * d * d)
}

func lambdaGGX(nDotW, alpha float64) float64 {
	nDotW = saturate(nDotW)
	if nDotW < 1e-7 {
		return 25 // Large but finite grazing masking.
	}
	a2 := alpha * alpha
	tan2 := (1 - nDotW*nDotW) / (nDotW * nDotW)
	return 0.5 * (math.Sqrt(1+a2*tan2) - 1)
}

// SmithG2GGX is the height-correlated Smith shadowing/masking term.
func SmithG2GGX(nDotV, nDotL, alpha float64) float64 {
	lv := lambdaGGX(nDotV, alpha)
	ll := lambdaGGX(nDotL, alpha)
	return saturate(1 / (1 + lv + ll))
}

func smithG1GGX(nDotW, alpha float64) float64 {
	return saturate(1 / (1 + lambdaGGX(nDotW, alpha)))
}

// SpecularGGX evaluates one-scattering Cook-Torrance GGX.
// V and L point away from the surface in a local frame whose normal is +Z.
func SpecularGGX(f0 Vec3, roughness float64, v, l Vec3) Vec3 {
	nDotV := v.Z
	nDotL := l.Z
	if nDotV <= 1e-6 || nDotL <= 1e-6 {
		return V(0, 0, 0)
	}
	h := v.Add(l).Normalize()
	nDotH := h.Z
	vDotH := saturate(v.Dot(h))
	if nDotH <= 0 || vDotH <= 0 {
		return V(0, 0, 0)
	}
	alpha := clamp(clamp(roughness, 0.035, 1)*clamp(roughness, 0.035, 1), 0.0015, 1)
	d := DistributionGGX(nDotH, alpha)
	g := SmithG2GGX(nDotV, nDotL, alpha)
	f := FresnelSchlickV(vDotH, f0)
	denom := 4 * nDotV * nDotL
	if denom < 1e-6 {
		return V(0, 0, 0)
	}
	return f.Mul(d * g / denom)
}

// ClearcoatBRDF is the unnormalized dielectric GGX top lobe. Its Fresnel is
// physically bounded, but RawBRDF intentionally adds the full diffuse base.
func ClearcoatBRDF(m Material, v, l Vec3) Vec3 {
	nDotV := v.Z
	nDotL := l.Z
	if m.Coat <= 0 || nDotV <= 1e-6 || nDotL <= 1e-6 {
		return V(0, 0, 0)
	}
	f0 := clamp(m.CoatF0, 0.02, 0.08)
	spec := SpecularGGX(V(f0, f0, f0), m.CoatRoughness, v, l)
	return spec.Mul(m.Coat)
}

// VelvetBRDF evaluates a reciprocal retroreflection plus symmetric grazing
// sheen. The lobe is intentionally artist-weighted; RawBRDF performs no
// hemisphere energy budget for it.
func VelvetBRDF(m Material, v, l Vec3) Vec3 {
	nDotV := saturate(v.Z)
	nDotL := saturate(l.Z)
	if m.Sheen <= 0 || nDotV <= 0 || nDotL <= 0 {
		return V(0, 0, 0)
	}

	rough := clamp(m.SheenRoughness, 0.08, 1)
	retroPower := lerp(24, 3.5, rough)
	retro := math.Pow(saturate(v.Dot(l)), retroPower)
	retroWeight := m.Sheen * m.SheenRetro * (0.35 + 0.65*(1-rough))

	grazingPower := lerp(1.4, 3.2, rough)
	edgeV := math.Pow(1-nDotV, grazingPower)
	edgeL := math.Pow(1-nDotL, grazingPower)
	grazing := math.Sqrt(edgeV * edgeL)
	grazingWeight := m.Sheen * m.SheenGrazingGain * 2.35

	shape := retroWeight*retro + grazingWeight*grazing
	return m.SheenTint.Mul(shape)
}

// RawBRDF is the naive legal-parameter combination. It may reflect more or
// less than one unit of energy for a white furnace.
func RawBRDF(m Material, v, l Vec3) Vec3 {
	v = v.Normalize()
	l = l.Normalize()
	if v.Z <= 1e-7 || l.Z <= 1e-7 {
		return V(0, 0, 0)
	}

	out := V(0, 0, 0)

	metalWeight := saturate(m.Metallic)
	diffuseWeight := 1 - metalWeight
	if m.Kind == Clearcoat || m.Kind == Velvet {
		diffuseWeight = 1
	}
	if diffuseWeight > 0 {
		out = out.Add(LambertBRDF(m.Albedo).Mul(diffuseWeight))
	}

	switch m.Kind {
	case RoughMetal:
		out = out.Add(SpecularGGX(m.f0(), m.Roughness, v, l))
	case Clearcoat:
		// Deliberate raw combination: no (1-F) transmission attenuation and
		// no directional energy correction.
		out = out.Add(ClearcoatBRDF(m, v, l))
	case Velvet:
		out = out.Add(VelvetBRDF(m, v, l))
	}
	return out
}
