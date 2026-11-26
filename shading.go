package skinsss

// Material and lighting constants. All values are linear-space RGB.
var (
	// SkinAlbedo is the diffuse reflectance of light skin (no global SSS tint:
	// scattering is derived per-pixel from geometry and light direction).
	SkinAlbedo = Vec3{0.78, 0.55, 0.45}
	LightColor = Vec3{1.0, 0.96, 0.90}
	Ambient    = 0.06

	// Extinction coefficients per unit distance. The dermis (deep layer)
	// attenuates green/blue much harder than red, producing the warm
	// translucency of thin ears; the epidermis adds a shallow yellowing.
	SigmaShallow = Vec3{0.60, 1.30, 1.90}
	SigmaDeep    = Vec3{0.35, 1.90, 3.10}

	// Tint each layer contributes when photons re-emerge.
	TintShallow = Vec3{1.0, 0.72, 0.58}
	TintDeep    = Vec3{1.0, 0.40, 0.34}
)

// ShadeInput bundles everything the BRDF/BTDF model needs for one sample.
type ShadeInput struct {
	Normal     Vec3    // outward unit normal
	View       Vec3    // unit vector toward the camera
	Light      Vec3    // unit vector toward the light
	Thickness  float64 // chord length along the photon path (world units)
	Curvature  float64 // mean curvature, convex > 0
	SSSEnabled bool
}

// layerTransmittance applies Beer-Lambert extinction through one layer.
func layerTransmittance(sigma Vec3, depth float64) Vec3 {
	return Vec3{
		X: exp(-sigma.X * depth),
		Y: exp(-sigma.Y * depth),
		Z: exp(-sigma.Z * depth),
	}
}

// diffuseIrradiance evaluates a wrapped Lambert term. Wrap is only non-zero
// with SSS enabled, simulating light bleeding around curvature; the fallback
// (SSS off) is exact Lambert with max(nl, 0).
func diffuseIrradiance(in ShadeInput) float64 {
	nl := in.Normal.Dot(in.Light)
	if !in.SSSEnabled {
		return clamp(nl, 0, 1)
	}
	wrap := 0.18 + 0.27*curvatureFactor(in.Curvature)
	return clamp((nl+wrap)/(1+wrap), 0, 1)
}

// curvatureFactor maps mean curvature into [0,1] so thin, strongly curved
// regions (ears, rim) scatter through a shorter effective path.
func curvatureFactor(k float64) float64 {
	return smoothstep(0.2, 1.6, k)
}

// transmission evaluates the two-layer subsurface BTDF. The effective path is
// shortened by curvature, the two layers are independently attenuated with
// Beer-Lambert, and their re-emitted tints are mixed by depth. Everything is
// multiplicative — there is no additive emissive term.
func transmission(in ShadeInput) Vec3 {
	cf := curvatureFactor(in.Curvature)
	// The shallow (epidermis) layer is a roughly fixed thin shell. Curvature
	// only shortens it slightly. The deep (dermis) layer absorbs over the full
	// remaining chord, which is why thick cheeks go dark and red while thin
	// ears stay bright.
	shallowDepth := 0.09 - 0.03*cf
	deepDepth := in.Thickness - shallowDepth
	if deepDepth < 0 {
		deepDepth = 0
	}

	tShallow := layerTransmittance(SigmaShallow, shallowDepth)
	tDeep := layerTransmittance(SigmaDeep, deepDepth)

	// More curvature / thin geometry raises the relative shallow contribution.
	shallowWeight := 0.25 + 0.50*cf
	deepWeight := 1 - shallowWeight

	color := TintShallow.Mul(tShallow).Scale(shallowWeight).
		Add(TintDeep.Mul(tDeep).Scale(deepWeight))
	return color.Clamp01()
}

// Shade evaluates the full skin shading model and returns a linear-space RGB
// value in [0,1].
func Shade(in ShadeInput) Vec3 {
	// Ambient: flat albedo-bounded fill, present in both modes.
	ambient := SkinAlbedo.Scale(Ambient)

	if !in.SSSEnabled {
		// Pure Lambert diffuse; no transmission or wrapping at all.
		diff := clamp(in.Normal.Dot(in.Light), 0, 1)
		return ambient.Add(SkinAlbedo.Mul(LightColor).Scale(diff)).Clamp01()
	}

	// Diffuse (front-side response). The wrapped term fades smoothly across
	// the terminator so it hands over to transmission rather than stacking.
	diffIrr := diffuseIrradiance(in)
	diffTerm := SkinAlbedo.Mul(LightColor).Scale(diffIrr)

	// Transmission is gated by geometry: a photon chord must exist, and the
	// weight follows a grazing-angle envelope. Because the surface normal
	// turns toward the camera at the rim, grazing is high at the thin rim and
	// near zero at the thick center, so the bright-but-attenuated thin edge
	// always reads brighter than the dark, heavily absorbed center.
	grazing := 1 - clamp(abs(in.Normal.Dot(in.View)), 0, 1)
	chordGate := smoothstep(0.02, 0.12, in.Thickness)
	rim := smoothstep(0.25, 0.95, grazing)
	transWeight := chordGate * clamp(0.20+0.90*rim, 0, 1)
	transTerm := transmission(in).Mul(LightColor).Scale(transWeight)

	return ambient.Add(diffTerm).Add(transTerm).Clamp01()
}

// Heatmap color stops: thin (blue) -> cyan -> green -> yellow -> red (thick).
var heatStops = []struct {
	at  float64
	col Vec3
}{
	{0.0, Vec3{0.10, 0.15, 0.55}},
	{0.25, Vec3{0.15, 0.65, 0.85}},
	{0.5, Vec3{0.30, 0.80, 0.35}},
	{0.75, Vec3{0.95, 0.80, 0.20}},
	{1.0, Vec3{0.85, 0.15, 0.10}},
}

// HeatmapColor maps a normalized thickness in [0,1] to the heat ramp.
func HeatmapColor(normalized float64) Vec3 {
	t := clamp01(normalized)
	for i := 0; i < len(heatStops)-1; i++ {
		a := heatStops[i]
		b := heatStops[i+1]
		if t <= b.at {
			u := (t - a.at) / (b.at - a.at)
			return Lerp(a.col, b.col, u)
		}
	}
	return heatStops[len(heatStops)-1].col
}
