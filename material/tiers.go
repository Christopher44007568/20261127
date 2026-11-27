package material

import "math"

// Shade evaluates one tier for a unit surface normal and view vector.
// The returned color is linear HDR; call Display/ToneMap before encoding.
func Shade(m Material, tier Tier, normal, view Vec3) Color {
	n := normal.Normalize()
	v := view.Normalize()
	if v.Length() == 0 {
		v = Vec3{Z: 1}
	}
	switch tier {
	case TierLow:
		return shadeLow(m, n, v)
	case TierMedium:
		return shadeCookTorrance(m, n, v, false)
	default:
		return shadeCookTorrance(m, n, v, true)
	}
}

// ShadeLOD evaluates a continuous blend of all three tiers.
func ShadeLOD(m Material, lod float64, normal, view Vec3, width float64) (Color, LODWeights) {
	w := WeightsAt(lod, width)
	low := shadeLow(m, normal.Normalize(), view.Normalize())
	medium := shadeCookTorrance(m, normal.Normalize(), view.Normalize(), false)
	high := shadeCookTorrance(m, normal.Normalize(), view.Normalize(), true)
	return w.Blend(low, medium, high), w
}

func shadeLow(m Material, n, v Vec3) Color {
	u := m.LowUniforms()
	rough := clamp(u.Roughness, u.RoughnessMin, u.RoughnessMax)
	ndv := max0(n.Dot(v))
	ndl := max0(n.Dot(keyLightDir))

	// Roughness supplies the only soft-wrap approximation in the minimal tier.
	wrap := 0.16 + 0.52*rough
	wrapped := clamp((ndl+wrap)/(1+wrap), 0, 1)
	hemi := 0.18 + 0.10*max0(n.Y)

	ambient := u.BaseColor.Scale(hemi * (0.72 + 0.28*u.Metallic))
	diffuse := u.BaseColor.Scale((1 - u.Metallic) * 0.82 * wrapped).Mul(keyRadiance)

	h := keyLightDir.Add(v).Normalize()
	nh := max0(n.Dot(h))
	exponent := mix(112, 7, rough)
	specTint := Color{R: 1, G: 0.96, B: 0.90}
	specTint = Color{
		R: mix(specTint.R, u.BaseColor.R, u.Metallic),
		G: mix(specTint.G, u.BaseColor.G, u.Metallic),
		B: mix(specTint.B, u.BaseColor.B, u.Metallic),
	}
	specStrength := (0.20 + 0.42*(1-rough)) * (0.35 + 0.65*u.Metallic)
	spec := specTint.Scale(math.Pow(nh, exponent) * specStrength).Mul(keyRadiance)

	fresnel := math.Pow(1-clamp01(ndv), 4)
	rim := u.BaseColor.Scale(0.10 + 0.22*fresnel)
	edgeSpec := specTint.Scale(0.20 * fresnel * (1 - rough*0.45))

	return ambient.Add(diffuse).Add(spec).Add(rim).Add(edgeSpec)
}

func shadeCookTorrance(m Material, n, v Vec3, high bool) Color {
	rough := m.EffectiveRoughness()
	ndv := clamp(n.Dot(v), 0, 1)
	ndvSafe := math.Max(ndv, 0.002)
	f0 := f0For(m)

	color := environmentAmbient(m, n, v, ndv, f0, high)

	lights := []struct {
		dir      Vec3
		radiance Color
		spec     float64
	}{
		{keyLightDir, keyRadiance, 1},
		{fillLightDir, fillRadiance, 0.35},
	}
	for _, light := range lights {
		l := light.dir.Normalize()
		ndl := clamp(n.Dot(l), 0, 1)
		wrappedNDL := ndl
		if m.Wrap > 0 {
			wrappedNDL = clamp((ndl+m.Wrap)/(1+m.Wrap), 0, 1)
		}

		if ndl > 0 {
			h := l.Add(v).Normalize()
			vdh := max0(v.Dot(h))
			d := distributionGGX(n, h, rough)
			g := geometrySmith(n, v, l, rough)
			f := fresnelSchlick(vdh, f0)
			denom := 4 * ndl * ndvSafe
			specScale := light.spec
			specular := Color{
				R: d * g * f.R / denom,
				G: d * g * f.G / denom,
				B: d * g * f.B / denom,
			}.Mul(light.radiance).Scale(ndl * specScale)

			fAvg := (f.R + f.G + f.B) / 3
			kd := (1 - fAvg) * (1 - m.Metallic)
			diffuse := m.BaseColor.Scale(kd * wrappedNDL / math.Pi).Mul(light.radiance).Scale(math.Pi * 0.92)
			color = color.Add(diffuse).Add(specular)

			if high && m.ClearcoatWeight > 0 {
				ccRough := clamp(m.ClearcoatRough, 0.04, 1)
				dc := distributionGGX(n, h, ccRough)
				gc := geometrySmith(n, v, l, ccRough)
				fc := fresnelSchlick(vdh, Color{R: 0.04, G: 0.04, B: 0.04})
				cc := Color{
					R: dc * gc * fc.R / denom,
					G: dc * gc * fc.G / denom,
					B: dc * gc * fc.B / denom,
				}.Mul(light.radiance).Scale(ndl * m.ClearcoatWeight * specScale)
				color = color.Add(cc)
			}
		} else if m.Wrap > 0 {
			diffuse := m.BaseColor.Scale(wrappedNDL * (1 - m.Metallic) * 0.92).Mul(light.radiance)
			color = color.Add(diffuse)
		}
	}

	if high {
		color = color.Add(sheenTerm(m, n, v, ndv))
	}
	return color
}

