package main

import "math"

// integrateVolume marches the transparent medium independently from the
// opaque depth buffer. Every sample resolves its own froxel by (pixel, viewZ);
// no surface or neighboring-cell light list is reused.
func integrateVolume(scene *Scene, camera *Camera, ray Ray, maxT float64, info pixelInfo, cfg RenderConfig, stats *RenderStats) (Vec3, float64) {
	vol := scene.Volume
	if vol == nil || cfg.FogSteps <= 0 {
		return Vec3{}, 1
	}
	t0, t1, ok := rayBox(ray, vol.Bounds)
	if !ok {
		return Vec3{}, 1
	}
	t0 = math.Max(t0, camera.Near*0.5)
	if !math.IsInf(maxT, 1) {
		t1 = math.Min(t1, maxT)
	}
	t1 = math.Min(t1, camera.Far)
	if t1 <= t0 {
		return Vec3{}, 1
	}

	steps := cfg.FogSteps
	ds := (t1 - t0) / float64(steps)
	transmittance := 1.0
	scattered := Vec3{}
	for i := 0; i < steps; i++ {
		t := t0 + (float64(i)+0.5)*ds
		p := ray.At(t)
		density := mediumDensity(p, vol)
		if density <= 1e-6 {
			continue
		}
		sigma := vol.Density * density
		alpha := 1 - math.Exp(-sigma*ds)
		color := volumeLight(scene, camera, ray.Direction, p, info, vol, stats)
		scattered = scattered.Add(color.Mul(transmittance * alpha))
		transmittance *= 1 - alpha
		if transmittance < 0.002 {
			break
		}
	}
	return scattered, transmittance
}

func mediumDensity(p Vec3, vol *Volume) float64 {
	base := 0.72 + 0.28*math.Sin(p.X*0.82+1.1)*math.Sin(p.Z*0.67-0.35)
	wisps := 0.85 + 0.15*math.Sin(p.X*2.1+p.Z*0.73+p.Y*0.6)
	vertical := smoothstep(vol.Bounds.Min.Y, vol.Bounds.Min.Y+0.35, p.Y) *
		smoothstep(vol.Bounds.Max.Y, vol.Bounds.Max.Y-0.8, p.Y)
	return clamp(base*wisps*vertical, 0, 1.35)
}

func volumeLight(scene *Scene, camera *Camera, rayDir Vec3, p Vec3, info pixelInfo, vol *Volume, stats *RenderStats) Vec3 {
	viewZ := camera.ViewZ(p)
	lights, _, fallback := queryLights(scene, info, viewZ)
	stats.VolumeSamples++
	if fallback != NoFallback && info.Mode == FroxelMode {
		stats.VolumeFallback++
	}
	sum := Vec3{}
	for _, lightIndex := range lights {
		light := scene.Lights[lightIndex]
		att := light.Attenuation(p)
		if att <= 0 {
			continue
		}
		toLight := light.Position.Sub(p)
		distance := toLight.Length()
		if distance < 1e-6 {
			continue
		}
		cosTheta := toLight.Div(distance).Dot(rayDir.Neg())
		phase := henyeyGreenstein(cosTheta, vol.Anisotropy)
		sum = sum.Add(light.Color.Mul(att * phase))
	}
	ambient := V(0.010, 0.013, 0.021)
	return ambient.Add(sum.Mul(0.42))
}

func henyeyGreenstein(cosTheta, g float64) float64 {
	g2 := g * g
	denom := 1 + g2 - 2*g*cosTheta
	if denom < 1e-5 {
		denom = 1e-5
	}
	return (1 - g2) / (4 * math.Pi * math.Pow(denom, 1.5))
}
