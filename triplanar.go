package main

import "math"

const (
	AxisX = iota
	AxisY
	AxisZ
)

const (
	DefaultWorldScale = 0.55
	DefaultHardness   = 4.0
)

// Texture is a repeatable, linear-light 2D texture sampler.
type Texture interface {
	Sample(u, v float64) Vec3
}

// TextureFunc adapts a function to Texture.
type TextureFunc func(u, v float64) Vec3

func (f TextureFunc) Sample(u, v float64) Vec3 { return f(u, v) }

// Config contains the world-space triplanar material controls.
//
// WorldScale converts world units to texture repeats. Hardness is the exponent
// applied to absolute normal components before normalization. Larger values
// make transitions between projections narrower.
type Config struct {
	Texture    Texture
	WorldScale float64
	Hardness   float64
	Offset     Vec3
}

// Triplanar is the reusable sampler.
type Triplanar struct {
	Config Config
}

func NewTriplanar(texture Texture, worldScale, hardness float64) Triplanar {
	return Triplanar{Config: Config{
		Texture:    texture,
		WorldScale: worldScale,
		Hardness:   hardness,
	}}
}

func DefaultConfig(texture Texture) Config {
	return Config{
		Texture:    texture,
		WorldScale: DefaultWorldScale,
		Hardness:   DefaultHardness,
	}
}

func (c Config) withDefaults() Config {
	out := c
	if out.Texture == nil {
		out.Texture = NewRockTexture(256, 17)
	}
	if out.WorldScale <= 0 {
		out.WorldScale = DefaultWorldScale
	}
	if out.Hardness <= 0 {
		out.Hardness = DefaultHardness
	}
	return out
}

func (t Triplanar) WithDefaults() Triplanar {
	t.Config = t.Config.withDefaults()
	return t
}

// NormalWeights returns three positive weights summing to one.
func NormalWeights(normal Vec3, hardness float64) Vec3 {
	n := normal.Normalize().Abs()
	power := hardness
	if power < 1 {
		power = 1
	}

	wx := math.Pow(n.X, power)
	wy := math.Pow(n.Y, power)
	wz := math.Pow(n.Z, power)
	sum := wx + wy + wz
	if sum <= 1e-12 {
		return V(1.0/3.0, 1.0/3.0, 1.0/3.0)
	}
	return V(wx/sum, wy/sum, wz/sum)
}

func (t Triplanar) Weights(normal Vec3) Vec3 {
	cfg := t.Config.withDefaults()
	return NormalWeights(normal, cfg.Hardness)
}

// ProjectionUV maps a world position to one planar texture coordinate set.
// The gravity coordinate is always the second coordinate on X/Z projections.
func ProjectionUV(p Vec3, axis int, worldScale float64) (u, v float64) {
	s := worldScale
	switch axis {
	case AxisX:
		return p.Z * s, p.Y * s
	case AxisY:
		return p.X * s, p.Z * s
	default:
		return p.X * s, p.Y * s
	}
}

func (t Triplanar) projectionSample(p Vec3, axis int) Vec3 {
	cfg := t.Config.withDefaults()
	q := p.Add(cfg.Offset)
	u, v := ProjectionUV(q, axis, cfg.WorldScale)
	return cfg.Texture.Sample(u, v)
}

// Sample evaluates the normalized weighted projection average in linear light.
func (t Triplanar) Sample(p, normal Vec3) Vec3 {
	cfg := t.Config.withDefaults()
	weights := NormalWeights(normal, cfg.Hardness)

	cx := t.projectionSample(p, AxisX)
	cy := t.projectionSample(p, AxisY)
	cz := t.projectionSample(p, AxisZ)

	blended := V(
		cx.X*weights.X+cy.X*weights.Y+cz.X*weights.Z,
		cx.Y*weights.X+cy.Y*weights.Y+cz.Y*weights.Z,
		cx.Z*weights.X+cy.Z*weights.Y+cz.Z*weights.Z,
	)
	return V(clamp01(blended.X), clamp01(blended.Y), clamp01(blended.Z))
}

type baseSizedTexture interface {
	BaseSize() int
}

func projectionFootprintLOD(texture Texture, axis int, scale float64, dpdx, dpdy Vec3) float64 {
	s := scale
	var dudx, dvdx, dudy, dvdy float64
	switch axis {
	case AxisX:
		dudx, dvdx = dpdx.Z*s, dpdx.Y*s
		dudy, dvdy = dpdy.Z*s, dpdy.Y*s
	case AxisY:
		dudx, dvdx = dpdx.X*s, dpdx.Z*s
		dudy, dvdy = dpdy.X*s, dpdy.Z*s
	default:
		dudx, dvdx = dpdx.X*s, dpdx.Y*s
		dudy, dvdy = dpdy.X*s, dpdy.Y*s
	}
	footprintUV := math.Max(
		math.Sqrt(dudx*dudx+dvdx*dvdx),
		math.Sqrt(dudy*dudy+dvdy*dvdy),
	)
	sized, ok := texture.(baseSizedTexture)
	if !ok || footprintUV == 0 {
		return 0
	}
	return math.Log2(math.Max(1, footprintUV*float64(sized.BaseSize())))
}

// SampleGrad evaluates triplanar sampling with screen/world derivatives so
// mipmapped textures can suppress minification aliasing.
func (t Triplanar) SampleGrad(p, normal, dpdx, dpdy Vec3) Vec3 {
	cfg := t.Config.withDefaults()
	q := p.Add(cfg.Offset)
	weights := NormalWeights(normal, cfg.Hardness)

	sampleAxis := func(axis int) Vec3 {
		u, v := ProjectionUV(q, axis, cfg.WorldScale)
		lod := projectionFootprintLOD(cfg.Texture, axis, cfg.WorldScale, dpdx, dpdy)
		return sampleTextureLOD(cfg.Texture, u, v, lod)
	}

	cx := sampleAxis(AxisX)
	cy := sampleAxis(AxisY)
	cz := sampleAxis(AxisZ)
	blended := V(
		cx.X*weights.X+cy.X*weights.Y+cz.X*weights.Z,
		cx.Y*weights.X+cy.Y*weights.Y+cz.Y*weights.Z,
		cx.Z*weights.X+cy.Z*weights.Y+cz.Z*weights.Z,
	)
	return V(clamp01(blended.X), clamp01(blended.Y), clamp01(blended.Z))
}

// Sample is the package-level convenience wrapper.
func SampleTriplanar(cfg Config, p, normal Vec3) Vec3 {
	return Triplanar{Config: cfg}.Sample(p, normal)
}

func SampleTriplanarGrad(cfg Config, p, normal, dpdx, dpdy Vec3) Vec3 {
	return Triplanar{Config: cfg}.SampleGrad(p, normal, dpdx, dpdy)
}
