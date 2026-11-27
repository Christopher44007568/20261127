package flakepaint

import "math"

// Paint couples the flake field to a metallic basecoat and clearcoat.
type Paint struct {
	BaseColor         Vec3
	SpecularColor     Vec3
	BaseRoughness     float64
	ClearcoatRough    float64
	ClearcoatStrength float64
	Flakes            FlakeField
}

func DefaultPaint(seed uint64) Paint {
	return Paint{
		BaseColor:         V(0.012, 0.038, 0.088),
		SpecularColor:     V(0.080, 0.185, 0.340),
		BaseRoughness:     0.24,
		ClearcoatRough:    0.028,
		ClearcoatStrength: 1.02,
		Flakes:            DefaultFlakeField(seed),
	}
}

type Light struct {
	Dir       Vec3
	Color     Vec3
	Intensity float64
	Width     float64
}

var studioLightSet = []Light{
	{Dir: V(-0.48, 0.67, 0.57).Normalize(), Color: V(1.0, 0.77, 0.52), Intensity: 4.2, Width: 0.18},
	{Dir: V(0.72, 0.12, 0.68).Normalize(), Color: V(0.46, 0.73, 1.0), Intensity: 2.8, Width: 0.22},
	{Dir: V(-0.83, -0.12, 0.55).Normalize(), Color: V(0.70, 0.88, 1.0), Intensity: 1.9, Width: 0.12},
}

var glintLightSet = []Light{
	{Dir: V(-0.17, 0.20, 0.97).Normalize(), Color: V(1.00, 0.76, 0.48), Intensity: 0.88},
	{Dir: V(0.28, 0.10, 0.95).Normalize(), Color: V(0.42, 0.72, 1.00), Intensity: 0.68},
	{Dir: V(-0.02, -0.21, 0.98).Normalize(), Color: V(0.84, 0.94, 1.00), Intensity: 0.46},
}

func StudioLights() []Light {
	return studioLightSet
}

func strip(center, width, x float64) float64 {
	w := math.Max(width, 1e-4)
	z := (x - center) / w
	return math.Exp(-0.5 * z * z)
}

// studioEnvironment is a prefiltered procedural softbox environment. Blur is
// driven by the same roughness as the BRDF, so aggregated flakes cannot alias.
func studioEnvironment(d Vec3, blur float64) Vec3 {
	d = d.Normalize()
	if d.Length() < 0.5 {
		return V(0, 0, 0)
	}
	blur = clamp(blur, 0, 0.65)

	up := smoothstep(-0.62, 0.82, d.Y)
	base := V(0.0025, 0.0045, 0.0080).Lerp(V(0.016, 0.034, 0.058), up)
	base = base.Add(V(0.025, 0.035, 0.050).Mul(smoothstep(0.55, 1.0, d.Y)))

	// Overhead strip: the dominant continuous band at distance.
	wH := 0.037 + blur*0.38
	hBand := math.Exp(-0.5 * math.Pow((d.Y-0.46)/wH, 2))
	hWindow := smoothstep(-0.40, 0.10, d.Z) * (0.30 + 0.70*smoothstep(-0.92, 0.95, d.X))
	warm := V(1.00, 0.72, 0.39).Mul(hBand * hWindow * (0.82 / (1 + 2.2*blur)))

	// A cool vertical softbox creates the secondary reflected line.
	wV := 0.045 + blur*0.34
	vBand := math.Exp(-0.5 * math.Pow((d.X+0.55)/wV, 2))
	vWindow := smoothstep(-0.55, -0.10, d.Y) * (1 - smoothstep(0.80, 1.0, d.Y))
	vWindow *= smoothstep(-0.30, 0.32, d.Z)
	cool := V(0.30, 0.63, 1.00).Mul(vBand * vWindow * (0.52 / (1 + 2.0*blur)))

	// Eye-level strip reflected by a mostly vertical car panel.
	wE := 0.011 + blur*0.20
	eBand := math.Exp(-0.5 * math.Pow((d.Y+0.115)/wE, 2))
	eWindow := smoothstep(0.38, 0.94, d.Z) * (0.36 + 0.64*smoothstep(-0.82, 0.82, d.X))
	eye := V(0.66, 0.83, 1.00).Mul(eBand * eWindow * (0.94 / (1 + 2.1*blur)))

	// A second, finite softbox gives the flat panel a readable reflected shape.
	softX := math.Exp(-0.5 * math.Pow((d.X+0.34)/0.15, 2))
	softY := math.Exp(-0.5 * math.Pow((d.Y+0.03)/0.038, 2))
	soft := V(1.00, 0.72, 0.42).Mul(softX * softY * smoothstep(0.22, 0.72, d.Z) * (0.80 / (1 + 1.7*blur)))

	// Soft frontal key, softened as roughness grows.
	keyDir := V(-0.25, 0.20, 0.95).Normalize()
	keyDot := clamp(d.Dot(keyDir), 0, 1)
	key := V(1.0, 0.82, 0.62).Mul(math.Pow(keyDot, 34/(1+10*blur)) * (0.72 / (1 + 1.4*blur)))

	return base.Add(warm).Add(cool).Add(eye).Add(soft).Add(key)
}

