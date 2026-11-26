package render

import "math"

// StudioEnvironment is a procedural, bounded-value studio. It contains broad
// gradients plus a few softboxes, giving the clearcoat something meaningful to
// reflect without requiring an HDR texture or any external package.
func StudioEnvironment(dir Vec3) Vec3 {
	dir = dir.Normalize()

	sky := LerpVec(
		V(0.012, 0.019, 0.030),
		V(0.115, 0.145, 0.190),
		Smoothstep(-0.65, 0.95, dir.Y),
	)
	if dir.Y < 0 {
		floorTone := LerpVec(V(0.010, 0.012, 0.016), V(0.045, 0.050, 0.058), Smoothstep(-1, 0, dir.Y))
		sky = LerpVec(sky, floorTone, Smoothstep(0, -0.22, dir.Y))
	}

	key := V(-0.42, 0.72, 0.56).Normalize()
	keyColor := V(1.00, 0.87, 0.70)
	sky = sky.Add(softbox(dir, key, 0.43, 0.22, 0.025, keyColor).Mul(0.98))

	strip := V(0.78, 0.30, 0.55).Normalize()
	stripColor := V(0.42, 0.66, 1.00)
	sky = sky.Add(softbox(dir, strip, 0.035, 0.38, 0.018, stripColor).Mul(0.72))

	rim := V(0.12, 0.34, -0.93).Normalize()
	rimColor := V(0.52, 0.84, 0.78)
	sky = sky.Add(softbox(dir, rim, 0.36, 0.045, 0.035, rimColor).Mul(0.75))

	return sky.Clamp01()
}

func softbox(dir, center Vec3, halfWidth, halfHeight, feather float64, color Vec3) Vec3 {
	d := dir.Dot(center)
	if d < 0.05 {
		return V(0, 0, 0)
	}

	tangent, bitangent := Basis(center)
	u := dir.Dot(tangent) / d
	v := dir.Dot(bitangent) / d

	maskU := 1 - Smoothstep(halfWidth-feather, halfWidth+feather, math.Abs(u))
	maskV := 1 - Smoothstep(halfHeight-feather, halfHeight+feather, math.Abs(v))
	facing := Smoothstep(0.05, 0.35, d)
	core := 0.72 + 0.28*Smoothstep(0, halfHeight, 1-math.Abs(v)/halfHeight)
	return color.Mul(maskU * maskV * facing * core)
}

// SampleReflection filters the studio with a deterministic cone. A smooth coat
// returns the mirror sample; rough coats average a wider neighborhood.
func SampleReflection(dir Vec3, roughness float64) Vec3 {
	dir = dir.Normalize()
	spread := RoughnessToSpread(roughness)
	if spread < 0.012 {
		return StudioEnvironment(dir)
	}

	samples := 6 + int(18*Clamp01(roughness))
	tangent, bitangent := Basis(dir)
	sum := V(0, 0, 0)
	const goldenAngle = 2.399963229728653

	for i := 0; i < samples; i++ {
		radius := math.Sqrt((float64(i)+0.5)/float64(samples)) * spread
		angle := float64(i) * goldenAngle
		offset := tangent.Mul(math.Cos(angle) * radius).
			Add(bitangent.Mul(math.Sin(angle) * radius))
		sum = sum.Add(StudioEnvironment(dir.Add(offset).Normalize()))
	}
	return sum.Mul(1 / float64(samples)).Clamp01()
}
