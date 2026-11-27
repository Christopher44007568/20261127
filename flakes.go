package flakepaint

import "math"

type Vec2 struct {
	X, Y float64
}

func V2(x, y float64) Vec2 { return Vec2{x, y} }

// FlakeLayer describes one spatial scale of metallic flakes.
type FlakeLayer struct {
	CellMM    float64
	RadiusMM  float64
	Density   float64
	Weight    float64
	Tilt      float64
	Roughness float64
	Tint      Vec3
}

// FlakeField is a deterministic, mip-like stack of flake distributions.
type FlakeField struct {
	Seed         uint64
	MacroScaleMM float64
	Layers       []FlakeLayer
}

// FlakeSample contains the combined, view-ready surface response.
type FlakeSample struct {
	N          Vec3
	Roughness  float64
	Coverage   float64
	Unresolved float64
	Detail     float64
	Tint       Vec3
	Glint      float64
}

// FlakeInstance is an actual flake cell, used to build the density map.
type FlakeInstance struct {
	Layer  int
	Center Vec2
	RX     float64
	RY     float64
	Rot    float64
	Tint   Vec3
}

// DefaultFlakeField uses four overlapping physical scales. The dimensions are
// slightly enlarged for legibility, but remain tied to millimetre world space.
func DefaultFlakeField(seed uint64) FlakeField {
	return FlakeField{
		Seed:         seed,
		MacroScaleMM: 52,
		Layers: []FlakeLayer{
			{CellMM: 1.25, RadiusMM: 0.46, Density: 0.58, Weight: 0.96, Tilt: 0.085, Roughness: 0.035, Tint: V(1.00, 1.18, 1.46)},
			{CellMM: 2.05, RadiusMM: 0.72, Density: 0.42, Weight: 0.86, Tilt: 0.120, Roughness: 0.050, Tint: V(0.46, 1.36, 1.08)},
			{CellMM: 3.30, RadiusMM: 1.12, Density: 0.24, Weight: 0.66, Tilt: 0.160, Roughness: 0.068, Tint: V(1.42, 0.80, 0.28)},
			{CellMM: 5.30, RadiusMM: 1.75, Density: 0.11, Weight: 0.44, Tilt: 0.205, Roughness: 0.090, Tint: V(1.48, 0.48, 0.15)},
		},
	}
}

type flakeCell struct {
	ok     bool
	center Vec2
	rx, ry float64
	rot    float64
	offX   float64
	offY   float64
	rough  float64
	tint   Vec3
	bright float64
}

func (f FlakeField) cellFlake(layer int, ix, iy int64) flakeCell {
	l := f.Layers[layer]
	h0 := hash2i(ix, iy, f.Seed^uint64(layer+1)*0x94d049bb133111eb)
	center := V2(
		(float64(ix)+0.18+0.64*unitFromHash(mix64(h0+1)))*l.CellMM,
		(float64(iy)+0.18+0.64*unitFromHash(mix64(h0+2)))*l.CellMM,
	)
	macro := macroDensityAt(center.X/f.MacroScaleMM, center.Y/f.MacroScaleMM, f.Seed)
	if unitFromHash(mix64(h0+3)) > clamp(l.Density*macro, 0, 1) {
		return flakeCell{}
	}

	rx := l.RadiusMM * (0.72 + 0.56*unitFromHash(mix64(h0+4)))
	ry := rx * (0.62 + 0.34*unitFromHash(mix64(h0+5)))
	rot := 2 * math.Pi * unitFromHash(mix64(h0+6))
	azimuth := 2 * math.Pi * unitFromHash(mix64(h0+7))
	hemisphere := math.Sqrt(unitFromHash(mix64(h0 + 8)))
	tilt := l.Tilt * (0.30 + 0.70*hemisphere)
	rough := l.Roughness + 0.035*(unitFromHash(mix64(h0+9))-0.5)
	bright := 0.72 + 0.56*unitFromHash(mix64(h0+10))

	return flakeCell{
		ok:     true,
		center: center,
		rx:     rx,
		ry:     ry,
		rot:    rot,
		offX:   math.Cos(azimuth) * tilt,
		offY:   math.Sin(azimuth) * tilt,
		rough:  rough,
		tint:   l.Tint.Mul(bright),
		bright: bright,
	}
}

func ellipseDistance(uv, center Vec2, rx, ry, rot float64) float64 {
	dx := uv.X - center.X
	dy := uv.Y - center.Y
	c := math.Cos(rot)
	s := math.Sin(rot)
	x := (c*dx + s*dy) / rx
	y := (-s*dx + c*dy) / ry
	return math.Sqrt(x*x + y*y)
}