// aggregateEnvironment is the prefiltered far-field environment. The
// unresolved microfacets have already widened the lobe, so only the
// integrated studio strip remains: a single connected horizontal band.
func aggregateEnvironment(d Vec3, blur float64) Vec3 {
	d = d.Normalize()
	if d.Length() < 0.5 {
		return V(0, 0, 0)
	}
	blur = clamp(blur, 0, 0.8)
	up := smoothstep(-0.70, 0.85, d.Y)
	base := V(0.0022, 0.0040, 0.0075).Lerp(V(0.012, 0.027, 0.050), up)

	w := 0.045 + 0.20*blur
	band := math.Exp(-0.5 * math.Pow((d.Y-0.39)/w, 2))
	window := 0.64 + 0.36*smoothstep(-0.72, 0.25, d.Z)
	window *= 0.72 + 0.28*smoothstep(-0.96, 0.96, d.X)
	energy := 1.18 / (1 + 0.95*blur)
	strip := V(0.90, 0.96, 1.00).Mul(band * window * energy)
	return base.Add(strip)
}

func distributionGGX(ndh, alpha float64) float64 {
	a2 := alpha * alpha
	d := ndh*ndh*(a2-1) + 1
	return a2 / (math.Pi*d*d + 1e-9)
}

func geometrySmith(nDotV, nDotL, rough float64) float64 {
	k := (rough + 1) * (rough + 1) / 8
	return (nDotV / (nDotV*(1-k) + k + 1e-8)) *
		(nDotL / (nDotL*(1-k) + k + 1e-8))
}

func directSpecular(n, v, l Vec3, alpha float64, f0 Vec3) Vec3 {
	ndl := clamp(n.Dot(l), 0, 1)
	ndv := clamp(n.Dot(v), 1e-4, 1)
	if ndl <= 0 {
		return V(0, 0, 0)
	}
	h := l.Add(v).Normalize()
	ndh := clamp(n.Dot(h), 0, 1)
	vdh := clamp(v.Dot(h), 0, 1)
	d := distributionGGX(ndh, alpha)
	g := geometrySmith(ndv, ndl, alpha)
	f := f0.Add(V(1, 1, 1).Sub(f0).Mul(math.Pow(1-vdh, 5)))
	return f.Mul(d * g / (4*ndv*ndl + 1e-7) * ndl)
}

type ShadeInput struct {
	Normal     Vec3
	GeoNormal  Vec3
	ViewDir    Vec3
	Flake      FlakeSample
	ShadowTerm float64
	EdgeLight  float64
	Aggregate  bool
}

