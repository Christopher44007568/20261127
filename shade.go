package main

import (
	"image"
	"math"
)

type ShadeMode int

const (
	ShadeHeight ShadeMode = iota
	ShadeNaive
	ShadeMaterialMud
	ShadeMaterialMoss
	ShadeMaterialStone
	ShadeReflectionWet
	ShadeReflectionDry
	ShadeReflectionDiff
)

var sunDir = Vec3{-0.48, 0.78, 0.40}.Normalize()

func colorToRGBA(c Vec3) (r, g, b, a uint8) {
	toByte := func(v float64) uint8 {
		v = math.Pow(saturate(v), 1.0/2.2)
		return uint8(math.Round(v * 255))
	}
	return toByte(c.X), toByte(c.Y), toByte(c.Z), 255
}

func rgbaOf(c Vec3) (r, g, b, a uint8) { return colorToRGBA(c) }

func luminance(c Vec3) float64 {
	return 0.2126*c.X + 0.7152*c.Y + 0.0722*c.Z
}

func skyColor(d Vec3) Vec3 {
	t := saturate(d.Y*0.85 + 0.12)
	horizon := Vec3{0.115, 0.135, 0.145}
	zenith := Vec3{0.028, 0.072, 0.135}
	c := horizon.Lerp(zenith, t)
	sun := math.Pow(saturate(d.Dot(sunDir)), 80)
	return c.Add(Vec3{0.50, 0.39, 0.24}.Mul(sun * 0.45))
}

func materialAlbedo(i int, wetStone float64, s MaterialSample) Vec3 {
	a := s.Albedo
	if i == WetStone {
		a = a.Mul(1.0 + 0.34*wetStone)
		a = a.Add(Vec3{0.045, 0.052, 0.050}.Mul(wetStone))
	}
	return a
}

func reflectionRadiance(cam Camera, p, n Vec3, samples [numMaterials]MaterialSample, w Mat3, wetness float64, wetPass bool) Vec3 {
	view := cam.Eye.Sub(p).Normalize()
	ndv := saturate(n.Dot(view))
	ndl := saturate(n.Dot(sunDir))
	rough, specular := 0.0, 0.0
	for i := 0; i < numMaterials; i++ {
		rough += w[i] * samples[i].Rough
		specular += w[i] * samples[i].Specular
	}
	wetStone := 0.0
	if wetPass {
		wetStone = w[WetStone] * wetness
		rough = mix(rough, 0.16, wetStone*0.75)
		specular += wetStone * 0.52
	}
	half := sunDir.Add(view).Normalize()
	ndh := saturate(n.Dot(half))
	shininess := mix(18, 150, saturate((1-rough)*0.85))
	fresnel := 0.035 + 0.965*math.Pow(1-ndv, 5)
	glint := math.Pow(ndh, shininess) * (shininess + 8) / (8 * math.Pi)
	sun := Vec3{1.00, 0.90, 0.75}.Mul(glint * ndl * specular * fresnel * 2.8)

	reflDir := view.Mul(-1).Sub(n.Mul(2 * view.Mul(-1).Dot(n))).Normalize()
	env := skyColor(reflDir).Mul(fresnel * specular * (0.20 + 0.80*wetStone) * 1.7)
	return sun.Add(env)
}

func shadeFinal(cam Camera, p, n Vec3, samples [numMaterials]MaterialSample, w Mat3, wetness float64) Vec3 {
	view := cam.Eye.Sub(p).Normalize()
	ndv := saturate(n.Dot(view))
	ndl := saturate(n.Dot(sunDir))
	wetStone := w[WetStone] * wetness
	albedo := Vec3{}
	for i := 0; i < numMaterials; i++ {
		albedo = albedo.Add(materialAlbedo(i, wetness, samples[i]).Mul(w[i]))
	}
	sky := Vec3{0.105, 0.145, 0.175}
	ground := Vec3{0.080, 0.065, 0.045}
	ambient := sky.Mul(0.22 + 0.30*n.Y).Add(ground.Mul(0.10 * (1 - n.Y)))
	diffuse := albedo.MulVec(ambient.Add(Vec3{1.22, 1.10, 0.96}.Mul(ndl * 1.42)))
	refl := reflectionRadiance(cam, p, n, samples, w, wetness, true)
	refl = refl.Mul(1.0 + 0.42*wetStone)
	col := diffuse.Add(refl)
	col = col.Add(Vec3{0.095, 0.135, 0.140}.Mul(wetStone * (0.14 + 0.34*math.Pow(1-ndv, 2))))

	depth := waterLevel - p.Y
	if depth > 0 {
		waterTint := Vec3{0.012, 0.060, 0.070}
		a := (0.15 + 0.28*smoothstep(0, 0.20, depth)) * (1 - 0.35*wetStone)
		col = col.Lerp(waterTint, a)
		ripple := math.Sin(p.X*21 + p.Z*17 + 2.5*fbm2(p.X*5, p.Z*5, 2))
		col = col.Add(Vec3{0.018, 0.050, 0.052}.Mul(saturate(ripple) * 0.25 * (1 - ndv)))
	}
	waterline := math.Exp(-math.Abs(p.Y-waterLevel) / 0.0060)
	col = col.Add(Vec3{0.055, 0.245, 0.255}.Mul(waterline * 0.72))
	return col
}

