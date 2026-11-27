package furnace

import "math"

// BRDF is implemented by material-specific evaluators.
type BRDF interface {
	BRDF(v, l Vec3) Vec3
}

// Hammersley2D returns a deterministic low-discrepancy sample in [0,1)^2.
func Hammersley2D(i, n int) (u1, u2 float64) {
	u1 = (float64(i) + 0.5) / float64(n)
	bits := uint32(i + 1)
	bits = ((bits >> 1) & 0x55555555) | ((bits & 0x55555555) << 1)
	bits = ((bits >> 2) & 0x33333333) | ((bits & 0x33333333) << 2)
	bits = ((bits >> 4) & 0x0f0f0f0f) | ((bits & 0x0f0f0f0f) << 4)
	bits = ((bits >> 8) & 0x00ff00ff) | ((bits & 0x00ff00ff) << 8)
	bits = (bits >> 16) | (bits << 16)
	u2 = float64(bits) / 4294967296.0
	return u1, u2
}

// CosineHemisphere maps two uniform random values to a cosine-weighted
// direction about +Z. Its pdf is z/pi.
func CosineHemisphere(u1, u2 float64) Vec3 {
	r := math.Sqrt(math.Max(0, u1))
	phi := 2 * math.Pi * u2
	x := r * math.Cos(phi)
	y := r * math.Sin(phi)
	z := math.Sqrt(math.Max(0, 1-u1))
	return V(x, y, z)
}

func outgoingDirection(mu float64) Vec3 {
	mu = clamp(mu, 1e-4, 1)
	s := math.Sqrt(math.Max(0, 1-mu*mu))
	return V(s, 0, mu)
}

// IntegrateHemisphere computes E(v)=∫ f(v,l) cosThetaL dOmega with deterministic
// cosine-weighted Monte Carlo. The returned color is the directional albedo.
func IntegrateHemisphere(f func(v, l Vec3) Vec3, mu float64, samples int) Vec3 {
	if samples < 16 {
		samples = 16
	}
	v := outgoingDirection(mu)
	sum := V(0, 0, 0)
	for i := 0; i < samples; i++ {
		u1, u2 := Hammersley2D(i, samples)
		l := CosineHemisphere(u1, u2)
		brdf := f(v, l)
		if !finiteVec(brdf) || brdf.X < 0 || brdf.Y < 0 || brdf.Z < 0 {
			continue
		}
		// Divide f*z by pdf=z/pi.
		sum = sum.Add(brdf.Mul(math.Pi))
	}
	return sum.Div(float64(samples))
}

// DirectionalIntegral is a readable alias for callers thinking in terms of the
// white-furnace directional energy integral.
func DirectionalIntegral(f func(v, l Vec3) Vec3, nDotV float64, samples int) Vec3 {
	return IntegrateHemisphere(f, nDotV, samples)
}

// DirectionalAlbedo integrates the raw material BRDF plus its diffuse lobe.
// GGX terms use visible-normal importance sampling; broad cloth terms use
// cosine sampling. This avoids Monte Carlo noise in narrow highlights.
func DirectionalAlbedo(m Material, nDotV float64, samples int) Vec3 {
	if samples < 32 {
		samples = 32
	}
	mu := clamp(nDotV, 1e-4, 1)

	metalWeight := saturate(m.Metallic)
	diffuseWeight := 1 - metalWeight
	if m.Kind == Clearcoat || m.Kind == Velvet {
		diffuseWeight = 1
	}
	e := m.Albedo.Mul(diffuseWeight)

	switch m.Kind {
	case RoughMetal:
		e = e.Add(integrateGGXEnergy(mu, m.alpha(), m.f0(), samples))
	case Clearcoat:
		f0 := clamp(m.CoatF0, 0.02, 0.08)
		coatE := integrateGGXEnergy(mu, m.coatAlpha(), V(f0, f0, f0), samples)
		e = e.Add(coatE.Mul(m.Coat))
	case Velvet:
		sheen := IntegrateHemisphere(func(v, l Vec3) Vec3 {
			return VelvetBRDF(m, v, l)
		}, mu, samples)
		e = e.Add(sheen)
		if metalWeight > 0 {
			e = e.Add(integrateGGXEnergy(mu, m.alpha(), m.f0(), samples).Mul(metalWeight))
		}
	}
	return e
}

func integrateGGXEnergy(mu, alpha float64, f0 Vec3, samples int) Vec3 {
	if samples < 16 {
		samples = 16
	}
	v := outgoingDirection(mu)
	lambdaV := lambdaGGX(mu, alpha)
	g1V := 1 / (1 + lambdaV)
	sum := V(0, 0, 0)
	valid := 0

	for i := 0; i < samples; i++ {
		u1, u2 := Hammersley2D(i, samples)
		h := sampleGGXVNDF(v, alpha, u1, u2)
		vDotH := v.Dot(h)
		if vDotH <= 1e-6 {
			continue
		}
		l := h.Mul(2 * vDotH).Sub(v).Normalize()
		if l.Z <= 1e-6 {
			continue
		}
		g2 := SmithG2GGX(mu, l.Z, alpha)
		weight := g2 / g1V
		if !finite(weight) || weight < 0 {
			continue
		}
		f := FresnelSchlickV(vDotH, f0)
		sum = sum.Add(f.Mul(weight))
		valid++
	}
	if valid == 0 {
		return V(0, 0, 0)
	}
	return sum.Div(float64(samples))
}

func sampleGGXVNDF(v Vec3, alpha, u1, u2 float64) Vec3 {
	return sampleAnisotropicGGXVNDF(v, alpha, alpha, u1, u2)
}

func sampleAnisotropicGGXVNDF(ve Vec3, alphaX, alphaY, u1, u2 float64) Vec3 {
	vh := V(alphaX*ve.X, alphaY*ve.Y, ve.Z).Normalize()
	lenSqr := vh.X*vh.X + vh.Y*vh.Y
	var t1 Vec3
	if lenSqr > 1e-12 {
		inv := 1 / math.Sqrt(lenSqr)
		t1 = V(-vh.Y*inv, vh.X*inv, 0)
	} else {
		t1 = V(1, 0, 0)
	}
	t2 := vh.Cross(t1).Normalize()

	r := math.Sqrt(math.Max(0, u1))
	phi := 2 * math.Pi * u2
	x := r * math.Cos(phi)
	y := r * math.Sin(phi)

	// Fold the disk into the visible half of the ellipsoid.
	s := 0.5 * (1 + vh.Z)
	y = (1-s)*math.Sqrt(math.Max(0, 1-x*x)) + s*y

	oneMinus := math.Max(0, 1-x*x-y*y)
	nh := t1.Mul(x).Add(t2.Mul(y)).Add(vh.Mul(math.Sqrt(oneMinus)))
	ne := V(alphaX*nh.X, alphaY*nh.Y, nh.Z)
	return ne.Normalize()
}
