package furnace

import "math"

// RenderConfig controls a single specimen sphere.
type RenderConfig struct {
	Width     int
	Height    int
	Corrected bool
	Table     *EnergyTable
	Exposure  float64
	AA        int
}

// DefaultRenderConfig returns a compact, deterministic high-quality setup.
func DefaultRenderConfig(corrected bool) RenderConfig {
	return RenderConfig{Width: 360, Height: 360, Corrected: corrected, AA: 2, Exposure: 1.15}
}

type areaLight struct {
	dir      Vec3
	color    Vec3
	angle    float64
	samples  int
	hardness float64
}

func studioLights() []areaLight {
	return []areaLight{
		{dir: V(-0.52, 0.58, 0.63).Normalize(), color: V(4.2, 3.9, 3.45), angle: 0.16, samples: 1},
		{dir: V(0.72, 0.30, 0.62).Normalize(), color: V(1.15, 1.45, 2.05), angle: 0.22, samples: 1},
		{dir: V(-0.28, 0.42, -0.86).Normalize(), color: V(2.1, 0.72, 0.42), angle: 0.18, samples: 1},
	}
}

// StudioEnvironment is a deterministic softbox studio gradient.
func StudioEnvironment(d Vec3) Vec3 {
	d = d.Normalize()
	up := saturate(d.Y*0.5 + 0.5)
	col := V(0.028, 0.034, 0.052).Mul(1 - up).Add(V(0.20, 0.235, 0.30).Mul(up))

	keyDir := V(-0.48, 0.72, 0.50).Normalize()
	key := math.Pow(saturate(d.Dot(keyDir)), 220)
	col = col.Add(V(5.0, 4.7, 4.1).Mul(key))
	keyGlow := math.Pow(saturate(d.Dot(keyDir)), 18)
	col = col.Add(V(0.34, 0.32, 0.29).Mul(keyGlow))

	stripDir := V(0.78, 0.18, 0.60).Normalize()
	strip := math.Pow(saturate(d.Dot(stripDir)), 90)
	col = col.Add(V(1.1, 1.45, 2.2).Mul(strip))

	rimDir := V(-0.2, 0.35, -0.92).Normalize()
	rim := math.Pow(saturate(d.Dot(rimDir)), 70)
	col = col.Add(V(1.5, 0.42, 0.24).Mul(rim))
	return col
}

func tangentFrame(n Vec3) (t, b Vec3) {
	ref := V(0, 1, 0)
	if math.Abs(n.Y) > 0.94 {
		ref = V(1, 0, 0)
	}
	t = ref.Sub(n.Mul(ref.Dot(n))).Normalize()
	b = n.Cross(t).Normalize()
	return t, b
}

func toLocal(v, n, t, b Vec3) Vec3 {
	return V(v.Dot(t), v.Dot(b), v.Dot(n))
}

func fromLocal(v Vec3, n, t, b Vec3) Vec3 {
	return t.Mul(v.X).Add(b.Mul(v.Y)).Add(n.Mul(v.Z)).Normalize()
}

func coneSample(center Vec3, angle, u1, u2 float64) Vec3 {
	t, b := tangentFrame(center)
	theta := angle * math.Sqrt(u1)
	phi := 2 * math.Pi * u2
	local := V(math.Sin(theta)*math.Cos(phi), math.Sin(theta)*math.Sin(phi), math.Cos(theta))
	return fromLocal(local, center, t, b)
}

