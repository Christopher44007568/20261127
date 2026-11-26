package pipeline

import "math"

type sphere struct {
	center Vec3
	radius float64
	color  Vec3
}

type hit struct {
	t      float64
	point  Vec3
	normal Vec3
	sphere *sphere
}

var cameraPos = Vec3{X: 0, Y: 1.35, Z: 0}
var sunDirection = Vec3{X: -0.62, Y: 0.76, Z: -0.25}.Normalize()
var sunRadiance = Vec3{X: 3.4, Y: 2.9, Z: 2.15}

var sceneSpheres = []sphere{
	{center: Vec3{X: -0.92, Y: 0.84, Z: 3.15}, radius: 0.84, color: Vec3{X: 0.82, Y: 0.20, Z: 0.06}},
	{center: Vec3{X: 1.08, Y: 0.62, Z: 4.15}, radius: 0.62, color: Vec3{X: 0.03, Y: 0.38, Z: 0.46}},
}

const samplesPerAxis = 2

func renderShadow() (*ShadowBuffer, error) {
	buffer := &ShadowBuffer{Width: Width, Height: Height, Visibility: make([]float64, Width*Height)}
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			visibility := 0.0
			for sy := 0; sy < samplesPerAxis; sy++ {
				for sx := 0; sx < samplesPerAxis; sx++ {
					sampleVisibility := 1.0
					if point, ok := groundHit(cameraRaySample(x, y, sx, sy)); ok {
						sampleVisibility = sunVisibility(point)
					}
					visibility += sampleVisibility
				}
			}
			buffer.Visibility[y*Width+x] = visibility / (samplesPerAxis * samplesPerAxis)
		}
	}
	return buffer, nil
}

func renderScene(shadow *ShadowBuffer) (*SceneHDR, error) {
	scene := &SceneHDR{Width: Width, Height: Height, Pixels: make([]Vec3, Width*Height)}
	for y := 0; y < Height; y++ {
		for x := 0; x < Width; x++ {
			color := Vec3{}
			for sy := 0; sy < samplesPerAxis; sy++ {
				for sx := 0; sx < samplesPerAxis; sx++ {
					color = color.Add(shadeRay(cameraRaySample(x, y, sx, sy), shadow.Visibility[y*Width+x]))
				}
			}
			scene.Pixels[y*Width+x] = color.Mul(1.0 / (samplesPerAxis * samplesPerAxis))
		}
	}
	return scene, nil
}

func renderTonemap(scene *SceneHDR) *SceneLDR {
	ldr := &SceneLDR{Width: scene.Width, Height: scene.Height, Pixels: make([]Vec3, len(scene.Pixels))}
	for y := 0; y < scene.Height; y++ {
		for x := 0; x < scene.Width; x++ {
			index := y*scene.Width + x
			color := scene.Pixels[index].Mul(1.12)
			color = Vec3{
				X: acesCurve(color.X),
				Y: acesCurve(color.Y),
				Z: acesCurve(color.Z),
			}
			dx := (float64(x) + 0.5 - float64(scene.Width)*0.5) / (float64(scene.Width) * 0.5)
			dy := (float64(y) + 0.5 - float64(scene.Height)*0.5) / (float64(scene.Height) * 0.5)
			vignette := 1.0 - 0.16*smoothstep(0.35, 1.25, dx*dx+dy*dy)
			color = color.Mul(vignette)
			ldr.Pixels[index] = Vec3{
				X: math.Pow(clamp(color.X, 0, 1), 1/2.2),
				Y: math.Pow(clamp(color.Y, 0, 1), 1/2.2),
				Z: math.Pow(clamp(color.Z, 0, 1), 1/2.2),
			}
		}
	}
	return ldr
}

func cameraRaySample(x, y, sampleX, sampleY int) Vec3 {
	pixelX := (float64(x) + (float64(sampleX)+0.5)/samplesPerAxis) / Width
	pixelY := (float64(y) + (float64(sampleY)+0.5)/samplesPerAxis) / Height
	u := pixelX*2 - 1
	v := 1 - pixelY*2
	const tanHalfFOV = 0.52
	aspect := float64(Width) / float64(Height)
	return Vec3{X: u * aspect * tanHalfFOV, Y: v*tanHalfFOV - 0.075, Z: 1}.Normalize()
}

func shadeRay(ray Vec3, shadowVisibility float64) Vec3 {
	if objectHit, ok := intersectScene(cameraPos, ray); ok {
		color := shadeSurface(objectHit, ray)
		return applyFog(color, objectHit.t, ray)
	}
	if point, ok := groundHit(ray); ok {
		color := shadeGround(point, ray, shadowVisibility)
		return applyFog(color, lengthToGround(ray), ray)
	}
	return skyColor(ray)
}

func intersectScene(origin, direction Vec3) (hit, bool) {
	closest := hit{t: math.Inf(1)}
	found := false
	for i := range sceneSpheres {
		candidate, ok := intersectSphere(origin, direction, sceneSpheres[i])
		if ok && candidate.t < closest.t {
			closest = candidate
			found = true
		}
	}
	return closest, found
}