func materialOnlyColor(cam Camera, mat int, p, n Vec3, sample MaterialSample, weight, wetness float64) Vec3 {
	ndl := saturate(n.Dot(sunDir))
	albedo := materialAlbedo(mat, wetness, sample).Mul(0.08 + 1.05*weight)
	sky := Vec3{0.095, 0.130, 0.160}
	ground := Vec3{0.070, 0.058, 0.040}
	ambient := sky.Mul(0.25 + 0.35*n.Y).Add(ground.Mul(0.10 * (1 - n.Y)))
	col := albedo.MulVec(ambient.Add(Vec3{1.08, 0.92, 0.70}.Mul(ndl * 0.90)))
	if mat == WetStone {
		w := [numMaterials]float64{}
		w[WetStone] = weight
		samples := [numMaterials]MaterialSample{}
		samples[WetStone] = sample
		col = col.Add(reflectionRadiance(cam, p, n, samples, w, wetness, true))
	}
	line := math.Exp(-math.Abs(p.Y-waterLevel) / 0.0060)
	col = col.Add(Vec3{0.035, 0.135, 0.145}.Mul(line * 0.35))
	return col
}

func heatColor(v float64) Vec3 {
	v = saturate(v)
	dark := Vec3{0.010, 0.022, 0.045}
	blue := Vec3{0.020, 0.185, 0.330}
	cyan := Vec3{0.050, 0.700, 0.720}
	yellow := Vec3{0.960, 0.720, 0.120}
	white := Vec3{1.000, 0.980, 0.850}
	switch {
	case v < 0.12:
		return dark.Lerp(blue, v/0.12)
	case v < 0.42:
		return blue.Lerp(cyan, (v-0.12)/0.30)
	case v < 0.75:
		return cyan.Lerp(yellow, (v-0.42)/0.33)
	default:
		return yellow.Lerp(white, (v-0.75)/0.25)
	}
}

func shadeImage(cam Camera, g *GBuffer, mode ShadeMode) *image.RGBA {
	return gbufferToImage(g, func(x, y, i int) Vec3 {
		p, n := g.Pos[i], g.Nrm[i]
		var samples [numMaterials]MaterialSample
		for m := 0; m < numMaterials; m++ {
			samples[m] = sampleMaterial(m, p, n)
		}
		wn := naiveWeights(g.Cov[i])
		wh := heightWeights(g.Cov[i], samples)
		wetness := wetnessAt(p.X, p.Z, p.Y)
		switch mode {
		case ShadeNaive:
			c := shadeFinal(cam, p, n, samples, Mat3(wn), wetness)
			return c.Mul(0.48 + 0.52*g.AO[i])
		case ShadeMaterialMud:
			c := materialOnlyColor(cam, Mud, p, n, samples[Mud], wh[Mud], wetness)
			return c.Mul(0.55 + 0.45*g.AO[i])
		case ShadeMaterialMoss:
			c := materialOnlyColor(cam, Moss, p, n, samples[Moss], wh[Moss], wetness)
			return c.Mul(0.55 + 0.45*g.AO[i])
		case ShadeMaterialStone:
			c := materialOnlyColor(cam, WetStone, p, n, samples[WetStone], wh[WetStone], wetness)
			return c.Mul(0.55 + 0.45*g.AO[i])
		case ShadeReflectionWet:
			r := reflectionRadiance(cam, p, n, samples, Mat3(wh), wetness, true)
			return Vec3{0.006, 0.014, 0.025}.Add(r.Mul(5.5))
		case ShadeReflectionDry:
			r := reflectionRadiance(cam, p, n, samples, Mat3(wh), 0, false)
			return Vec3{0.006, 0.014, 0.025}.Add(r.Mul(5.5))
		case ShadeReflectionDiff:
			wet := reflectionRadiance(cam, p, n, samples, Mat3(wh), wetness, true)
			dry := reflectionRadiance(cam, p, n, samples, Mat3(wh), 0, false)
			return heatColor(luminance(wet.Sub(dry)) * 6.5)
		default:
			c := shadeFinal(cam, p, n, samples, Mat3(wh), wetness)
			return c.Mul(0.48 + 0.52*g.AO[i])
		}
	})
}
