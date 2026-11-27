package main

import "math"

type RGB struct {
	R, G, B float32
}

func (c RGB) scale(s float32) RGB {
	return RGB{c.R * s, c.G * s, c.B * s}
}

func (c RGB) add(o RGB) RGB {
	return RGB{c.R + o.R, c.G + o.G, c.B + o.B}
}

func (c RGB) mul(o RGB) RGB {
	return RGB{c.R * o.R, c.G * o.G, c.B * o.B}
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func lerpf(a, b, t float32) float32 {
	return a + (b-a)*t
}

func smoothstep(edge0, edge1, x float64) float64 {
	t := clamp01((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}

func mix32(x uint32) uint32 {
	x ^= x >> 16
	x *= 0x7feb352d
	x ^= x >> 15
	x *= 0x846ca68b
	x ^= x >> 16
	return x
}

func hash01(x, y uint32, seed uint32) float64 {
	h := mix32(x*0x9e3779b1 ^ y*0x85ebca77 ^ seed)
	return float64(h) / float64(^uint32(0))
}

func valueNoise(x, y float64, seed uint32) float64 {
	x0, y0 := math.Floor(x), math.Floor(y)
	fx, fy := x-x0, y-y0
	u := fx * fx * (3 - 2*fx)
	v := fy * fy * (3 - 2*fy)
	ix, iy := uint32(int64(x0)), uint32(int64(y0))
	a := hash01(ix, iy, seed)
	b := hash01(ix+1, iy, seed)
	c := hash01(ix, iy+1, seed)
	d := hash01(ix+1, iy+1, seed)
	return float64(lerpf(lerpf(float32(a), float32(b), float32(u)),
		lerpf(float32(c), float32(d), float32(u)), float32(v)))
}

func fbm(x, y float64, seed uint32) float64 {
	sum, amp, norm := 0.0, 1.0, 0.0
	for octave := 0; octave < 4; octave++ {
		sum += amp * valueNoise(x, y, seed+uint32(octave)*0x9e37)
		norm += amp
		amp *= 0.5
		x *= 2.03
		y *= 2.01
	}
	return sum / norm
}

type StoneTexture struct {
	Width       int
	Height      int
	heightField []float32
	albedo      []RGB
	gradX       []float32
	gradY       []float32
}

func (t *StoneTexture) index(x, y int) int {
	if x < 0 {
		x = 0
	} else if x >= t.Width {
		x = t.Width - 1
	}
	if y < 0 {
		y = 0
	} else if y >= t.Height {
		y = t.Height - 1
	}
	return y*t.Width + x
}

func (t *StoneTexture) HeightAt(u, v float64) float32 {
	x := u * float64(t.Width-1)
	y := v * float64(t.Height-1)
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := float32(x-float64(x0)), float32(y-float64(y0))
	i00 := t.index(x0, y0)
	i10 := t.index(x0+1, y0)
	i01 := t.index(x0, y0+1)
	i11 := t.index(x0+1, y0+1)
	a := lerpf(t.heightField[i00], t.heightField[i10], fx)
	b := lerpf(t.heightField[i01], t.heightField[i11], fx)
	return lerpf(a, b, fy)
}

func (t *StoneTexture) GradientAt(u, v float64) (float32, float32) {
	x := u * float64(t.Width-1)
	y := v * float64(t.Height-1)
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := float32(x-float64(x0)), float32(y-float64(y0))
	i00 := t.index(x0, y0)
	i10 := t.index(x0+1, y0)
	i01 := t.index(x0, y0+1)
	i11 := t.index(x0+1, y0+1)
	gx := lerpf(lerpf(t.gradX[i00], t.gradX[i10], fx),
		lerpf(t.gradX[i01], t.gradX[i11], fx), fy)
	gy := lerpf(lerpf(t.gradY[i00], t.gradY[i10], fx),
		lerpf(t.gradY[i01], t.gradY[i11], fx), fy)
	return gx, gy
}

func (t *StoneTexture) ColorAt(u, v float64) RGB {
	x := u * float64(t.Width-1)
	y := v * float64(t.Height-1)
	x0, y0 := int(math.Floor(x)), int(math.Floor(y))
	fx, fy := float32(x-float64(x0)), float32(y-float64(y0))
	i00 := t.index(x0, y0)
	i10 := t.index(x0+1, y0)
	i01 := t.index(x0, y0+1)
	i11 := t.index(x0+1, y0+1)
	mix := func(channel func(RGB) float32) float32 {
		a := lerpf(channel(t.albedo[i00]), channel(t.albedo[i10]), fx)
		b := lerpf(channel(t.albedo[i01]), channel(t.albedo[i11]), fx)
		return lerpf(a, b, fy)
	}
	return RGB{mix(func(c RGB) float32 { return c.R }),
		mix(func(c RGB) float32 { return c.G }),
		mix(func(c RGB) float32 { return c.B })}
}

type stoneCell struct {
	x0, x1, y0, y1 float64
	tint           RGB
	lift           float64
	phase          float64
	rough          float64
	crackA, crackB float64
	crackC         float64
}

type stoneRow struct {
	y0, y1 float64
	cells  []stoneCell
}

type sequence struct {
	state uint64
}

func newSequence(seed uint64) *sequence {
	return &sequence{state: seed}
}

func (s *sequence) next() float64 {
	s.state = s.state*6364136223846793005 + 1442695040888963407
	return float64((s.state>>11)&((1<<53)-1)) / float64(1<<53)
}

func (s *sequence) between(lo, hi float64) float64 {
	return lo + (hi-lo)*s.next()
}

func buildStoneTexture(width, height int) *StoneTexture {
	seq := newSequence(0x51a7c0de)
	rows := make([]stoneRow, 0, 6)
	rowWeights := make([]float64, 6)
	rowTotal := 0.0
	for i := range rowWeights {
		rowWeights[i] = seq.between(0.86, 1.14)
		rowTotal += rowWeights[i]
	}
	y := 0.0
	for _, weight := range rowWeights {
		y1 := y + float64(height)*weight/rowTotal
		if len(rows) == len(rowWeights)-1 {
			y1 = float64(height)
		}
		row := stoneRow{y0: y, y1: y1}
		row.cells = buildRowCells(row, width, seq)
		rows = append(rows, row)
		y = y1
	}

	texture := &StoneTexture{
		Width:       width,
		Height:      height,
		heightField: make([]float32, width*height),
		albedo:      make([]RGB, width*height),
		gradX:       make([]float32, width*height),
		gradY:       make([]float32, width*height),
	}
	for py := 0; py < height; py++ {
		row := rowAt(rows, float64(py)+0.5)
		for px := 0; px < width; px++ {
			cell := cellAt(row.cells, float64(px)+0.5)
			h, color := shadeStonePixel(cell, float64(px)+0.5, float64(py)+0.5)
			idx := py*width + px
			texture.heightField[idx] = float32(h)
			texture.albedo[idx] = color
		}
	}
	computeGradients(texture)
	return texture
}

func buildRowCells(row stoneRow, width int, seq *sequence) []stoneCell {
	count := 5
	if seq.next() > 0.46 {
		count = 6
	}
	if seq.next() > 0.82 {
		count = 7
	}
	weights := make([]float64, count)
	total := 0.0
	for i := range weights {
		weights[i] = seq.between(0.76, 1.28)
		total += weights[i]
	}
	cells := make([]stoneCell, 0, count)
	x := 0.0
	for i, weight := range weights {
		x1 := x + float64(width)*weight/total
		if i == count-1 {
			x1 = float64(width)
		}
		palette := []RGB{
			{0.39, 0.41, 0.42}, {0.46, 0.42, 0.37},
			{0.34, 0.38, 0.39}, {0.43, 0.39, 0.34},
			{0.31, 0.35, 0.38}, {0.48, 0.45, 0.40},
		}
		tint := palette[int(seq.next()*float64(len(palette)))%len(palette)]
		tint = tint.scale(float32(seq.between(0.84, 1.08)))
		cells = append(cells, stoneCell{
			x0: x, x1: x1, y0: row.y0, y1: row.y1,
			tint: tint, lift: seq.between(-0.045, 0.075),
			phase: seq.between(0, 1000), rough: seq.between(0.35, 1.0),
			crackA: seq.between(-0.8, 0.8), crackB: seq.between(-0.5, 0.5),
			crackC: seq.between(-0.75, 0.75),
		})
		x = x1
	}
	return cells
}

func rowAt(rows []stoneRow, y float64) *stoneRow {
	for i := range rows {
		if y < rows[i].y1 || i == len(rows)-1 {
			return &rows[i]
		}
	}
	return &rows[len(rows)-1]
}

func cellAt(cells []stoneCell, x float64) *stoneCell {
	for i := range cells {
		if x < cells[i].x1 || i == len(cells)-1 {
			return &cells[i]
		}
	}
	return &cells[len(cells)-1]
}

func shadeStonePixel(cell *stoneCell, px, py float64) (float64, RGB) {
	edge := math.Min(math.Min(px-cell.x0, cell.x1-px),
		math.Min(py-cell.y0, cell.y1-py))
	edge += 5.0 * (fbm(px*0.038, py*0.043, uint32(cell.phase)+37) - 0.5)
	mortar := 2.8 + 2.2*cell.rough
	bevel := 5.5 + 7.0*cell.rough
	face := smoothstep(mortar, mortar+bevel, edge)

	coarse := fbm(px*0.007, py*0.009, uint32(cell.phase))
	mid := fbm(px*0.033, py*0.038, uint32(cell.phase)+91)
	fine := valueNoise(px*0.23, py*0.24, uint32(cell.phase)+173)
	floor := 0.24 + 0.045*fine
	faceHeight := 0.70 + cell.lift + 0.16*coarse + 0.05*mid
	faceHeight += 0.018 * math.Sin((px+cell.phase)*0.035) * math.Sin((py-cell.phase)*0.027)
	height := floor + (faceHeight-floor)*face
	height += 0.012 * (valueNoise(px*0.17, py*0.19, uint32(cell.phase)+514) - 0.5) * face

	localX := (px - cell.x0) / math.Max(1, cell.x1-cell.x0)
	localY := (py - cell.y0) / math.Max(1, cell.y1-cell.y0)
	crackLine := localX*cell.crackA + localY*cell.crackB + cell.crackC
	crack := 1 - smoothstep(0.0, 0.012, math.Abs(crackLine))
	crack *= smoothstep(0.08, 0.3, face)
	height -= 0.085 * crack

	stone := cell.tint.scale(float32(0.84 + 0.26*coarse + 0.10*mid))
	grain := hash01(uint32(px*1.7), uint32(py*1.9), uint32(cell.phase)+401)
	stone = stone.scale(float32(0.91 + 0.18*grain))
	if grain > 0.988 {
		stone = stone.scale(1.24)
	} else if grain < 0.010 {
		stone = stone.scale(0.72)
	}
	stone.R *= float32(0.97 + 0.08*(coarse-0.5))
	stone.G *= float32(0.98 + 0.05*(mid-0.5))
	stone.B *= float32(1.01 - 0.06*(coarse-0.5))

	lichenMask := smoothstep(0.61, 0.75,
		fbm(px*0.012+cell.phase*0.07, py*0.015, uint32(cell.phase)+777))
	lichen := RGB{0.30, 0.34, 0.19}
	stone.R = lerpf(stone.R, lichen.R, float32(0.18*lichenMask*face))
	stone.G = lerpf(stone.G, lichen.G, float32(0.18*lichenMask*face))
	stone.B = lerpf(stone.B, lichen.B, float32(0.18*lichenMask*face))

	mortarColor := RGB{
		R: float32(0.185 + 0.045*mid),
		G: float32(0.195 + 0.04*coarse),
		B: float32(0.165 + 0.03*fine),
	}
	faceMix := float32(face)
	color := RGB{
		R: lerpf(mortarColor.R, stone.R, faceMix),
		G: lerpf(mortarColor.G, stone.G, faceMix),
		B: lerpf(mortarColor.B, stone.B, faceMix),
	}
	cavity := float32(0.64 + 0.36*face)
	color = color.scale(cavity)
	color = color.scale(float32(1 - 0.5*crack))
	return clamp01(height), color
}

func computeGradients(texture *StoneTexture) {
	width, height := texture.Width, texture.Height
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			xm, xp := x-1, x+1
			ym, yp := y-1, y+1
			if xm < 0 {
				xm = 0
			}
			if xp >= width {
				xp = width - 1
			}
			if ym < 0 {
				ym = 0
			}
			if yp >= height {
				yp = height - 1
			}
			i := y*width + x
			gx := (texture.heightField[y*width+xp] - texture.heightField[y*width+xm]) * 0.5
			gy := (texture.heightField[yp*width+x] - texture.heightField[ym*width+x]) * 0.5
			texture.gradX[i] = gx * float32(width)
			texture.gradY[i] = gy * float32(height)
		}
	}
}