func intersectSphere(origin, direction Vec3, s sphere) (hit, bool) {
	oc := origin.Sub(s.center)
	a := direction.Dot(direction)
	b := 2 * oc.Dot(direction)
	c := oc.Dot(oc) - s.radius*s.radius
	disc := b*b - 4*a*c
	if disc < 0 {
		return hit{}, false
	}
	root := math.Sqrt(disc)
	t := (-b - root) / (2 * a)
	if t <= 1e-4 {
		t = (-b + root) / (2 * a)
	}
	if t <= 1e-4 {
		return hit{}, false
	}
	point := origin.Add(direction.Mul(t))
	return hit{t: t, point: point, normal: point.Sub(s.center).Mul(1 / s.radius), sphere: &s}, true
}

func groundHit(ray Vec3) (Vec3, bool) {
	if ray.Y >= -1e-6 {
		return Vec3{}, false
	}
	t := -cameraPos.Y / ray.Y
	if t <= 0 {
		return Vec3{}, false
	}
	return cameraPos.Add(ray.Mul(t)), true
}

func lengthToGround(ray Vec3) float64 {
	if ray.Y >= -1e-6 {
		return math.Inf(1)
	}
	return -cameraPos.Y / ray.Y
}

func shadeGround(point, ray Vec3, visibility float64) Vec3 {
	normal := Vec3{Y: 1}
	tileX := int(math.Floor(point.X*1.6 + 1000))
	tileZ := int(math.Floor(point.Z*1.6 + 1000))
	checker := (tileX + tileZ) & 1
	albedo := Vec3{X: 0.22, Y: 0.25, Z: 0.30}
	if checker == 0 {
		albedo = Vec3{X: 0.31, Y: 0.34, Z: 0.39}
	}
	groove := 1 - 0.18*smoothstep(0.04, 0.0, math.Abs(math.Mod(point.X*1.6, 1)-0.5))
	groove *= 1 - 0.18*smoothstep(0.04, 0.0, math.Abs(math.Mod(point.Z*1.6, 1)-0.5))
	albedo = albedo.Mul(groove)

	ndotl := maxFloat(normal.Dot(sunDirection), 0)
	direct := sunRadiance.Mul(ndotl * visibility * 0.78)
	skyFill := Vec3{X: 0.13, Y: 0.20, Z: 0.33}
	bounce := Vec3{X: 0.035, Y: 0.030, Z: 0.025}
	return albedo.MulV(direct.Add(skyFill).Add(bounce))
}

func shadeSurface(h hit, ray Vec3) Vec3 {
	normal := h.normal
	view := ray.Mul(-1)
	halfVector := sunDirection.Add(view).Normalize()
	ndotl := maxFloat(normal.Dot(sunDirection), 0)
	rim := math.Pow(1-maxFloat(normal.Dot(view), 0), 3)
	specular := math.Pow(maxFloat(normal.Dot(halfVector), 0), 64)

	ambient := Vec3{X: 0.07, Y: 0.10, Z: 0.16}
	direct := sunRadiance.Mul(ndotl * 0.88)
	rimLight := Vec3{X: 0.14, Y: 0.34, Z: 0.46}.Mul(rim * 0.55)
	highlight := sunRadiance.Mul(specular * 0.32)
	return h.sphere.color.MulV(direct.Add(ambient).Add(rimLight)).Add(highlight)
}

func skyColor(ray Vec3) Vec3 {
	horizon := Vec3{X: 0.42, Y: 0.59, Z: 0.73}
	zenith := Vec3{X: 0.045, Y: 0.15, Z: 0.34}
	t := smoothstep(-0.02, 0.72, ray.Y)
	sky := horizon.Lerp(zenith, t)
	sunDot := maxFloat(ray.Dot(sunDirection), 0)
	disc := math.Pow(sunDot, 1400) * 18
	halo := math.Pow(sunDot, 18) * 0.34
	return sky.Add(Vec3{X: 1.0, Y: 0.80, Z: 0.52}.Mul(disc + halo))
}

func applyFog(color Vec3, distance float64, ray Vec3) Vec3 {
	fogAmount := 1 - math.Exp(-math.Max(distance-3.5, 0)*0.055)
	return color.Lerp(skyColor(Vec3{X: ray.X, Y: maxFloat(ray.Y, 0.02), Z: ray.Z}.Normalize()), clamp(fogAmount, 0, 0.42))
}

func acesCurve(value float64) float64 {
	value = maxFloat(value, 0)
	return clamp((value*(2.51*value+0.03))/(value*(2.43*value+0.59)+0.14), 0, 1)
}

func sunVisibility(point Vec3) float64 {
	up := Vec3{Y: 1}
	tangent := sunDirection.Sub(up.Mul(sunDirection.Dot(up))).Normalize()
	bitangent := sunDirection.Cross(tangent).Normalize()
	const sampleCount = 9
	const angularRadius = 0.028
	visible := 0.0
	for i := 0; i < sampleCount; i++ {
		angle := 2 * math.Pi * float64(i) / sampleCount
		radius := angularRadius * math.Sqrt(float64(i)/sampleCount)
		offset := tangent.Mul(math.Cos(angle) * radius).Add(bitangent.Mul(math.Sin(angle) * radius))
		direction := sunDirection.Add(offset).Normalize()
		if !sunOccluded(point.Add(Vec3{Y: 0.002}), direction) {
			visible++
		}
	}
	return visible / sampleCount
}

func sunOccluded(origin, direction Vec3) bool {
	for _, s := range sceneSpheres {
		oc := s.center.Sub(origin)
		t := oc.Dot(direction)
		if t <= 0 {
			continue
		}
		closest := oc.Sub(direction.Mul(t))
		if closest.Dot(closest) < s.radius*s.radius {
			return true
		}
	}
	return false
}
