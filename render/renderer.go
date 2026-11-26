package render

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"
)

var (
	keyDirection  = V(-0.44, 0.74, 0.52).Normalize()
	fillDirection = V(0.74, 0.26, 0.62).Normalize()
)

func diffuseLighting(normal Vec3, keyVisibility, fillVisibility float64) Vec3 {
	key := Clamp01(normal.Dot(keyDirection)) * 0.64 * Clamp01(keyVisibility)
	fill := Clamp01(normal.Dot(fillDirection)) * 0.15 * Clamp01(fillVisibility)
	ambientT := Smoothstep(-0.35, 0.95, normal.Y)
	ambient := LerpVec(V(0.028, 0.033, 0.045), V(0.16, 0.19, 0.235), ambientT)

	warm := V(1.00, 0.87, 0.70).Mul(key)
	cool := V(0.43, 0.62, 1.00).Mul(fill)
	return warm.Add(cool).Add(ambient).Clamp01()
}

func backdrop(dir Vec3) Vec3 {
	t := Smoothstep(-0.22, 0.92, dir.Normalize().Y)
	base := LerpVec(V(0.006, 0.009, 0.015), V(0.072, 0.091, 0.123), t)
	glow := math.Exp(-dir.X*dir.X*18) * 0.006 * Smoothstep(-0.1, 0.7, dir.Y)
	return base.Add(V(0.55, 0.72, 1.00).Mul(glow))
}

func shadeSphere(scene Scene, ray Ray, distance float64, mode RenderMode) Vec3 {
	sphere := scene.Sphere
	point := ray.At(distance)
	normal := point.Sub(sphere.Center).Normalize()
	view := ray.Direction.Mul(-1)
	nDotV := Clamp01(normal.Dot(view))

	light := diffuseLighting(normal, 1, 1)
	reflected := SampleReflection(Reflect(ray.Direction, normal), sphere.RoughnessAt(normal))
	sample := sphere.Material.Shade(nDotV, light, reflected)

	switch mode {
	case RenderDiffuse:
		return sample.Diffuse
	case RenderClearcoat:
		return sample.Clearcoat
	default:
		return sample.Composite()
	}
}

func shadeFloor(scene Scene, point Vec3) Vec3 {
	normal := V(0, 1, 0)
	origin := point.Add(normal.Mul(1e-4))
	keyVisibility := 1.0
	if scene.Sphere.Occludes(origin, keyDirection, 30) {
		keyVisibility = 0
	}
	fillVisibility := 1.0
	if scene.Sphere.Occludes(origin, fillDirection, 30) {
		fillVisibility = 0
	}

	albedo := V(0.047, 0.055, 0.066)
	contact := 1 - Smoothstep(0.35, 4.0, point.Sub(scene.Sphere.Center).Length())
	albedo = albedo.Mul(0.72 + 0.28*contact)
	floor := Material{Albedo: albedo}
	return floor.Shade(1, diffuseLighting(normal, keyVisibility, fillVisibility), V(0, 0, 0)).Diffuse
}

func trace(scene Scene, ray Ray, opts RenderOptions) Vec3 {
	sphereT, sphereHit := scene.Sphere.Hit(ray)
	planeT := math.Inf(1)
	if ray.Direction.Y < -1e-8 && ray.Origin.Y > scene.FloorY {
		planeT = (scene.FloorY - ray.Origin.Y) / ray.Direction.Y
	}

	if sphereHit && sphereT < planeT {
		return shadeSphere(scene, ray, sphereT, opts.Mode)
	}
	if planeT > 1e-6 && !math.IsInf(planeT, 1) {
		if opts.Isolated {
			return V(0, 0, 0)
		}
		return shadeFloor(scene, ray.At(planeT))
	}
	if opts.Isolated {
		return V(0.004, 0.005, 0.008)
	}
	return backdrop(ray.Direction)
}

func Render(scene Scene, camera Camera, opts RenderOptions) *image.RGBA {
	if opts.Width < 1 || opts.Height < 1 {
		panic("render: image dimensions must be positive")
	}
	samples := opts.Samples
	if samples < 1 {
		samples = 1
	}

	img := image.NewRGBA(image.Rect(0, 0, opts.Width, opts.Height))
	aspect := float64(opts.Width) / float64(opts.Height)
	rows := make(chan int, opts.Height)
	var wg sync.WaitGroup

	workers := runtime.GOMAXPROCS(0)
	if workers > opts.Height {
		workers = opts.Height
	}
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for y := range rows {
				renderRow(img, scene, camera, opts, aspect, samples, y)
			}
		}()
	}
	for y := 0; y < opts.Height; y++ {
		rows <- y
	}
	close(rows)
	wg.Wait()
	return img
}

func renderRow(img *image.RGBA, scene Scene, camera Camera, opts RenderOptions, aspect float64, samples, y int) {
	invSamples := 1 / float64(samples*samples)
	for x := 0; x < opts.Width; x++ {
		accum := V(0, 0, 0)
		for sy := 0; sy < samples; sy++ {
			for sx := 0; sx < samples; sx++ {
				u := 2*(float64(x)+(float64(sx)+0.5)/float64(samples))/float64(opts.Width) - 1
				v := 1 - 2*(float64(y)+(float64(sy)+0.5)/float64(samples))/float64(opts.Height)
				accum = accum.Add(trace(scene, camera.Ray(u, v, aspect), opts))
			}
		}
		accum = accum.Mul(invSamples)
		img.SetRGBA(x, y, encodeSRGB(accum, x, y))
	}
}

func encodeSRGB(linear Vec3, x, y int) color.RGBA {
	linear = linear.Clamp01()
	dither := float64(hash2D(x, y))/float64(^uint32(0)) - 0.5
	return color.RGBA{
		R: encodeChannel(linear.X + dither/255),
		G: encodeChannel(linear.Y + dither/255),
		B: encodeChannel(linear.Z + dither/255),
		A: 255,
	}
}

func encodeChannel(v float64) uint8 {
	v = Clamp01(v)
	if v <= 0.0031308 {
		v *= 12.92
	} else {
		v = 1.055*math.Pow(v, 1/2.4) - 0.055
	}
	return uint8(math.Round(v * 255))
}

func hash2D(x, y int) uint32 {
	h := uint32(x)*0x1f123bb5 ^ uint32(y)*0x9e3779b9
	h ^= h >> 16
	h *= 0x7feb352d
	h ^= h >> 15
	h *= 0x846ca68b
	h ^= h >> 16
	return h
}
