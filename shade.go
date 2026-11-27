package main

import "math"

func ShadeSurface(scene *Scene, mode RoughnessMode, n, view Vec3, u, v float64, rough RoughSample, fp Footprint) Vec3 {
	nDotV := clamp01(n.Dot(view))
	if nDotV <= 0 {
		return V(0.01, 0.015, 0.025)
	}
	f0 := scene.F0
	r := rough.Roughness

	reflection := view.Mul(-1).Add(n.Mul(2 * nDotV)).Normalize()
	env := studioEnvironment(reflection, r)
	fresnel := fresnelSchlick(nDotV, f0)

	color := env.MulVec(fresnel).Mul(0.72 + 0.28*nDotV)

	for _, light := range scene.Lights {
		nDotL := clamp01(n.Dot(light.Direction))
		if nDotL <= 0 {
			continue
		}
		brdf := specularGGX(f0, r, n, view, light.Direction)
		radiance := light.Color.Mul(light.Radiance)
		color = color.Add(brdf.MulVec(radiance).Mul(nDotL))

		if mode == ModeFootprint && scene.LUT != nil {
			nDotH := clamp01(n.Dot(view.Add(light.Direction).Normalize()))
			nDotLEnergy := math.Max(nDotL, 0.05)
			ms := scene.LUT.MultiScatterBRDF(f0, r, nDotV, nDotLEnergy)
			color = color.Add(ms.MulVec(radiance).Mul(nDotL * 0.85))
			_ = nDotH
		}
	}

	if mode == ModeFootprint && scene.LUT != nil {
		directionalEnergy := scene.LUT.Energy(r, nDotV)
		deficit := math.Max(0, 1-directionalEnergy)
		fAvg := V(
			f0.X+(1-f0.X)/21,
			f0.Y+(1-f0.Y)/21,
			f0.Z+(1-f0.Z)/21,
		)
		ambient := V(0.020, 0.036, 0.066)
		color = color.Add(ambient.MulVec(fAvg).Mul(deficit * 0.85))

		// A small view-independent multiscatter floor keeps coarse mip levels
		// from losing total highlight energy as the lobe becomes very broad.
		color = color.Add(V(0.006, 0.011, 0.022).Mul(r * r * (0.20 + deficit)))
	}

	// Slight blue rim makes the sphere silhouette readable against the dark
	// floor without adding texture-dependent sparkle.
	rim := math.Pow(1-nDotV, 3.2)
	color = color.Add(V(0.012, 0.024, 0.052).Mul(rim))
	return color
}

func studioEnvironment(r Vec3, roughness float64) Vec3 {
	up := clamp01(r.Y*0.5 + 0.5)
	ground := V(0.003, 0.006, 0.011)
	sky := V(0.018, 0.034, 0.061)
	col := ground.Mul(1 - up).Add(sky.Mul(up))

	horizon := math.Exp(-math.Abs(r.Y) * 7.5)
	col = col.Add(V(0.014, 0.028, 0.058).Mul(horizon * 0.28))

	// Softbox reflections are deliberately broadened with roughness. These are
	// low-frequency environment contributions; the sharp comparison highlights
	// come from the GGX light loop.
	boxes := []struct {
		dir Vec3
		col Vec3
		k   float64
	}{
		{V(-0.48, 0.40, 0.78).Normalize(), V(1.0, 0.86, 0.66), 0.34},
		{V(0.05, 0.68, 0.73).Normalize(), V(0.92, 0.96, 1.0), 0.42},
		{V(0.67, 0.10, 0.74).Normalize(), V(0.40, 0.78, 1.0), 0.38},
		{V(-0.53, -0.22, 0.82).Normalize(), V(0.35, 0.52, 1.0), 0.24},
	}
	width := 0.004 + roughness*roughness*0.16
	peak := 0.020 / (width + 0.006)
	for _, box := range boxes {
		d := clamp01(r.Dot(box.dir))
		lobe := math.Exp(-(1 - d) / width)
		col = col.Add(box.col.Mul(lobe * box.k * peak * 0.34))
	}

	avg := V(0.010, 0.020, 0.040)
	blur := clamp01(roughness * 0.72)
	col = col.Mul(1 - blur).Add(avg.Mul(blur))

	return col
}
