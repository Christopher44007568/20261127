package main

import (
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"
)

type RenderMode int

const (
	ReferenceMode RenderMode = iota
	FroxelMode
)

type RenderConfig struct {
	Width       int
	Height      int
	Samples     int
	Mode        RenderMode
	Fog         bool
	FogSteps    int
	Exposure    float64
	WorkerCount int
}

type RenderStats struct {
	PrimaryRays      int64
	SurfaceFragments int64
	SurfaceFallback  int64
	VolumeSamples    int64
	VolumeFallback   int64
	AverageLights    float64
	MaxLightsVisited int
}

type RenderResult struct {
	Image *image.RGBA
	Stats RenderStats
}

type pixelInfo struct {
	X, Y          float64
	Width, Height float64
	Grid          *FroxelGrid
	Mode          RenderMode
}

func DefaultRenderConfig() RenderConfig {
	return RenderConfig{
		Width:       960,
		Height:      540,
		Samples:     4,
		Mode:        ReferenceMode,
		Fog:         true,
		FogSteps:    32,
		Exposure:    1.05,
		WorkerCount: runtime.NumCPU(),
	}
}

func RenderScene(scene *Scene, camera *Camera, grid *FroxelGrid, cfg RenderConfig) RenderResult {
	cfg.Width = max(cfg.Width, 1)
	cfg.Height = max(cfg.Height, 1)
	cfg.Samples = max(cfg.Samples, 1)
	cfg.FogSteps = max(cfg.FogSteps, 1)
	if cfg.WorkerCount <= 0 {
		cfg.WorkerCount = runtime.NumCPU()
	}
	img := image.NewRGBA(image.Rect(0, 0, cfg.Width, cfg.Height))
	perWorker := make([]RenderStats, cfg.WorkerCount)
	var wg sync.WaitGroup
	nextRow := make(chan int, cfg.Height)
	for y := 0; y < cfg.Height; y++ {
		nextRow <- y
	}
	close(nextRow)
	for worker := 0; worker < cfg.WorkerCount; worker++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			stats := &perWorker[id]
			for y := range nextRow {
				for x := 0; x < cfg.Width; x++ {
					c := renderPixel(scene, camera, grid, cfg, x, y, stats)
					putSRGB(img, x, y, c)
				}
			}
		}(worker)
	}
	wg.Wait()
	var total RenderStats
	var weighted int64
	for i := range perWorker {
		s := perWorker[i]
		total.PrimaryRays += s.PrimaryRays
		total.SurfaceFragments += s.SurfaceFragments
		total.SurfaceFallback += s.SurfaceFallback
		total.VolumeSamples += s.VolumeSamples
		total.VolumeFallback += s.VolumeFallback
		total.MaxLightsVisited = max(total.MaxLightsVisited, s.MaxLightsVisited)
		weighted += int64(s.AverageLights)
	}
	if total.SurfaceFragments > 0 {
		total.AverageLights = float64(weighted) / float64(total.SurfaceFragments)
	}
	return RenderResult{Image: img, Stats: total}
}

func renderPixel(scene *Scene, camera *Camera, grid *FroxelGrid, cfg RenderConfig, x, y int, stats *RenderStats) Vec3 {
	offsets := sampleOffsets(cfg.Samples)
	var sum Vec3
	for _, o := range offsets {
		px := float64(x) + o[0]
		py := float64(y) + o[1]
		ray := camera.RayAt(px, py, float64(cfg.Width), float64(cfg.Height))
		info := pixelInfo{
			X:      px,
			Y:      py,
			Width:  float64(cfg.Width),
			Height: float64(cfg.Height),
			Grid:   grid,
			Mode:   cfg.Mode,
		}
		sum = sum.Add(traceRay(scene, camera, ray, info, cfg, stats))
		stats.PrimaryRays++
	}
	return sum.Div(float64(len(offsets)))
}

func sampleOffsets(count int) [][2]float64 {
	if count == 1 {
		return [][2]float64{{0.5, 0.5}}
	}
	if count == 2 {
		return [][2]float64{{0.25, 0.25}, {0.75, 0.75}}
	}
	offsets := make([][2]float64, 0, count)
	grid := int(math.Ceil(math.Sqrt(float64(count))))
	for i := 0; i < count; i++ {
		ix := i % grid
		iy := (i / grid) % grid
		offsets = append(offsets, [2]float64{
			(float64(ix) + 0.5) / float64(grid),
			(float64(iy) + 0.5) / float64(grid),
		})
	}
	return offsets
}

