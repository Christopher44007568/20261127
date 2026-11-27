package main

import "math"

// DirectionalLight is a small angular disc (sun) sampled as an area light.
type DirectionalLight struct {
	Dir        Vec3 // direction from surface toward the light
	Color      Vec3
	Intensity  float64
	AngularRad float64
	Samples    int
}

// SoftBox is an elongated rectangular area light parameterized on the
// directional sphere; its long axis stretches the anisotropic highlight.
type SoftBox struct {
	Center    Vec3
	AxisU     Vec3 // long axis (unit)
	AxisV     Vec3 // short axis (unit)
	HalfU     float64
	HalfV     float64
	Color     Vec3
	Intensity float64
	Samples   int
}

type Env struct {
	Top    Vec3
	Bottom Vec3
	Sun    *DirectionalLight
	Box    *SoftBox
}

// skyAt evaluates the procedural vertical-gradient environment for a ray.
func (e *Env) skyAt(dir Vec3) Vec3 {
	t := clamp01(dir.Y*0.5 + 0.5)
	return Vec3{
		mix(e.Bottom.X, e.Top.X, t),
		mix(e.Bottom.Y, e.Top.Y, t),
		mix(e.Bottom.Z, e.Top.Z, t),
	}
}

func (e *Env) average() Vec3 {
	return e.Top.Scale(0.35).Add(e.Bottom.Scale(0.65))
}

// sampleSun returns one golden-angle low-discrepancy direction inside the
// disc. The pixelSeed rotates the pattern per pixel to avoid grid aliasing;
// the HDR Gaussian blur then removes the residual high-frequency grain.
func (l *DirectionalLight) sample(index, total int, pixelSeed float64) Vec3 {
	base := l.Dir.Normalize()
	bs := basisFromNormal(base)
	golden := math.Pi * (3 - math.Sqrt(5))
	f := (float64(index) + 0.5) / float64(total)
	angle := float64(index)*golden + pixelSeed*golden
	r := math.Sqrt(f) * l.AngularRad
	offset := bs.t.Scale(r * math.Cos(angle)).Add(bs.b.Scale(r * math.Sin(angle)))
	return base.Add(offset).Normalize()
}

// sampleBox returns one golden-ratio stratified direction toward the box.
func (b *SoftBox) sample(index, total int, pixelSeed float64) Vec3 {
	golden := 0.61803398875
	fx := math.Mod(float64(index)*golden+pixelSeed, 1)
	fy := math.Mod(float64(index)*golden*golden+pixelSeed*1.7, 1)
	u := (fx*2 - 1) * b.HalfU
	v := (fy*2 - 1) * b.HalfV
	return b.Center.Add(b.AxisU.Scale(u)).Add(b.AxisV.Scale(v)).Normalize()
}
