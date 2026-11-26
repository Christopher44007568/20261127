package render

import "math"

const (
	DefaultCoatingF0    = 0.04
	DefaultEnergyCap    = 0.85
	DefaultRoughness    = 0.12
	MinCoatingRoughness = 0.015
)

// Material is a diffuse substrate under a single dielectric clearcoat layer.
// There is deliberately no extra broadband specular lobe on the substrate:
// adding one on top of the coat is what makes wet colors look chalky.
type Material struct {
	Albedo           Vec3
	ClearcoatWeight  float64
	CoatingRoughness float64
	CoatingF0        float64
	EnergyCap        float64
}

// EnergySplit describes how incident light is divided by the clearcoat.
// Clearcoat is the reflected fraction; Underlayer is transmitted to the base.
type EnergySplit struct {
	Clearcoat  float64
	Underlayer float64
}

// SurfaceSample keeps both physical layers separate so render diagnostics can
// display the substrate and the reflection independently.
type SurfaceSample struct {
	Diffuse   Vec3
	Clearcoat Vec3
	Split     EnergySplit
}

func (s SurfaceSample) Composite() Vec3 {
	return s.Diffuse.Add(s.Clearcoat)
}

// SchlickFresnel returns unpolarized dielectric reflectance at cosTheta.
func SchlickFresnel(cosTheta, f0 float64) float64 {
	cosTheta = Clamp01(cosTheta)
	f0 = Clamp(f0, 0, 1)
	m := 1 - cosTheta
	m2 := m * m
	return f0 + (1-f0)*m2*m2*m
}

func (m Material) coatingF0() float64 {
	if m.CoatingF0 <= 0 {
		return DefaultCoatingF0
	}
	return Clamp(m.CoatingF0, 0, 1)
}

func (m Material) roughness() float64 {
	r := m.CoatingRoughness
	if r <= 0 {
		r = DefaultRoughness
	}
	return Clamp(r, MinCoatingRoughness, 1)
}

func (m Material) energyCap() float64 {
	c := m.EnergyCap
	if c <= 0 {
		c = DefaultEnergyCap
	}
	return Clamp(c, 0, 1)
}

// SplitEnergy allocates reflected energy to the coat and the remainder to the
// diffuse underlayer. The cap is applied before allocation, so the clearcoat
// can never consume more than EnergyCap, even at grazing incidence.
func (m Material) SplitEnergy(nDotV float64) EnergySplit {
	weight := Clamp01(m.ClearcoatWeight)
	if weight == 0 {
		return EnergySplit{Clearcoat: 0, Underlayer: 1}
	}

	reflected := weight * SchlickFresnel(nDotV, m.coatingF0())
	reflected = math.Min(reflected, m.energyCap())
	return EnergySplit{
		Clearcoat:  reflected,
		Underlayer: 1 - reflected,
	}
}

// Shade combines the diffuse underlayer and environment reflection. Inputs are
// irradiance/radiance values in [0,1], which keeps the two-layer result within
// the same energy budget after the split.
func (m Material) Shade(nDotV float64, diffuseLight, reflectedEnv Vec3) SurfaceSample {
	split := m.SplitEnergy(nDotV)
	baseLight := diffuseLight.Clamp01()
	coatLight := reflectedEnv.Clamp01()

	return SurfaceSample{
		Diffuse:   m.Albedo.Clamp01().MulVec(baseLight).Mul(split.Underlayer),
		Clearcoat: coatLight.Mul(split.Clearcoat),
		Split:     split,
	}
}

// RoughnessToSpread maps clearcoat roughness to the cone angle used when
// filtering the environment. Low roughness stays close to a mirror.
func RoughnessToSpread(roughness float64) float64 {
	r := Clamp(roughness, MinCoatingRoughness, 1)
	return 0.006 + 0.72*r*r
}