// ShadeStudio returns linear radiance for a sphere surface point.
func ShadeStudio(m Material, corrected bool, table *EnergyTable, n, worldV Vec3) Vec3 {
	n = n.Normalize()
	worldV = worldV.Normalize()
	nDotV := saturate(n.Dot(worldV))
	t, b := tangentFrame(n)
	localV := toLocal(worldV, n, t, b)
	if table == nil && corrected {
		table = DefaultEnergyTable(m)
	}

	eval := func(localL Vec3) Vec3 {
		if corrected {
			return NormalizedBRDF(m, table, localV, localL)
		}
		return RawBRDF(m, localV, localL)
	}

	col := V(0, 0, 0)
	for _, light := range studioLights() {
		nDotL := saturate(n.Dot(light.dir))
		if nDotL > 0 {
			localL := toLocal(light.dir, n, t, b)
			col = col.Add(eval(localL).Mul(nDotL).MulV(light.color))
		}
	}

	// Low-frequency studio ambient keeps rough/dark specimens readable.
	up := n.Y*0.5 + 0.5
	ambient := V(0.055, 0.060, 0.075).Mul(1 - up).Add(V(0.24, 0.27, 0.33).Mul(up))
	scale, deficit := 1.0, 0.0
	if corrected {
		scale, deficit = table.Correction(nDotV)
	}

	diffuseWeight := 1 - saturate(m.Metallic)
	if m.Kind == Clearcoat || m.Kind == Velvet {
		diffuseWeight = 1
	}
	diffuse := m.Albedo.Mul(diffuseWeight).MulV(ambient).Mul(0.85 * scale)
	col = col.Add(diffuse)
	if corrected && deficit > 0 {
		col = col.Add(CompensationColor(m).MulV(ambient).Mul(deficit * 0.55))
	}

	// Prefiltered reflection approximation for studio softboxes.
	rough := m.Roughness
	if m.Kind == Clearcoat {
		rough = m.CoatRoughness
	}
	worldR := n.Mul(2 * nDotV).Sub(worldV).Normalize()
	env := V(0, 0, 0)
	envSamples := 7
	for i := 0; i < envSamples; i++ {
		u1, u2 := Hammersley2D(i, envSamples)
		dir := coneSample(worldR, 0.05+rough*rough*0.42, u1, u2)
		env = env.Add(StudioEnvironment(dir))
	}
	env = env.Div(float64(envSamples))

	switch m.Kind {
	case RoughMetal:
		f := FresnelSchlickV(nDotV, m.f0())
		col = col.Add(env.MulV(f).Mul(0.55 + 0.45*scale))
	case Clearcoat:
		f := FresnelSchlick(nDotV, clamp(m.CoatF0, 0.02, 0.08))
		col = col.Add(env.Mul(f * m.Coat * (0.75 + 0.25*scale)))
	case Velvet:
		f := math.Pow(1-nDotV, 3.2)
		col = col.Add(env.MulV(m.SheenTint).Mul(0.10 + 0.22*f*m.Sheen))
	}
	return col
}

// RenderSpecimen renders one orthographic sphere.
func RenderSpecimen(m Material, cfg RenderConfig) *Canvas {
	if cfg.Width <= 0 {
		cfg.Width = 360
	}
	if cfg.Height <= 0 {
		cfg.Height = 360
	}
	aa := cfg.AA
	if aa < 1 {
		aa = 1
	}
	hiW, hiH := cfg.Width*aa, cfg.Height*aa
	hi := NewCanvas(hiW, hiH)
	hi.Fill(V(0.035, 0.041, 0.058))

	viewHeight := 2.42
	for py := 0; py < hiH; py++ {
		for px := 0; px < hiW; px++ {
			x := (float64(px)+0.5)/float64(hiH)*viewHeight - viewHeight/2
			y := viewHeight/2 - (float64(py)+0.5)/float64(hiH)*viewHeight
			r2 := x*x + y*y
			if r2 > 1 {
				d := V(x, y, -0.65).Normalize()
				hi.Set(px, py, StudioEnvironment(d).Mul(0.55))
				continue
			}
			z := math.Sqrt(math.Max(0, 1-r2))
			n := V(x, y, z).Normalize()
			view := V(0, 0, 1)
			col := ShadeStudio(m, cfg.Corrected, cfg.Table, n, view)
			hi.Set(px, py, col)
		}
	}

	out := NewCanvas(cfg.Width, cfg.Height)
	for y := 0; y < cfg.Height; y++ {
		for x := 0; x < cfg.Width; x++ {
			sum := V(0, 0, 0)
			for ay := 0; ay < aa; ay++ {
				for ax := 0; ax < aa; ax++ {
					sx := x*aa + ax
					sy := y*aa + ay
					sum = sum.Add(hi.Pix[hi.index(sx, sy)])
				}
			}
			out.Set(x, y, sum.Div(float64(aa*aa)))
		}
	}
	return out
}
