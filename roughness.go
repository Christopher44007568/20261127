package main

import (
	"math"
)

const MinRoughness = 0.045

type RoughnessMode int

const (
	ModeOriginal RoughnessMode = iota
	ModeNaiveMIP
	ModeFootprint
)

type MipLevel struct {
	Width  int
	Height int
	MeanR  []float64
	MeanA  []float64 // E[r*r]
	MeanA2 []float64 // E[r*r*r*r], for variance of alpha if needed
}

type RoughnessTexture struct {
	Size   int
	Base   []float64
	Levels []MipLevel
}

type Footprint struct {
	DUDx       float64
	DVDx       float64
	DUDy       float64
	DVDy       float64
	TexelX     float64
	TexelY     float64
	AreaTexels float64
	LongTexels float64
	LOD        float64
	Anisotropy float64
}

type RoughSample struct {
	Roughness       float64
	Alpha           float64
	MeanRoughness   float64
	RMSRoughness    float64
	Variance        float64
	LOD             float64
	FootprintTexels float64
}

func BuildRoughnessTexture(size int) *RoughnessTexture {
	if size < 4 {
		size = 4
	}
	if size&(size-1) != 0 {
		panic("roughness texture size must be a power of two")
	}
	r := &RoughnessTexture{Size: size, Base: make([]float64, size*size)}
	for y := 0; y < size; y++ {
		v := (float64(y) + 0.5) / float64(size)
		for x := 0; x < size; x++ {
			u := (float64(x) + 0.5) / float64(size)
			cloud := fbm(u*7.0+13.7, v*7.0-4.1, 0x51a7, 4)
			mid := fbm(u*28.0-8.0, v*30.0+3.0, 0x9e37, 3)
			fine := fbm(u*150.0, v*150.0, 0x77aa, 2)
			streak := fbm(u*12.0+4.0, v*190.0-9.0, 0x5319, 2)
			spot, groove := cellSpeckle(u+0.13*cloud, v, 155, 0xc0ffee)

			grainHash := hash01(uint32(x), uint32(y), 0xbadc0de)
			grain := smoothstepFloat(0.60, 0.88, grainHash)
			cloudMask := smoothstepFloat(0.42, 0.76, cloud)
			midMask := smoothstepFloat(0.58, 0.82, mid)
			fineMask := smoothstepFloat(0.56, 0.82, fine)
			streakMask := smoothstepFloat(0.68, 0.90, streak)
			poleFade := smoothstepFloat(0.015, 0.16, v) * smoothstepFloat(0.015, 0.16, 1-v)

			value := 0.055 + 0.050*cloudMask*poleFade
			value += 0.56 * grain
			value += 0.15 * midMask
			value += 0.12 * fineMask
			value += 0.07 * streakMask
			value += 0.16 * spot
			value += 0.08 * groove * smoothstepFloat(0.55, 0.82, mid)
			value += (hash01(uint32(x+17), uint32(y-31), 0x1234) - 0.5) * 0.045
			r.Base[y*size+x] = clampFloat(value, MinRoughness, 0.88)
		}
	}
	r.buildMips()
	return r
}

func NewRoughnessFromValues(values []float64, size int) *RoughnessTexture {
	r := &RoughnessTexture{Size: size, Base: append([]float64(nil), values...)}
	r.buildMips()
	return r
}

func (t *RoughnessTexture) buildMips() {
	w, h := t.Size, t.Size
	first := MipLevel{Width: w, Height: h,
		MeanR: append([]float64(nil), t.Base...)}
	first.MeanA = make([]float64, w*h)
	first.MeanA2 = make([]float64, w*h)
	for i, rv := range t.Base {
		a := rv * rv
		first.MeanA[i] = a
		first.MeanA2[i] = a * a
	}
	t.Levels = []MipLevel{first}

	for w > 1 && h > 1 {
		p := t.Levels[len(t.Levels)-1]
		nw, nh := w/2, h/2
		n := MipLevel{Width: nw, Height: nh,
			MeanR: make([]float64, nw*nh), MeanA: make([]float64, nw*nh), MeanA2: make([]float64, nw*nh)}
		for y := 0; y < nh; y++ {
			for x := 0; x < nw; x++ {
				i00 := (2*y)*w + 2*x
				i10 := i00 + 1
				i01 := i00 + w
				i11 := i01 + 1
				ni := y*nw + x
				n.MeanR[ni] = (p.MeanR[i00] + p.MeanR[i10] + p.MeanR[i01] + p.MeanR[i11]) * 0.25
				n.MeanA[ni] = (p.MeanA[i00] + p.MeanA[i10] + p.MeanA[i01] + p.MeanA[i11]) * 0.25
				n.MeanA2[ni] = (p.MeanA2[i00] + p.MeanA2[i10] + p.MeanA2[i01] + p.MeanA2[i11]) * 0.25
			}
		}
		t.Levels = append(t.Levels, n)
		w, h = nw, nh
	}
}

