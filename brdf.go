package main

import "math"

func ggxAlpha(roughness float64) float64 {
	r := clampFloat(roughness, MinRoughness, 1)
	return clampFloat(r*r, 0.0018, 1)
}

func distributionGGX(nDotH, alpha float64) float64 {
	if nDotH <= 0 {
		return 0
	}
	a2 := alpha * alpha
	d := nDotH*nDotH*(a2-1) + 1
	return a2 / (math.Pi * d * d)
}

func lambdaGGX(nDotW, alpha float64) float64 {
	nDotW = clamp01(nDotW)
	if nDotW < 1e-7 {
		return 25
	}
	a2 := alpha * alpha
	tan2 := (1 - nDotW*nDotW) / (nDotW * nDotW)
	return 0.5 * (math.Sqrt(1+a2*tan2) - 1)
}

func smithG2GGX(nDotV, nDotL, alpha float64) float64 {
	g := 1 + lambdaGGX(nDotV, alpha) + lambdaGGX(nDotL, alpha)
	return 1 / g
}

func fresnelSchlick(cosTheta float64, f0 Vec3) Vec3 {
	u := math.Pow(1-clamp01(cosTheta), 5)
	return f0.Add(V(1, 1, 1).Sub(f0).Mul(u))
}

func specularGGX(f0 Vec3, roughness float64, n, v, l Vec3) Vec3 {
	nDotV := clamp01(n.Dot(v))
	nDotL := clamp01(n.Dot(l))
	if nDotV <= 1e-5 || nDotL <= 1e-5 {
		return V(0, 0, 0)
	}
	h := v.Add(l).Normalize()
	nDotH := clamp01(n.Dot(h))
	vDotH := clamp01(v.Dot(h))
	if nDotH <= 0 || vDotH <= 0 {
		return V(0, 0, 0)
	}
	alpha := ggxAlpha(roughness)
	d := distributionGGX(nDotH, alpha)
	g2 := smithG2GGX(nDotV, nDotL, alpha)
	f := fresnelSchlick(vDotH, f0)
	return f.Mul(d * g2 / (4 * nDotV * nDotL))
}

func hammersley(i, n int) (float64, float64) {
	u1 := (float64(i) + 0.5) / float64(n)
	bits := uint32(i + 1)
	bits = ((bits >> 1) & 0x55555555) | ((bits & 0x55555555) << 1)
	bits = ((bits >> 2) & 0x33333333) | ((bits & 0x33333333) << 2)
	bits = ((bits >> 4) & 0x0f0f0f0f) | ((bits & 0x0f0f0f0f) << 4)
	bits = ((bits >> 8) & 0x00ff00ff) | ((bits & 0x00ff00ff) << 8)
	bits = (bits >> 16) | (bits << 16)
	return u1, float64(bits) / 4294967296.0
}

func sampleGGXVNDF(v Vec3, alpha, u1, u2 float64) Vec3 {
	vh := V(alpha*v.X, alpha*v.Y, v.Z).Normalize()
	lengthSqr := vh.X*vh.X + vh.Y*vh.Y
	var t1 Vec3
	if lengthSqr > 1e-12 {
		inv := 1 / math.Sqrt(lengthSqr)
		t1 = V(-vh.Y*inv, vh.X*inv, 0)
	} else {
		t1 = V(1, 0, 0)
	}
	t2 := vh.Cross(t1).Normalize()

	r := math.Sqrt(math.Max(0, u1))
	phi := 2 * math.Pi * u2
	x := r * math.Cos(phi)
	y := r * math.Sin(phi)
	s := 0.5 * (1 + vh.Z)
	y = (1-s)*math.Sqrt(math.Max(0, 1-x*x)) + s*y

	oneMinus := math.Max(0, 1-x*x-y*y)
	h := t1.Mul(x).Add(t2.Mul(y)).Add(vh.Mul(math.Sqrt(oneMinus)))
	return V(alpha*h.X, alpha*h.Y, h.Z).Normalize()
}

// EnergyLUT stores white-Furnace single-scattering directional albedo E(mu,r).
type EnergyLUT struct {
	MuBins    int
	RoughBins int
	Samples   int
	Mu        []float64
	White     [][]float64
	Average   []float64
}