func environmentAmbient(m Material, n, v Vec3, ndv float64, f0 Color, high bool) Color {
	hemi := 0.16 + 0.11*max0(n.Y)
	diffuseAmbient := m.BaseColor.Scale(hemi * (1 - m.Metallic) * 0.72)

	f := fresnelSchlick(math.Max(ndv, 0.002), f0)
	reflection := reflectVector(v.Mul(-1), n)
	env := sampleEnvironment(reflection)
	rough := m.EffectiveRoughness()
	if high {
		blur := clamp01(rough * 0.85)
		avg := (env.R + env.G + env.B) / 3
		env = Color{
			R: mix(env.R, avg*0.82+0.10, blur),
			G: mix(env.G, avg*0.86+0.11, blur),
			B: mix(env.B, avg*0.95+0.16, blur),
		}
		ibl := env.Mul(f).Scale((1 - rough*0.55) * 0.72)
		metalBody := m.BaseColor.Mul(env).Scale(m.Metallic * 0.34)
		return diffuseAmbient.Add(ibl).Add(metalBody).Add(m.BaseColor.Scale(0.055))
	}

	simpleSpec := f.Scale(0.10 + 0.10*(1-rough))
	return diffuseAmbient.Add(simpleSpec).Add(m.BaseColor.Scale(0.05 * m.Metallic))
}

func reflectVector(incident, normal Vec3) Vec3 {
	d := incident.Dot(normal)
	return incident.Sub(normal.Mul(2 * d)).Normalize()
}

func sampleEnvironment(r Vec3) Color {
	up := clamp01(r.Y*0.5 + 0.5)
	ground := Color{R: 0.16, G: 0.17, B: 0.20}
	sky := Color{R: 0.62, G: 0.72, B: 0.88}
	env := Color{
		R: mix(ground.R, sky.R, up),
		G: mix(ground.G, sky.G, up),
		B: mix(ground.B, sky.B, up),
	}
	box := math.Pow(max0(r.Dot(Vec3{X: -0.35, Y: 0.72, Z: 0.60}.Normalize())), 24)
	soft := math.Pow(max0(r.Dot(Vec3{X: 0.82, Y: 0.18, Z: 0.54}.Normalize())), 42)
	return env.Add(Color{R: box * 2.2, G: box * 2.05, B: box * 1.75}).
		Add(Color{R: soft * 0.9, G: soft * 1.0, B: soft * 1.25})
}

func sheenTerm(m Material, n, v Vec3, ndv float64) Color {
	if m.SheenWeight <= 0 {
		return Color{}
	}
	l := keyLightDir
	ndl := max0(n.Dot(l))
	h := l.Add(v).Normalize()
	nh := max0(n.Dot(h))
	shape := math.Pow(nh, mix(18, 3, m.SheenRoughness))
	rim := math.Pow(1-clamp01(ndv), 2.4)
	intensity := m.SheenWeight * (0.28*ndl + 0.34*shape + 0.22*rim)
	return m.SheenColor.Scale(intensity)
}
