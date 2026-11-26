package main

import "math"

type Material struct {
	BaseColor    Vec3
	SheenColor   Vec3
	SpecularTint Vec3
	GrazingTint  Vec3
	TransmitTint Vec3

	FiberRoughness  float64
	CrossRoughness  float64
	SpecularEnergy  float64
	SheenEnergy     float64
	GrazingEnergy   float64
	BacklightBudget float64
}

func velvetMaterial() Material {
	return Material{
		BaseColor:       V(0.018, 0.0042, 0.0080),
		SheenColor:      V(0.310, 0.072, 0.112),
		SpecularTint:    V(0.780, 0.390, 0.270),
		GrazingTint:     V(0.360, 0.090, 0.125),
		TransmitTint:    V(0.820, 0.125, 0.105),
		FiberRoughness:  0.10,
		CrossRoughness:  0.52,
		SpecularEnergy:  0.72,
		SheenEnergy:     0.66,
		GrazingEnergy:   0.20,
		BacklightBudget: 0.70,
	}
}

type Light struct {
	Direction Vec3
	Radiance  Vec3
	Behind    bool
}

type ShadePoint struct {
	Normal   Vec3
	Tangent  Vec3
	View     Vec3
	EdgeDist float64
}

func (m Material) Shade(p ShadePoint, lights []Light) Vec3 {
	out := Vec3{}
	n := p.Normal.Normalize()
	view := p.View.Normalize()
	ndvRaw := n.Dot(view)
	if ndvRaw < 0 {
		n = n.Scale(-1)
		ndvRaw = -ndvRaw
	}
	ndv := math.Max(ndvRaw, 1e-4)
	fiber := p.Tangent.Sub(n.Scale(n.Dot(p.Tangent))).Normalize()
	if fiber.Length() < 1e-8 {
		fiber = n.Cross(view).Normalize()
	}
	bitangent := n.Cross(fiber).Normalize()

	viewGrazing := math.Pow(1-ndv, 4)
	edge := math.Exp(-math.Max(p.EdgeDist, 0) / 0.072)
	fiberPhase := 0.86 + 0.14*math.Pow(math.Abs(fiber.Dot(view)), 2)
	hasBacklight := false

	for _, light := range lights {
		if light.Behind {
			hasBacklight = true
			out = out.Add(m.shadeTransmitted(n, fiber, viewGrazing, edge, fiberPhase, light))
			continue
		}
		out = out.Add(m.shadeReflected(n, fiber, bitangent, view, ndv, viewGrazing, edge, fiberPhase, light))
	}
	if hasBacklight {
		out = limitRadiance(out, m.BacklightBudget)
	}
	return out
}

func (m Material) shadeReflected(n, fiber, bitangent, view Vec3, ndv, viewGrazing, edge, fiberPhase float64, light Light) Vec3 {
	ldir := light.Direction.Normalize()
	ndlRaw := n.Dot(ldir)
	if ndlRaw <= 0 {
		return Vec3{}
	}
	ndl := math.Max(ndlRaw, 1e-4)
	h := ldir.Add(view).Normalize()
	ndh := math.Max(n.Dot(h), 1e-4)

	diffuse := m.BaseColor.Scale(ndl / math.Pi)
	diffuse = diffuse.Scale((0.74 + 0.26*(1-viewGrazing)) * (1 - 0.42*edge))

	sheenD := math.Pow(1-ndh, 5)
	sheenGrazing := math.Pow(1-ndl, 2)
	sheen := m.SheenColor.Scale(
		m.SheenEnergy * ndl * sheenD * fiberPhase *
			(0.16 + 0.84*viewGrazing + 0.28*sheenGrazing),
	)

	xf := fiber.Dot(h)
	yf := bitangent.Dot(h)
	at := m.FiberRoughness
	ab := m.CrossRoughness
	den := xf*xf/(at*at) + yf*yf/(ab*ab) + ndh*ndh
	dGGX := 1 / (math.Pi * at * ab * den * den)
	vis := 0.5 / (ndl*math.Sqrt(ndv*ndv*(1-at*at)+at*at) +
		ndv*math.Sqrt(ndl*ndl*(1-ab*ab)+ab*ab))
	ndv5 := math.Pow(1-ndv, 5)
	fresnel := 0.11 + 0.89*ndv5
	aniso := m.SpecularTint.Scale(m.SpecularEnergy * dGGX * vis * ndl * fresnel)

	grazing := m.GrazingTint.Scale(
		m.GrazingEnergy * viewGrazing * ndl * (0.25 + 0.75*math.Pow(1-ndl, 3)),
	)
	direct := diffuse.Add(sheen).Add(aniso).Add(grazing)
	direct = limitRadiance(direct, 2.2)
	return direct.Mul(light.Radiance)
}

func (m Material) shadeTransmitted(n, fiber Vec3, viewGrazing, edge, fiberPhase float64, light Light) Vec3 {
	ldir := light.Direction.Normalize()
	back := math.Max(-n.Dot(ldir), 0)
	if back <= 0 {
		return Vec3{}
	}
	foldGrazing := math.Pow(1-clamp(-n.Dot(ldir), 0, 1), 2)
	thin := clamp(edge*edge+0.30*viewGrazing+0.08*foldGrazing, 0, 1)
	braid := 0.72 + 0.28*math.Pow(math.Abs(fiber.Dot(ldir)), 2)
	phase := 0.62 + 0.38*back
	transmit := m.TransmitTint.Scale(
		(0.010 + 0.990*thin) * phase * braid * fiberPhase,
	)
	body := m.BaseColor.Scale(0.18 * back * thin)
	result := transmit.Add(body).Mul(light.Radiance)
	return limitRadiance(result, m.BacklightBudget)
}

func limitRadiance(c Vec3, ceiling float64) Vec3 {
	peak := c.MaxComponent()
	if peak <= ceiling {
		return c
	}
	return c.Scale(ceiling / peak)
}