// Shade applies basecoat, flake metal, and clearcoat in a single energy-aware
// pass. The flake's normal changes both the reflected environment lookup and
// the GGX half-vector, while its coverage modifies the lobe width.
func (p Paint) Shade(in ShadeInput) Vec3 {
	n := in.Normal.Normalize()
	geoN := in.GeoNormal.Normalize()
	v := in.ViewDir.Normalize()
	ndv := clamp(n.Dot(v), 1e-4, 1)
	geoNdv := clamp(geoN.Dot(v), 1e-4, 1)

	rough := mix(p.BaseRoughness, in.Flake.Roughness, in.Flake.Coverage*0.92)
	rough += 0.20 * in.Flake.Unresolved
	rough += 0.17 * in.Flake.Unresolved * in.Flake.Unresolved
	rough = clamp(rough, 0.035, 0.48)
	alpha := math.Max(rough*rough, 0.002)

	flakeMix := clamp(0.58*in.Flake.Coverage+0.30*in.Flake.Unresolved, 0, 1)
	metalTint := p.SpecularColor.Lerp(in.Flake.Tint, flakeMix)
	reflectDirN := reflectDir(v.Mul(-1), n)
	env := studioEnvironment(reflectDirN, rough)
	if in.Aggregate {
		env = aggregateEnvironment(reflectDirN, rough)
	}
	fresnelMetal := metalTint.Add(V(1, 1, 1).Sub(metalTint).Mul(pow5(1 - ndv)))

	shadow := in.ShadowTerm
	if shadow <= 0 {
		shadow = 1
	}
	base := p.BaseColor.Mul(0.24 + 0.76*clamp(n.Y*0.5+0.5, 0, 1))
	diffuse := base.Mul(0.15 * shadow)
	geoReflect := reflectDir(v.Mul(-1), geoN)
	baseEnv := studioEnvironment(geoReflect, p.BaseRoughness+0.10*in.Flake.Unresolved)
	if in.Aggregate {
		baseEnv = aggregateEnvironment(geoReflect, p.BaseRoughness+0.10*in.Flake.Unresolved)
	}
	baseFresnel := p.SpecularColor.Add(V(1, 1, 1).Sub(p.SpecularColor).Mul(pow5(1 - geoNdv)))
	broadMetal := baseEnv.MulV(baseFresnel).Mul(0.74 * shadow)
	flakeGain := 0.30*in.Flake.Coverage + 0.055*in.Flake.Unresolved
	flakeMetal := env.MulV(fresnelMetal).Mul(flakeGain * (0.55 + 0.45*in.Flake.Glint) * shadow)
	metal := broadMetal.Add(flakeMetal)

	lights := studioLightSet
	directScale := 1.0
	ccDirectScale := 1.0
	clearcoatScale := 1.0
	if in.Aggregate {
		directScale = 0.0
		ccDirectScale = 0.0
		clearcoatScale = 1.18
	}
	direct := V(0, 0, 0)
	glints := V(0, 0, 0)
	for _, light := range lights {
		ndl := clamp(n.Dot(light.Dir), 0, 1)
		if ndl <= 0 {
			continue
		}
		broad := directSpecular(n, v, light.Dir, math.Max(alpha, 0.045), metalTint)
		direct = direct.Add(broad.MulV(light.Color).Mul(light.Intensity * 0.42 * directScale))

		if in.Flake.Coverage > 0.01 && rough < 0.16 {
			tight := directSpecular(n, v, light.Dir, 0.014, metalTint.Mul(3.0))
			glints = glints.Add(tight.MulV(light.Color).Mul(light.Intensity * in.Flake.Coverage * (1.1 + 2.0*in.Flake.Glint)))
		}
	}
	if in.Flake.Coverage > 0.01 {
		// Directional micro-glints: each resolved flake family reflects the
		// studio line lights through its own hemisphere-perturbed normal.
		for _, light := range glintLightSet {
			tight := directSpecular(n, v, light.Dir, 0.011, in.Flake.Tint.Mul(1.35))
			glints = glints.Add(tight.MulV(light.Color).Mul(light.Intensity * in.Flake.Coverage * (0.55 + 0.85*in.Flake.Glint)))
		}
	}

	// Clearcoat reflection uses the unperturbed geometry. This is the stable,
	// continuous highlight that survives flake LOD in the distance.
	ccRough := p.ClearcoatRough + 0.045*in.Flake.Unresolved
	ccEnv := studioEnvironment(reflectDirN, ccRough)
	if in.Aggregate {
		ccEnv = aggregateEnvironment(reflectDirN, ccRough)
	}
	ccFresnel := 0.045 + (1-0.045)*pow5(1-geoNdv)
	clearcoat := ccEnv.Mul(ccFresnel * p.ClearcoatStrength * clearcoatScale * (0.86 + 0.14*shadow))

	ccDirect := V(0, 0, 0)
	for _, light := range lights {
		spec := directSpecular(geoN, v, light.Dir, math.Max(ccRough*ccRough, 0.0012), V(0.045, 0.045, 0.045))
		ccDirect = ccDirect.Add(spec.MulV(light.Color).Mul(light.Intensity * 0.48 * ccDirectScale))
	}

	rim := V(0.34, 0.52, 0.76).Mul(in.EdgeLight * (0.35 + 0.65*geoNdv))
	return diffuse.Add(metal).Add(direct).Add(glints).Add(clearcoat).Add(ccDirect).Add(rim)
}