func BuildEnergyLUT(muBins, roughBins, samples int) *EnergyLUT {
	if muBins < 4 {
		muBins = 4
	}
	if roughBins < 4 {
		roughBins = 4
	}
	if samples < 32 {
		samples = 32
	}
	lut := &EnergyLUT{MuBins: muBins, RoughBins: roughBins, Samples: samples,
		Mu: make([]float64, muBins), White: make([][]float64, roughBins),
		Average: make([]float64, roughBins)}
	for i := 0; i < muBins; i++ {
		f := float64(i) / float64(muBins-1)
		lut.Mu[i] = 0.004 + 0.996*f
	}
	for r := 0; r < roughBins; r++ {
		rough := MinRoughness + (1-MinRoughness)*float64(r)/float64(roughBins-1)
		lut.White[r] = make([]float64, muBins)
		var avgWeight, avgSum float64
		for m, mu := range lut.Mu {
			e := directionalWhiteEnergy(mu, ggxAlpha(rough), samples)
			lut.White[r][m] = clampFloat(e, 0, 1)
			weight := 2 * mu
			avgSum += lut.White[r][m] * weight
			avgWeight += weight
		}
		lut.Average[r] = avgSum / avgWeight
	}
	return lut
}

func directionalWhiteEnergy(mu, alpha float64, samples int) float64 {
	v := V(math.Sqrt(math.Max(0, 1-mu*mu)), 0, mu)
	g1V := 1 / (1 + lambdaGGX(mu, alpha))
	sum := 0.0
	for i := 0; i < samples; i++ {
		u1, u2 := hammersley(i, samples)
		h := sampleGGXVNDF(v, alpha, u1, u2)
		vDotH := v.Dot(h)
		if vDotH <= 1e-6 {
			continue
		}
		l := h.Mul(2 * vDotH).Sub(v).Normalize()
		if l.Z <= 1e-6 {
			continue
		}
		weight := smithG2GGX(mu, l.Z, alpha) / g1V
		if math.IsNaN(weight) || math.IsInf(weight, 0) || weight < 0 {
			continue
		}
		sum += clampFloat(weight, 0, 4)
	}
	return sum / float64(samples)
}

func (lut *EnergyLUT) Energy(roughness, nDotV float64) float64 {
	if lut == nil {
		return 0.85
	}
	rf := clamp01((clampFloat(roughness, MinRoughness, 1)-MinRoughness)/(1-MinRoughness)) * float64(lut.RoughBins-1)
	r0 := int(math.Floor(rf))
	r1 := r0 + 1
	rt := rf - math.Floor(rf)
	if r1 >= lut.RoughBins {
		r1 = lut.RoughBins - 1
	}
	mu := clampFloat(nDotV, lut.Mu[0], lut.Mu[len(lut.Mu)-1])
	mf := (mu - lut.Mu[0]) / (lut.Mu[len(lut.Mu)-1] - lut.Mu[0]) * float64(lut.MuBins-1)
	m0 := int(math.Floor(mf))
	m1 := m0 + 1
	mt := mf - math.Floor(mf)
	if m1 >= lut.MuBins {
		m1 = lut.MuBins - 1
	}
	a := lerpFloat(lut.White[r0][m0], lut.White[r0][m1], mt)
	b := lerpFloat(lut.White[r1][m0], lut.White[r1][m1], mt)
	return lerpFloat(a, b, rt)
}

func (lut *EnergyLUT) AverageEnergy(roughness float64) float64 {
	if lut == nil {
		return 0.8
	}
	rf := clamp01((clampFloat(roughness, MinRoughness, 1)-MinRoughness)/(1-MinRoughness)) * float64(lut.RoughBins-1)
	r0 := int(math.Floor(rf))
	r1 := r0 + 1
	rt := rf - math.Floor(rf)
	if r1 >= lut.RoughBins {
		r1 = lut.RoughBins - 1
	}
	return lerpFloat(lut.Average[r0], lut.Average[r1], rt)
}

// MultiScatterBRDF returns Kulla-Conty style energy compensation. For white
// F0 it completes the directional energy to one; colored F0 uses the same
// cosine-weighted Schlick average as the split-sum BRDF LUT.
func (lut *EnergyLUT) MultiScatterBRDF(f0 Vec3, roughness, nDotV, nDotL float64) Vec3 {
	eo := lut.Energy(roughness, nDotV)
	ei := lut.Energy(roughness, nDotL)
	eAvg := clampFloat(lut.AverageEnergy(roughness), 0.02, 0.98)
	fAvg := V(
		f0.X+(1-f0.X)/21,
		f0.Y+(1-f0.Y)/21,
		f0.Z+(1-f0.Z)/21,
	)
	oneMinusEAvg := math.Max(1-eAvg, 1e-4)
	denomFactor := math.Max(1-componentAvg(fAvg)*(1-eAvg), 0.05)
	common := eAvg / (math.Pi * oneMinusEAvg * denomFactor)
	geometry := math.Max(0, 1-eo) * math.Max(0, 1-ei)
	return V(
		common*geometry*fAvg.X*fAvg.X,
		common*geometry*fAvg.Y*fAvg.Y,
		common*geometry*fAvg.Z*fAvg.Z,
	)
}

func componentAvg(c Vec3) float64 { return (c.X + c.Y + c.Z) / 3 }