func traceRay(scene *Scene, camera *Camera, ray Ray, info pixelInfo, cfg RenderConfig, stats *RenderStats) Vec3 {
	hit, ok := scene.Intersect(ray)
	if !ok {
		c := background(ray.Direction)
		if cfg.Fog {
			scattered, trans := integrateVolume(scene, camera, ray, math.Inf(1), info, cfg, stats)
			c = c.Mul(trans).Add(scattered)
		}
		return c
	}
	surface := shadeSurface(scene, camera, ray, hit, info, stats)
	if !cfg.Fog || scene.Volume == nil {
		return surface
	}
	scattered, trans := integrateVolume(scene, camera, ray, hit.T, info, cfg, stats)
	return surface.Mul(trans).Add(scattered)
}

func background(dir Vec3) Vec3 {
	t := clamp(dir.Y*0.5+0.5, 0, 1)
	return V(0.006, 0.008, 0.016).Lerp(V(0.025, 0.035, 0.065), t)
}

func shadeSurface(scene *Scene, camera *Camera, ray Ray, hit Hit, info pixelInfo, stats *RenderStats) Vec3 {
	n := hit.Normal.Normalize()
	viewDir := ray.Direction.Neg()
	if n.Dot(viewDir) < 0 {
		n = n.Neg()
	}
	albedo := hit.Material.Albedo
	if hit.Material.Checker > 0 {
		scale := hit.Material.Checker
		cx := int(math.Floor(hit.Point.X / scale))
		cz := int(math.Floor(hit.Point.Z / scale))
		if (cx+cz)&1 == 0 {
			albedo = albedo.Mul(1.42)
		} else {
			albedo = albedo.Mul(0.62)
		}
	}
	ambient := V(0.027, 0.033, 0.045)
	diffuse := Vec3{}
	specular := Vec3{}
	viewZ := camera.ViewZ(hit.Point)
	lights, _, fallback := queryLights(scene, info, viewZ)
	stats.SurfaceFragments++
	if fallback != NoFallback && info.Mode == FroxelMode {
		stats.SurfaceFallback++
	}
	stats.AverageLights += float64(len(lights))
	stats.MaxLightsVisited = max(stats.MaxLightsVisited, len(lights))
	for _, lightIndex := range lights {
		light := scene.Lights[lightIndex]
		toLight := light.Position.Sub(hit.Point)
		distance := toLight.Length()
		if distance < 1e-6 {
			continue
		}
		l := toLight.Div(distance)
		nDotL := n.Dot(l)
		if nDotL <= 0 {
			continue
		}
		att := light.Attenuation(hit.Point)
		if att <= 0 {
			continue
		}
		shadowOrigin := hit.Point.Add(n.Mul(0.002))
		if scene.Occluded(Ray{Origin: shadowOrigin, Direction: l}, distance-0.004) {
			continue
		}
		diffuse = diffuse.Add(light.Color.Mul(att * nDotL))
		h := l.Add(viewDir).Normalize()
		spec := math.Pow(math.Max(n.Dot(h), 0), hit.Material.Shininess)
		specular = specular.Add(light.Color.Mul(att * spec))
	}
	result := ambient.MulVec(albedo).
		Add(albedo.MulVec(diffuse)).
		Add(specular.Mul(hit.Material.Specular * 0.45)).
		Add(hit.Material.Emissive)
	return result
}

func queryLights(scene *Scene, info pixelInfo, viewZ float64) ([]int, int, FallbackReason) {
	if info.Mode == ReferenceMode {
		if info.Grid != nil {
			return info.Grid.AllLights, -1, NoFallback
		}
		all := make([]int, len(scene.Lights))
		for i := range all {
			all[i] = i
		}
		return all, -1, NoFallback
	}
	return info.Grid.Query(info.X, info.Y, info.Width, info.Height, viewZ)
}

func putSRGB(img *image.RGBA, x, y int, linear Vec3) {
	r := toneMap(linear.X)
	g := toneMap(linear.Y)
	b := toneMap(linear.Z)
	img.SetRGBA(x, y, color.RGBA{R: toSRGB(r), G: toSRGB(g), B: toSRGB(b), A: 255})
}

func toneMap(x float64) float64 {
	x = math.Max(x, 0)
	return clamp((x*(2.51*x+0.03))/(x*(2.43*x+0.59)+0.14), 0, 1)
}

func toSRGB(x float64) uint8 {
	x = clamp(x, 0, 1)
	if x <= 0.0031308 {
		x *= 12.92
	} else {
		x = 1.055*math.Pow(x, 1/2.4) - 0.055
	}
	return uint8(clamp(x*255+0.5, 0, 255))
}
