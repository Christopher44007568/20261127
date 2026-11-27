package material

import "fmt"

// Tier identifies one of the three lighting approximations.
type Tier int

const (
	TierLow Tier = iota
	TierMedium
	TierHigh
)

func (t Tier) String() string {
	switch t {
	case TierLow:
		return "low"
	case TierMedium:
		return "medium"
	case TierHigh:
		return "high"
	default:
		return fmt.Sprintf("tier-%d", int(t))
	}
}

func ParseTier(s string) (Tier, error) {
	switch s {
	case "low":
		return TierLow, nil
	case "medium", "mid":
		return TierMedium, nil
	case "high":
		return TierHigh, nil
	default:
		return 0, fmt.Errorf("unknown tier %q", s)
	}
}

// Material is the single parameter set shared by every lighting tier.
// Tiers may ignore expensive terms, but must not replace identity fields.
type Material struct {
	ID              string  `json:"id"`
	Label           string  `json:"label"`
	Kind            string  `json:"kind"`
	Version         int     `json:"version"`
	BaseColor       Color   `json:"base_color"`
	Metallic        float64 `json:"metallic"`
	Roughness       float64 `json:"roughness"`
	RoughnessMin    float64 `json:"roughness_min"`
	RoughnessMax    float64 `json:"roughness_max"`
	Specular        float64 `json:"specular"`
	Wrap            float64 `json:"wrap"`
	SheenColor      Color   `json:"sheen_color"`
	SheenWeight     float64 `json:"sheen_weight"`
	SheenRoughness  float64 `json:"sheen_roughness"`
	ClearcoatWeight float64 `json:"clearcoat_weight"`
	ClearcoatRough  float64 `json:"clearcoat_roughness"`
}

// LowUniforms is the intentionally small uniform block for the low tier.
// It retains material identity: base color, metalness, nominal roughness,
// and the allowed roughness interval.
type LowUniforms struct {
	ID           string  `json:"id"`
	BaseColor    Color   `json:"base_color"`
	Metallic     float64 `json:"metallic"`
	Roughness    float64 `json:"roughness"`
	RoughnessMin float64 `json:"roughness_min"`
	RoughnessMax float64 `json:"roughness_max"`
}

func (m Material) LowUniforms() LowUniforms {
	return LowUniforms{
		ID:           m.ID,
		BaseColor:    m.BaseColor,
		Metallic:     m.Metallic,
		Roughness:    m.Roughness,
		RoughnessMin: m.RoughnessMin,
		RoughnessMax: m.RoughnessMax,
	}
}

// EffectiveRoughness keeps all tiers inside the shared roughness interval.
func (m Material) EffectiveRoughness() float64 {
	return clamp(m.Roughness, m.RoughnessMin, m.RoughnessMax)
}

func mustHex(s string) Color {
	c, err := HexColor(s)
	if err != nil {
		panic(err)
	}
	return c
}

// DefaultMaterials returns metal, cloth, and plastic presets.
func DefaultMaterials() []Material {
	return []Material{
		{
			ID:              "copper_metal",
			Label:           "金属 / 铜",
			Kind:            "metal",
			Version:         1,
			BaseColor:       mustHex("#C87942"),
			Metallic:        1,
			Roughness:       0.27,
			RoughnessMin:    0.16,
			RoughnessMax:    0.43,
			Specular:        0.50,
			Wrap:            0.00,
			SheenColor:      Color{R: 1, G: 0.72, B: 0.48},
			SheenWeight:     0,
			SheenRoughness:  0.60,
			ClearcoatWeight: 0.08,
			ClearcoatRough:  0.20,
		},
		{
			ID:              "woven_cloth",
			Label:           "布 / 莓红编织",
			Kind:            "cloth",
			Version:         1,
			BaseColor:       mustHex("#9B3F5D"),
			Metallic:        0,
			Roughness:       0.84,
			RoughnessMin:    0.69,
			RoughnessMax:    0.96,
			Specular:        0.32,
			Wrap:            0.58,
			SheenColor:      mustHex("#E8AD92"),
			SheenWeight:     0.42,
			SheenRoughness:  0.72,
			ClearcoatWeight: 0,
			ClearcoatRough:  0.50,
		},
		{
			ID:              "blue_plastic",
			Label:           "塑料 / 亮蓝",
			Kind:            "plastic",
			Version:         1,
			BaseColor:       mustHex("#1769E0"),
			Metallic:        0,
			Roughness:       0.36,
			RoughnessMin:    0.22,
			RoughnessMax:    0.58,
			Specular:        0.56,
			Wrap:            0.08,
			SheenColor:      Color{R: 0.70, G: 0.85, B: 1.00},
			SheenWeight:     0,
			SheenRoughness:  0.50,
			ClearcoatWeight: 0.62,
			ClearcoatRough:  0.16,
		},
	}
}

func ByID(materials []Material, id string) (Material, bool) {
	for _, m := range materials {
		if m.ID == id {
			return m, true
		}
	}
	return Material{}, false
}
