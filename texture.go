package main

import "math"

type StripePattern struct {
	Period float64
	Dir    Vec2
	Light  RGB
	Ink    RGB
	Coeff  []float64
}

func NewStripePattern() *StripePattern {
	const (
		power   = 7
		samples = 8192
		order   = power
	)
	p := &StripePattern{
		Period: 16,
		Dir:    Vec2{1 / math.Sqrt2, 1 / math.Sqrt2},
		Light:  RGB{0.78, 0.60, 0.36},
		Ink:    RGB{0.49, 0.38, 0.24},
		Coeff:  make([]float64, order+1),
	}
	for n := 0; n <= order; n++ {
		var sum float64
		for i := 0; i < samples; i++ {
			theta := 2 * math.Pi * float64(i) / samples
			g := math.Pow(0.5+0.5*math.Cos(theta), power)
			sum += g * math.Cos(float64(n)*theta)
		}
		p.Coeff[n] = sum / samples
		if n > 0 {
			p.Coeff[n] *= 2
		}
	}
	return p
}

func (p *StripePattern) phase(uv Vec2) float64 {
	return 2 * math.Pi * p.Dir.Dot(uv) / p.Period
}

func (p *StripePattern) colorAt(uv Vec2) RGB {
	theta := p.phase(uv)
	g := p.Coeff[0]
	for n := 1; n < len(p.Coeff); n++ {
		g += p.Coeff[n] * math.Cos(float64(n)*theta)
	}
	return p.Light.Sub(p.Ink.Mul(g)).Clamp()
}

func (p *StripePattern) analyticFiltered(uv Vec2, fp Footprint) RGB {
	theta := p.phase(uv)
	color := p.Light.Sub(p.Ink.Mul(p.Coeff[0]))
	for n := 1; n < len(p.Coeff); n++ {
		k := 2 * math.Pi * float64(n) / p.Period
		projMajor := k * fp.DirDotMajor(p.Dir)
		projMinor := k * fp.DirDotMinor(p.Dir)
		variance := projMajor*projMajor*fp.Major*fp.Major +
			projMinor*projMinor*fp.Minor*fp.Minor
		attenuation := math.Exp(-variance / 24)
		color = color.Sub(p.Ink.Mul(p.Coeff[n] * attenuation * math.Cos(float64(n)*theta)))
	}
	return color.Clamp()
}

func (fp Footprint) DirDotMajor(d Vec2) float64 { return d.Dot(fp.AxisMajor) }
func (fp Footprint) DirDotMinor(d Vec2) float64 { return d.Dot(fp.AxisMinor) }

type Texture struct {
	Size   int
	Levels [][]float32
}

func NewTexture(pattern *StripePattern, size int) *Texture {
	t := &Texture{Size: size}
	base := make([]float32, size*size*4)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			c := pattern.colorAt(Vec2{float64(x) + 0.5, float64(y) + 0.5}).Clamp()
			i := (y*size + x) * 4
			base[i+0] = float32(c.R)
			base[i+1] = float32(c.G)
			base[i+2] = float32(c.B)
			base[i+3] = 1
		}
	}
	t.Levels = append(t.Levels, base)
	for n := size; n > 1; n /= 2 {
		prev := t.Levels[len(t.Levels)-1]
		nextSize := n / 2
		next := make([]float32, nextSize*nextSize*4)
		for y := 0; y < nextSize; y++ {
			for x := 0; x < nextSize; x++ {
				var r, g, b, a float32
				for oy := 0; oy < 2; oy++ {
					for ox := 0; ox < 2; ox++ {
						i := ((2*y+oy)*n + (2*x + ox)) * 4
						r += prev[i+0]
						g += prev[i+1]
						b += prev[i+2]
						a += prev[i+3]
					}
				}
				i := (y*nextSize + x) * 4
				next[i+0] = r * 0.25
				next[i+1] = g * 0.25
				next[i+2] = b * 0.25
				next[i+3] = a * 0.25
			}
		}
		t.Levels = append(t.Levels, next)
	}
	return t
}

func wrapInt(v, n int) int {
	v %= n
	if v < 0 {
		v += n
	}
	return v
}

func (t *Texture) sampleLevel(level int, uv Vec2) (RGB, int) {
	level = int(clamp(float64(level), 0, float64(len(t.Levels)-1)))
	size := t.Size >> level
	if size < 1 {
		size = 1
	}
	x := uv.X - 0.5
	y := uv.Y - 0.5
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	fx := x - float64(x0)
	fy := y - float64(y0)
	x0 = wrapInt(x0, size)
	x1 := (x0 + 1) % size
	y0 = wrapInt(y0, size)
	y1 := (y0 + 1) % size
	data := t.Levels[level]

	c00 := t.fetch(data, size, x0, y0)
	c10 := t.fetch(data, size, x1, y0)
	c01 := t.fetch(data, size, x0, y1)
	c11 := t.fetch(data, size, x1, y1)
	top := c00.Mul(1 - fx).Add(c10.Mul(fx))
	bottom := c01.Mul(1 - fx).Add(c11.Mul(fx))
	return top.Mul(1 - fy).Add(bottom.Mul(fy)), 4
}

func (t *Texture) fetch(data []float32, size, x, y int) RGB {
	i := (y*size + x) * 4
	return RGB{float64(data[i]), float64(data[i+1]), float64(data[i+2])}
}

func (t *Texture) sampleNearestMip(uv Vec2, lod float64) (RGB, int) {
	maxLevel := len(t.Levels) - 1
	level := int(math.Floor(clamp(lod, 0, float64(maxLevel)) + 0.5))
	return t.sampleLevel(level, uv)
}