// Sample evaluates resolved flake normals and unresolved aggregate statistics.
// footprintMM is the projected pixel footprint on the painted surface.
func (f FlakeField) Sample(uv Vec2, base Frame, footprintMM float64) FlakeSample {
	base = Frame{N: base.N.Normalize(), T: base.T.Normalize(), B: base.B.Normalize()}
	if footprintMM < 1e-6 {
		footprintMM = 1e-6
	}

	sumCov := 0.0
	sumOffX := 0.0
	sumOffY := 0.0
	sumRough := 0.0
	sumTint := V(0, 0, 0)
	sumGlint := 0.0
	unresolved := 0.0

	for li, l := range f.Layers {
		ratio := footprintMM / l.CellMM
		nearFade := 1 - smoothstep(0.18, 0.58, ratio)
		macro := macroDensityAtLOD(uv.X/f.MacroScaleMM, uv.Y/f.MacroScaleMM, footprintMM, f.Seed)
		macro = mix(macro, 0.62, (1-nearFade)*0.94)
		avgArea := math.Pi * l.RadiusMM * (0.78 * l.RadiusMM)
		aggCov := l.Density * macro * avgArea / (l.CellMM * l.CellMM)
		aggCov = clamp(aggCov, 0, 0.92) * l.Weight
		unresolved += aggCov * (1 - nearFade)

		if nearFade <= 0.001 {
			sumTint = sumTint.Add(l.Tint.Mul(aggCov))
			continue
		}

		cx := int64(math.Floor(uv.X / l.CellMM))
		cy := int64(math.Floor(uv.Y / l.CellMM))
		for oy := int64(-1); oy <= 1; oy++ {
			for ox := int64(-1); ox <= 1; ox++ {
				cell := f.cellFlake(li, cx+ox, cy+oy)
				if !cell.ok {
					continue
				}
				q := ellipseDistance(uv, cell.center, cell.rx, cell.ry, cell.rot)
				edge := clamp(footprintMM/(2*math.Min(cell.rx, cell.ry)), 0.055, 0.75)
				cov := 1 - smoothstep(1-edge, 1+edge, q)
				if cov <= 0 {
					continue
				}
				w := cov * l.Weight * nearFade
				sumCov += w
				sumOffX += cell.offX * w
				sumOffY += cell.offY * w
				sumRough += cell.rough * w
				sumTint = sumTint.Add(cell.tint.Mul(w * (1 - 0.55*ratio)))
				sumGlint += cell.bright * w
			}
		}
	}

	normDen := math.Max(1, sumCov)
	n := base.N.
		Add(base.T.Mul(sumOffX / normDen)).
		Add(base.B.Mul(sumOffY / normDen)).
		Normalize()
	if n.Length() < 0.5 {
		n = base.N
	}

	coverage := 1 - math.Exp(-1.35*sumCov)
	detail := coverage
	unresolved = 1 - math.Exp(-1.35*unresolved)
	roughness := 0.085
	if sumCov > 1e-8 {
		roughness = sumRough / sumCov
	}
	tint := V(0.72, 0.82, 1.0)
	tintDen := sumCov + unresolved*0.20
	if tintDen > 1e-8 {
		tint = sumTint.Add(V(0.72, 0.82, 1.0).Mul(unresolved * 0.20)).Mul(1 / tintDen)
	}
	glint := 0.0
	if sumCov > 1e-8 {
		glint = clamp(sumGlint/sumCov-0.72, 0, 1)
	}

	return FlakeSample{
		N:          n,
		Roughness:  roughness,
		Coverage:   coverage,
		Unresolved: clamp(unresolved, 0, 1),
		Detail:     detail,
		Tint:       tint,
		Glint:      glint,
	}
}

// EnumerateFlakes exposes active cells for histogram and density-map rendering.
func (f FlakeField) EnumerateFlakes(min, max Vec2, fn func(FlakeInstance)) {
	for li, l := range f.Layers {
		x0 := int64(math.Floor(min.X/l.CellMM)) - 1
		x1 := int64(math.Ceil(max.X / l.CellMM))
		y0 := int64(math.Floor(min.Y/l.CellMM)) - 1
		y1 := int64(math.Ceil(max.Y / l.CellMM))
		for y := y0; y <= y1; y++ {
			for x := x0; x <= x1; x++ {
				cell := f.cellFlake(li, x, y)
				if !cell.ok {
					continue
				}
				if cell.center.X < min.X-cell.rx || cell.center.X > max.X+cell.rx ||
					cell.center.Y < min.Y-cell.ry || cell.center.Y > max.Y+cell.ry {
					continue
				}
				fn(FlakeInstance{
					Layer:  li,
					Center: cell.center,
					RX:     cell.rx,
					RY:     cell.ry,
					Rot:    cell.rot,
					Tint:   cell.tint,
				})
			}
		}
	}
}
