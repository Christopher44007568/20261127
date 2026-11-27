package material

import "math"

var (
	keyLightDir  = Vec3{X: 0.38, Y: 0.58, Z: 0.72}.Normalize()
	fillLightDir = Vec3{X: -0.72, Y: 0.18, Z: 0.52}.Normalize()
	keyRadiance  = Color{R: 2.35, G: 2.18, B: 1.95}
	fillRadiance = Color{R: 0.15, G: 0.19, B: 0.30}
)

// LODWeights are the crossfade weights for continuous level-of-detail.
type LODWeights struct {
	Low    float64 `json:"low"`
	Medium float64 `json:"medium"`
	High   float64 `json:"high"`
}

// WeightsAt maps lod in [0,2] to low/medium/high weights.
// Boundaries are at 0.5 and 1.5. A symmetric smooth crossfade replaces a
// hard switch in [boundary-width, boundary+width].
func WeightsAt(lod, width float64) LODWeights {
	lod = clamp(lod, 0, 2)
	if width <= 0 {
		switch {
		case lod < 0.5:
			return LODWeights{Low: 1}
		case lod < 1.5:
			return LODWeights{Medium: 1}
		default:
			return LODWeights{High: 1}
		}
	}

	w := LODWeights{}
	switch {
	case lod <= 0.5-width:
		w.Low = 1
	case lod < 0.5+width:
		t := smoothStep(0.5-width, 0.5+width, lod)
		w.Low = 1 - t
		w.Medium = t
	case lod <= 1.5-width:
		w.Medium = 1
	case lod < 1.5+width:
		t := smoothStep(1.5-width, 1.5+width, lod)
		w.Medium = 1 - t
		w.High = t
	default:
		w.High = 1
	}
	return w
}

func (w LODWeights) Blend(low, medium, high Color) Color {
	return Color{
		R: w.Low*low.R + w.Medium*medium.R + w.High*high.R,
		G: w.Low*low.G + w.Medium*medium.G + w.High*high.G,
		B: w.Low*low.B + w.Medium*medium.B + w.High*high.B,
	}
}

func fresnelSchlick(cosTheta float64, f0 Color) Color {
	f := math.Pow(clamp01(1-cosTheta), 5)
	return Color{
		R: f0.R + (1-f0.R)*f,
		G: f0.G + (1-f0.G)*f,
		B: f0.B + (1-f0.B)*f,
	}
}

func distributionGGX(n, h Vec3, roughness float64) float64 {
	a := roughness * roughness
	a2 := a * a
	ndh := max0(n.Dot(h))
	ndh2 := ndh * ndh
	den := ndh2*(a2-1) + 1
	return a2 / (math.Pi * den * den)
}

func geometrySchlickGGX(ndv, roughness float64) float64 {
	r := roughness + 1
	k := (r * r) / 8
	return ndv / (ndv*(1-k) + k + 1e-6)
}

func geometrySmith(n, v, l Vec3, roughness float64) float64 {
	ndv := max0(n.Dot(v))
	ndl := max0(n.Dot(l))
	return geometrySchlickGGX(ndv, roughness) * geometrySchlickGGX(ndl, roughness)
}

func max0(x float64) float64 {
	if x < 0 {
		return 0
	}
	return x
}

func f0For(m Material) Color {
	dielectric := 0.032 + 0.028*m.Specular
	return Color{
		R: mix(dielectric, m.BaseColor.R, m.Metallic),
		G: mix(dielectric, m.BaseColor.G, m.Metallic),
		B: mix(dielectric, m.BaseColor.B, m.Metallic),
	}
}