func FootprintFromDerivatives(duDx, dvDx, duDy, dvDy float64, texelSize float64) Footprint {
	tx := math.Hypot(duDx, dvDx) * texelSize
	ty := math.Hypot(duDy, dvDy) * texelSize
	cross := math.Abs(duDx*dvDy - dvDx*duDy)
	area := cross * texelSize * texelSize
	long := math.Max(tx, ty)
	short := math.Min(tx, ty)
	lod := math.Log2(math.Max(long, 1e-6))
	return Footprint{
		DUDx: duDx, DVDx: dvDx, DUDy: duDy, DVDy: dvDy,
		TexelX: tx, TexelY: ty, AreaTexels: area, LongTexels: long,
		LOD: lod, Anisotropy: long / math.Max(short, 1e-6),
	}
}

func (t *RoughnessTexture) clampLOD(lod float64) float64 {
	return clampFloat(lod+1.0, 0, float64(len(t.Levels)-1)+1e-9)
}

func (l MipLevel) wrappedIndex(x, y int) int {
	x %= l.Width
	if x < 0 {
		x += l.Width
	}
	if y < 0 {
		y = 0
	} else if y >= l.Height {
		y = l.Height - 1
	}
	return y*l.Width + x
}

func (l MipLevel) sample(channel []float64, u, v float64) float64 {
	x := u*float64(l.Width) - 0.5
	y := v*float64(l.Height) - 0.5
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	fx := x - math.Floor(x)
	fy := y - math.Floor(y)
	a := channel[l.wrappedIndex(x0, y0)]
	b := channel[l.wrappedIndex(x0+1, y0)]
	c := channel[l.wrappedIndex(x0, y0+1)]
	d := channel[l.wrappedIndex(x0+1, y0+1)]
	return lerpFloat(lerpFloat(a, b, fx), lerpFloat(c, d, fx), fy)
}

type levelStats struct {
	meanR, meanA, meanA2 float64
}

func (t *RoughnessTexture) sampleLODStats(lod, u, v float64) levelStats {
	maxLevel := float64(len(t.Levels) - 1)
	lod = clampFloat(lod, 0, maxLevel)
	i0 := int(math.Floor(lod))
	i1 := i0 + 1
	if i1 >= len(t.Levels) {
		i1 = len(t.Levels) - 1
	}
	f := lod - math.Floor(lod)
	a := t.Levels[i0]
	b := t.Levels[i1]
	s0 := levelStats{a.sample(a.MeanR, u, v), a.sample(a.MeanA, u, v), a.sample(a.MeanA2, u, v)}
	s1 := levelStats{b.sample(b.MeanR, u, v), b.sample(b.MeanA, u, v), b.sample(b.MeanA2, u, v)}
	return levelStats{
		meanR:  lerpFloat(s0.meanR, s1.meanR, f),
		meanA:  lerpFloat(s0.meanA, s1.meanA, f),
		meanA2: lerpFloat(s0.meanA2, s1.meanA2, f),
	}
}

func (t *RoughnessTexture) OriginalAt(u, v float64) RoughSample {
	l := t.Levels[0]
	x := int(math.Floor(u * float64(t.Size)))
	y := int(math.Floor(v * float64(t.Size)))
	r := l.MeanR[l.wrappedIndex(x, y)]
	return RoughSample{Roughness: r, Alpha: r * r, MeanRoughness: r,
		RMSRoughness: r, LOD: 0, FootprintTexels: 0}
}

func (t *RoughnessTexture) Resolve(mode RoughnessMode, u, v float64, fp Footprint) RoughSample {
	if mode == ModeOriginal {
		return t.OriginalAt(u, v)
	}
	lod := t.clampLOD(fp.LOD)
	s := t.sampleLODStats(lod, u, v)
	variance := math.Max(0, s.meanA-s.meanR*s.meanR)
	rms := math.Sqrt(math.Max(s.meanA, 0))

	out := RoughSample{
		MeanRoughness:   s.meanR,
		RMSRoughness:    rms,
		Variance:        variance,
		LOD:             lod,
		FootprintTexels: math.Max(fp.LongTexels, math.Sqrt(math.Max(fp.AreaTexels, 1))),
	}

	if mode == ModeNaiveMIP {
		out.Roughness = clampFloat(s.meanR, MinRoughness, 1)
		out.Alpha = out.Roughness * out.Roughness
		return out
	}

	// alpha = roughness^2 is the GGX slope variance. A footprint average must
	// therefore average alpha (equivalent to mean(r)^2 + Var(r)), rather than
	// squaring the arithmetic mean roughness.
	out.Roughness = math.Sqrt(clampFloat(s.meanA, MinRoughness*MinRoughness, 1))
	out.Alpha = s.meanA
	return out
}

// AddGeometryVariance accounts for sphere curvature inside a screen footprint.
// It is added in alpha space, exactly like microfacet variance addition.
func (s *RoughSample) AddGeometryVariance(sigmaNormal2 float64) {
	s.Alpha = clampFloat(s.Alpha+math.Max(0, sigmaNormal2), MinRoughness*MinRoughness, 1)
	s.Roughness = math.Sqrt(s.Alpha)
}

func RoughnessAlpha(r float64) float64 {
	r = clampFloat(r, MinRoughness, 1)
	return r * r
}
