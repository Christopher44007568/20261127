package main

import "math"

type boundaryInfo struct {
	distance float64
	normal   Vec2
}

func circleBoundary(c Circle2, p Vec2) boundaryInfo {
	rel := p.Sub(c.C)
	d := rel.Len()
	if d < 1e-9 {
		return boundaryInfo{distance: c.R, normal: v2(0, 1)}
	}
	return boundaryInfo{distance: math.Abs(c.R - d), normal: rel.Mul(1 / d)}
}

func polygonBoundary(p Polygon2, q Vec2) boundaryInfo {
	bestD := math.Inf(1)
	bestN := p.Normals[0]
	for i := range p.Verts {
		a := p.Verts[i]
		b := p.Verts[(i+1)%len(p.Verts)]
		e := b.Sub(a)
		l2 := e.Dot(e)
		t := clamp01(q.Sub(a).Dot(e) / l2)
		closest := a.Add(e.Mul(t))
		delta := q.Sub(closest)
		d := delta.Len()
		if d < bestD {
			bestD = d
			if d > 1e-9 {
				bestN = delta.Mul(1 / d)
			} else {
				bestN = p.Normals[i]
			}
		}
	}
	return boundaryInfo{distance: bestD, normal: bestN}
}

func shapeBoundary(s Shape2, p Vec2) boundaryInfo {
	switch v := s.(type) {
	case Circle2:
		return circleBoundary(v, p)
	case Polygon2:
		return polygonBoundary(v, p)
	default:
		return boundaryInfo{distance: 1, normal: v2(0, 1)}
	}
}

func PaintGlassBody(c *Canvas, shape Shape2, extrusionDepth float64, accent Vec3) {
	b := shape.Bounds()
	x0f, y0f := c.WorldToPixel(b.Min)
	x1f, y1f := c.WorldToPixel(b.Max)
	// WorldToPixel flips Y; construct conservative pixel rectangle.
	px0 := int(math.Floor(math.Min(x0f, x1f))) - 2
	px1 := int(math.Ceil(math.Max(x0f, x1f))) + 2
	py0 := int(math.Floor(math.Min(y0f, y1f))) - 2
	py1 := int(math.Ceil(math.Max(y0f, y1f))) + 2
	samples := WavelengthSamples(14)
	light := v3(-0.42, 0.58, 0.70).Norm()
	half := light.Add(v3(0, 0, 1)).Norm()

	for py := py0; py <= py1; py++ {
		for px := px0; px <= px1; px++ {
			if px < 0 || py < 0 || px >= c.W || py >= c.H {
				continue
			}
			var sum Vec3
			for sy := 0; sy < 2; sy++ {
				for sx := 0; sx < 2; sx++ {
					world := c.PixelToWorld(
						float64(px)+(float64(sx)+0.5)/2,
						float64(py)+(float64(sy)+0.5)/2,
					)
					col := benchColor(world)
					if shape.Contains(world) {
						info := shapeBoundary(shape, world)
						thickness := extrusionDepth
						var normal3 Vec3
						if circle, ok := shape.(Circle2); ok {
							col = SphereBackground(world, circle)
							rel := world.Sub(circle.C)
							rho := clamp(rel.Len()/circle.R, 0, 1)
							nz := math.Sqrt(math.Max(0, 1-rho*rho))
							thickness = 2 * circle.R * nz
							normal3 = v3(rel.X/circle.R, rel.Y/circle.R, nz)
						} else {
							rim := 1 - smoothstep(0.02, 0.20, info.distance)
							normal3 = v3(info.normal.X*rim, info.normal.Y*rim, 0.72).Norm()
							sheenCoord := (world.X+1.15)*0.72 + world.Y*0.31
							sheen := math.Exp(-sheenCoord * sheenCoord / 0.055)
							col = col.Add(hexLinear(0xd8f6ff).Mul(0.10 * sheen))
							trans := absorptionRGB(thickness, samples)
							col = col.MulV(trans)
						}

						cosV := clamp01(normal3.Z)
						fresnel := 0.055 + 0.945*math.Pow(1-cosV, 5)
						col = col.Add(hexLinear(0x9edcff).Mul(0.20 * fresnel))
						nDotH := math.Max(0, normal3.Dot(half))
						spec := math.Pow(nDotH, 72)
						col = col.Add(v3(1, 1, 1).Mul(0.55 * spec))

						rimWorld := 1 - smoothstep(0.015, 0.13, info.distance)
						col = col.Add(accent.Mul(0.10*rimWorld + 0.10*fresnel))
					}
					sum = sum.Add(col)
				}
			}
			c.Set(px, py, sum.Mul(0.25))
		}
	}
}

func PaintGlassRim(c *Canvas, shape Shape2) {
	switch s := shape.(type) {
	case Circle2:
		strokeCircle(c, s.C, s.R, 4.0, hexLinear(0xbdefff), 0.78)
		strokeCircle(c, s.C, s.R-0.055, 1.4, hexLinear(0x245b86), 0.55)
	case Polygon2:
		strokeWorldPolygon(c, s, 4.2, hexLinear(0xc8f4ff), 0.78)
		strokeWorldPolygon(c, s, 1.4, hexLinear(0x245b86), 0.55)
	}
}

func strokeCircle(c *Canvas, center Vec2, radius, width float64, col Vec3, alpha float64) {
	const segments = 120
	for i := 0; i < segments; i++ {
		a0 := 2 * math.Pi * float64(i) / segments
		a1 := 2 * math.Pi * float64(i+1) / segments
		p0 := center.Add(v2(math.Cos(a0)*radius, math.Sin(a0)*radius))
		p1 := center.Add(v2(math.Cos(a1)*radius, math.Sin(a1)*radius))
		c.LineWorld(p0, p1, LineStyle{Color: col, Width: width, Alpha: alpha})
	}
}
