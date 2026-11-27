package furnace

import "math"

// GGXReflectionPDF returns the visible-normal reflection pdf wrt solid angle.
func GGXReflectionPDF(v, l Vec3, alpha float64) float64 {
	if v.Z <= 1e-7 || l.Z <= 1e-7 {
		return 0
	}
	h := v.Add(l).Normalize()
	vDotH := saturate(v.Dot(h))
	if vDotH <= 1e-7 || h.Z <= 0 {
		return 0
	}
	d := DistributionGGX(h.Z, alpha)
	g1 := smithG1GGX(v.Z, alpha)
	pdf := d * g1 / (4 * v.Z)
	if !finite(pdf) || pdf < 0 {
		return 0
	}
	return pdf
}

// IntegrateBRDFMIS integrates f(v,l)*NdotL using equal-count cosine and GGX
// visible-normal sampling with the balance heuristic. The samples argument is
// the total BRDF evaluation budget.
func IntegrateBRDFMIS(m Material, f func(v, l Vec3) Vec3, nDotV float64, samples int) Vec3 {
	if samples < 64 {
		samples = 64
	}
	n := samples / 2
	if n < 16 {
		n = 16
	}
	mu := clamp(nDotV, 1e-4, 1)
	v := outgoingDirection(mu)

	useGGX := m.Kind == RoughMetal || m.Kind == Clearcoat
	alpha := m.alpha()
	if m.Kind == Clearcoat {
		alpha = m.coatAlpha()
	}
	if !useGGX {
		return IntegrateHemisphere(f, mu, samples)
	}

	cosSum := V(0, 0, 0)
	ggxSum := V(0, 0, 0)

	for i := 0; i < n; i++ {
		u1, u2 := Hammersley2D(i, n)
		l := CosineHemisphere(u1, u2)
		pCos := l.Z / math.Pi
		pGGX := GGXReflectionPDF(v, l, alpha)
		w := 1.0
		if pGGX > 1e-12 {
			w = pCos / (pCos + pGGX)
		}
		brdf := f(v, l)
		if finiteVec(brdf) && brdf.MinChannel() >= 0 {
			cosSum = cosSum.Add(brdf.Mul(math.Pi * w))
		}
	}

	for i := 0; i < n; i++ {
		u1, u2 := Hammersley2D(i+17, n)
		h := sampleGGXVNDF(v, alpha, u1, u2)
		vDotH := v.Dot(h)
		if vDotH <= 1e-6 {
			continue
		}
		l := h.Mul(2 * vDotH).Sub(v).Normalize()
		if l.Z <= 1e-6 {
			continue
		}
		pGGX := GGXReflectionPDF(v, l, alpha)
		if pGGX <= 1e-12 {
			continue
		}
		pCos := l.Z / math.Pi
		w := pGGX / (pCos + pGGX)
		brdf := f(v, l)
		if !finiteVec(brdf) || brdf.MinChannel() < 0 {
			continue
		}
		estimate := brdf.Mul(l.Z / pGGX * w)
		ggxSum = ggxSum.Add(estimate)
	}

	return cosSum.Add(ggxSum).Div(float64(n))
}

// IntegrateNormalizedBRDFMIS white-furnace integrates the actual normalized
// BRDF closure rather than its component decomposition.
func IntegrateNormalizedBRDFMIS(m Material, table *EnergyTable, nDotV float64, samples int) Vec3 {
	return IntegrateBRDFMIS(m, func(v, l Vec3) Vec3 {
		return NormalizedBRDF(m, table, v, l)
	}, nDotV, samples)
}
