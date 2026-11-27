package main

import "math"

type RenderConfig struct {
	SamplesPerPixel int
	SunSamples      int
	BoxSamples      int
	BlurPasses      int
}

// Renderer traces direct lighting for the anisotropic metal scene.
type Renderer struct {
	Scene *Scene
	Env   *Env
	Cam   *Camera
	LUT   *EnergyLUT
	Cfg   RenderConfig
}

func (rd *Renderer) pixel(nx, ny float64, rng *Rng) Vec3 {
	ray := rd.Cam.ray(nx, ny)
	hit, ok := rd.Scene.trace(ray)
	if !ok {
		return rd.Env.skyAt(ray.Dir)
	}
	return rd.shade(hit, ray, rng.float())
}

func (rd *Renderer) shade(hit Hit, ray Ray, seed float64) Vec3 {
	frame := surfaceFrame(hit)
	view := ray.Dir.Neg().Normalize()
	noV := clamp01(view.Dot(frame.N))

	effRough := effectiveRoughness(hit, frame, hit.Mat.Roughness)
	params := Params{
		Roughness:  effRough,
		Anisotropy: hit.Mat.Anisotropy,
		F0:         hit.Mat.F0,
	}

	acc := Vec3{}
	acc = acc.Add(rd.directLights(params, frame, view, hit, seed))
	acc = acc.Add(rd.ambient(params, frame, noV))

	return Vec3{
		clampf(safeMax0(acc.X), 0, 32),
		clampf(safeMax0(acc.Y), 0, 32),
		clampf(safeMax0(acc.Z), 0, 32),
	}
}

func (rd *Renderer) directLights(p Params, f Frame, view Vec3, hit Hit, seed float64) Vec3 {
	total := Vec3{}

	sun := rd.Env.Sun
	for i := 0; i < rd.Cfg.SunSamples; i++ {
		wi := sun.sample(i, rd.Cfg.SunSamples, seed)
		if wi.Dot(f.N) <= 1e-4 || rd.occluded(hit, wi) {
			continue
		}
		brdf := evalBRDF(p, f, wi, view, rd.LUT)
		weight := sun.Color.Scale(sun.Intensity * clamp01(wi.Dot(f.N)))
		total = total.Add(brdf.Mul(weight))
	}
	total = total.Scale(1 / float64(rd.Cfg.SunSamples))

	box := rd.Env.Box
	boxAcc := Vec3{}
	for i := 0; i < rd.Cfg.BoxSamples; i++ {
		wi := box.sample(i, rd.Cfg.BoxSamples, seed)
		if wi.Dot(f.N) <= 1e-4 || rd.occluded(hit, wi) {
			continue
		}
		brdf := evalBRDF(p, f, wi, view, rd.LUT)
		weight := box.Color.Scale(box.Intensity * clamp01(wi.Dot(f.N)))
		boxAcc = boxAcc.Add(brdf.Mul(weight))
	}
	total = total.Add(boxAcc.Scale(1 / float64(rd.Cfg.BoxSamples)))
	return total
}

// occluded traces a shadow ray; the surface bias avoids self-intersection.
func (rd *Renderer) occluded(hit Hit, wi Vec3) bool {
	origin := hit.Point.Add(hit.Normal.Scale(1e-3))
	shadow := Ray{Orig: origin, Dir: wi}
	for _, obj := range rd.Scene.Objects {
		if obj == hit.Self {
			continue
		}
		if _, blocked := obj.intersect(shadow); blocked {
			return true
		}
	}
	return false
}

