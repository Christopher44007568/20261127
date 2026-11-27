package furnace

// Kind identifies one of the three white-furnace specimens.
type Kind uint8

const (
	RoughMetal Kind = iota
	Clearcoat
	Velvet
)

func (k Kind) String() string {
	switch k {
	case RoughMetal:
		return "rough_metal"
	case Clearcoat:
		return "clearcoat"
	case Velvet:
		return "velvet"
	default:
		return "unknown"
	}
}

// Material contains only legal parameter ranges. The Raw* BRDFs deliberately
// combine those legal pieces without global energy bookkeeping; NormalizedBRDF
// applies the directional correction.
type Material struct {
	Kind Kind
	Name string

	// Albedo is the diffuse/base reflectance. For metals it also drives F0.
	Albedo Vec3

	// Roughness is the base/metal GGX roughness in [0.04,1].
	Roughness float64

	// Metallic selects conductor vs dielectric base response.
	Metallic float64

	// Coat and CoatRoughness describe the clear dielectric layer.
	Coat          float64
	CoatRoughness float64
	CoatF0        float64

	// Sheen and SheenRoughness control the velvet grazing/retro lobe.
	Sheen            float64
	SheenRoughness   float64
	SheenTint        Vec3
	SheenRetro       float64
	SheenGrazingGain float64
}

// RoughMetalPreset is a deliberately rough warm conductor.
func RoughMetalPreset() Material {
	return Material{
		Kind:      RoughMetal,
		Name:      "粗糙金属",
		Albedo:    V(0.82, 0.55, 0.28),
		Roughness: 0.48,
		Metallic:  1,
	}
}

// ClearcoatPreset is oxblood paint with a visible dielectric top coat.
func ClearcoatPreset() Material {
	return Material{
		Kind:             Clearcoat,
		Name:             "清漆",
		Albedo:           V(0.52, 0.035, 0.025),
		Roughness:        0.46,
		Metallic:         0,
		Coat:             1,
		CoatRoughness:    0.16,
		CoatF0:           0.04,
		SheenTint:        V(1, 1, 1),
		SheenRetro:       0.35,
		SheenGrazingGain: 1,
	}
}

// VelvetPreset is dark wine cloth with a strong grazing fiber response.
func VelvetPreset() Material {
	return Material{
		Kind:             Velvet,
		Name:             "天鹅绒",
		Albedo:           V(0.30, 0.025, 0.105),
		Roughness:        0.82,
		Metallic:         0,
		Sheen:            1.35,
		SheenRoughness:   0.58,
		SheenTint:        V(1.00, 0.34, 0.52),
		SheenRetro:       1.15,
		SheenGrazingGain: 1.65,
	}
}

// Presets returns the three required specimens.
func Presets() []Material {
	return []Material{RoughMetalPreset(), ClearcoatPreset(), VelvetPreset()}
}

// WhiteFurnaceMaterial returns the energy-test version: every allowed
// reflectance color is set to one while roughness/lobe shapes are retained.
func WhiteFurnaceMaterial(m Material) Material {
	w := m
	w.Albedo = V(1, 1, 1)
	w.SheenTint = V(1, 1, 1)
	if m.Kind == RoughMetal {
		// For a metal, white albedo is white F0.
		w.Metallic = 1
	}
	if m.Kind == Clearcoat || m.Kind == Velvet {
		// The dielectric F0 remains a physical IOR parameter. The base and
		// multiple-scattering correction supply the transmitted remainder.
		w.Metallic = 0
	}
	return w
}

func (m Material) f0() Vec3 {
	dielectric := V(0.04, 0.04, 0.04)
	metalF0 := m.Albedo
	return dielectric.Mul(1 - m.Metallic).Add(metalF0.Mul(m.Metallic))
}

func (m Material) alpha() float64 {
	r := clamp(m.Roughness, 0.035, 1)
	return clamp(r*r, 0.0015, 1)
}

func (m Material) coatAlpha() float64 {
	r := clamp(m.CoatRoughness, 0.02, 1)
	return clamp(r*r, 0.0015, 1)
}