// ambient approximates image-based lighting with the energy LUT, keeping the
// response non-negative at every grazing angle.
func (rd *Renderer) ambient(p Params, f Frame, noV float64) Vec3 {
	_, _, aeff := p.alphaAxes()
	eV := rd.LUT.eval(aeff, noV)
	eAvg := rd.LUT.avg(aeff)
	fAvg := p.F0.Scale(20.0 / 21.0).Add(Vec3{1.0 / 21, 1.0 / 21, 1.0 / 21})
	num := p.F0.Sub(fAvg.Scale(eAvg))
	base := math.Max(1e-6, (1-eAvg)*(1-eAvg))
	fmsCoef := Vec3{
		safeMax0(num.X) / base,
		safeMax0(num.Y) / base,
		safeMax0(num.Z) / base,
	}
	fv := fresnelSchlick(noV, p.F0)
	single := fv.Scale(eV)
	ms := fmsCoef.Scale(1 - eV)
	return single.Add(ms).Mul(rd.Env.average())
}

// acesToneMap is the Narkowicz ACES filmic approximation.
func acesToneMap(x float64) float64 {
	const a = 2.51
	const b = 0.03
	const c = 2.43
	const d = 0.59
	const e = 0.14
	return clamp01((x * (a*x + b)) / (x*(c*x+d) + e))
}

func linearToSRGB(x float64) float64 {
	if x <= 0.0031308 {
		return 12.92 * x
	}
	return 1.055*math.Pow(x, 1/2.4) - 0.055
}

// Render runs the full image and returns sRGB 8-bit pixels.
func (rd *Renderer) Render() []uint8 {
	w, h := rd.Cam.Width, rd.Cam.Height
	spp := rd.Cfg.SamplesPerPixel
	hdr := make([]float64, w*h*3)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			acc := Vec3{}
			for s := 0; s < spp; s++ {
				jx := (float64(s%2) + 0.5) / 2
				jy := (float64(s/2) + 0.5) / 2
				if spp == 1 {
					jx, jy = 0.5, 0.5
				}
				nx := ((float64(x)+jx)/float64(w))*2 - 1
				ny := (1-(float64(y)+jy)/float64(h))*2 - 1
				rng := newRng(uint64(x)*1000003 + uint64(y)*9176 + uint64(s)*77 + 17)
				acc = acc.Add(rd.pixel(nx, ny, rng))
			}
			col := acc.Scale(1 / float64(spp))
			idx := (y*w + x) * 3
			hdr[idx+0] = col.X
			hdr[idx+1] = col.Y
			hdr[idx+2] = col.Z
		}
	}
	for pass := 0; pass < rd.Cfg.BlurPasses; pass++ {
		hdr = gaussianBlur3x3(hdr, w, h)
	}
	out := make([]uint8, w*h*3)
	for i := 0; i < w*h; i++ {
		out[i*3+0] = color8(linearToSRGB(acesToneMap(hdr[i*3+0])))
		out[i*3+1] = color8(linearToSRGB(acesToneMap(hdr[i*3+1])))
		out[i*3+2] = color8(linearToSRGB(acesToneMap(hdr[i*3+2])))
	}
	return out
}

// gaussianBlur3x3 applies a separable-ish 2D Gaussian on the HDR buffer.
func gaussianBlur3x3(src []float64, w, h int) []float64 {
	dst := make([]float64, len(src))
	kernel := [3]float64{0.25, 0.5, 0.25}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			acc := [3]float64{}
			weight := 0.0
			for ky := -1; ky <= 1; ky++ {
				sy := y + ky
				if sy < 0 || sy >= h {
					continue
				}
				for kx := -1; kx <= 1; kx++ {
					sx := x + kx
					if sx < 0 || sx >= w {
						continue
					}
					kw := kernel[ky+1] * kernel[kx+1]
					idx := (sy*w + sx) * 3
					acc[0] += src[idx+0] * kw
					acc[1] += src[idx+1] * kw
					acc[2] += src[idx+2] * kw
					weight += kw
				}
			}
			di := (y*w + x) * 3
			dst[di+0] = acc[0] / weight
			dst[di+1] = acc[1] / weight
			dst[di+2] = acc[2] / weight
		}
	}
	return dst
}

func color8(x float64) uint8 {
	v := int(math.Round(clamp01(x) * 255))
	if v < 0 {
		v = 0
	}
	if v > 255 {
		v = 255
	}
	return uint8(v)
}
