package main

import "math"

type ImageTexture struct {
	Size   int
	Pixels []Vec3
}

func NewImageTexture(size int) *ImageTexture {
	if size < 2 {
		size = 2
	}
	return &ImageTexture{Size: size, Pixels: make([]Vec3, size*size)}
}

func (t *ImageTexture) index(x, y int) int {
	return y*t.Size + x
}

func (t *ImageTexture) Set(x, y int, c Vec3) {
	x = x % t.Size
	y = y % t.Size
	if x < 0 {
		x += t.Size
	}
	if y < 0 {
		y += t.Size
	}
	t.Pixels[t.index(x, y)] = c
}

func repeatUnit(x float64) float64 {
	x = math.Mod(x, 1)
	if x < 0 {
		x++
	}
	return x
}

func (t ImageTexture) Sample(u, v float64) Vec3 {
	u = repeatUnit(u)
	v = repeatUnit(v)
	x := u * float64(t.Size)
	y := v * float64(t.Size)
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	tx := x - math.Floor(x)
	ty := y - math.Floor(y)

	at := func(ix, iy int) Vec3 {
		ix %= t.Size
		iy %= t.Size
		if ix < 0 {
			ix += t.Size
		}
		if iy < 0 {
			iy += t.Size
		}
		return t.Pixels[t.index(ix, iy)]
	}

	c00 := at(x0, y0)
	c10 := at(x0+1, y0)
	c01 := at(x0, y0+1)
	c11 := at(x0+1, y0+1)
	c0 := lerpVec(c00, c10, tx)
	c1 := lerpVec(c01, c11, tx)
	return lerpVec(c0, c1, ty)
}

func hashInt(ix, iy, seed int) uint64 {
	h := uint64(ix)*0x9e3779b97f4a7c15 ^ uint64(iy)*0xbf58476d1ce4e5b9 ^ uint64(seed)*0x94d049bb133111eb
	h ^= h >> 30
	h *= 0xbf58476d1ce4e5b9
	h ^= h >> 27
	h *= 0x94d049bb133111eb
	h ^= h >> 31
	return h
}

func periodicHash(ix, iy, period, seed int) float64 {
	if period > 0 {
		ix %= period
		iy %= period
		if ix < 0 {
			ix += period
		}
		if iy < 0 {
			iy += period
		}
	}
	return float64(hashInt(ix, iy, seed)&0xffffff) / float64(0xffffff)
}

func periodicNoise(u, v float64, cells, seed int) float64 {
	x := u * float64(cells)
	y := v * float64(cells)
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	tx := x - math.Floor(x)
	ty := y - math.Floor(y)
	sx := tx * tx * (3 - 2*tx)
	sy := ty * ty * (3 - 2*ty)

	n00 := periodicHash(x0, y0, cells, seed)
	n10 := periodicHash(x0+1, y0, cells, seed)
	n01 := periodicHash(x0, y0+1, cells, seed)
	n11 := periodicHash(x0+1, y0+1, cells, seed)
	return lerpFloat(lerpFloat(n00, n10, sx), lerpFloat(n01, n11, sx), sy)
}

func periodicFBM(u, v float64, baseCells, octaves, seed int) float64 {
	sum := 0.0
	amp := 0.5
	norm := 0.0
	cells := baseCells
	for octave := 0; octave < octaves; octave++ {
		sum += amp * periodicNoise(u, v, cells, seed+octave*101)
		norm += amp
		amp *= 0.5
		cells *= 2
	}
	return sum / norm
}

func srgbComponentToLinear(c float64) float64 {
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func srgb255(r, g, b int) Vec3 {
	return V(
		srgbComponentToLinear(float64(r)/255),
		srgbComponentToLinear(float64(g)/255),
		srgbComponentToLinear(float64(b)/255),
	)
}

func NewRockTexture(size, seed int) *ImageTexture {
	if size <= 0 {
		size = 256
	}
	tex := NewImageTexture(size)

	dark := srgb255(54, 50, 44)
	shale := srgb255(91, 84, 72)
	gray := srgb255(112, 113, 105)
	ochre := srgb255(145, 119, 78)
	light := srgb255(177, 164, 130)

	for y := 0; y < size; y++ {
		v := (float64(y) + 0.5) / float64(size)
		for x := 0; x < size; x++ {
			u := (float64(x) + 0.5) / float64(size)

			warpX := periodicFBM(u+0.137, v+0.419, 4, 4, seed) - 0.5
			warpY := periodicFBM(u+0.731, v+0.067, 4, 4, seed+17) - 0.5
			macro := periodicFBM(u+warpX*0.22, v+warpY*0.22, 3, 5, seed+31)
			medium := periodicFBM(u+warpX*0.10, v+warpY*0.12, 12, 4, seed+47)
			grain := periodicFBM(u, v, 56, 3, seed+71)

			strataPhase := v*4.75 + macro*0.95 + warpY*0.35
			strata := 0.5 + 0.5*math.Sin(2*math.Pi*strataPhase)
			strata = smoothstep(0.08, 0.96, strata)
			finePhase := v*11 + medium*1.25
			fine := 0.5 + 0.5*math.Sin(2*math.Pi*finePhase)
			vertical := periodicFBM(u*5.0, v, 10, 4, seed+137)

			base := lerpVec(dark, shale, clamp01(macro*1.25))
			base = lerpVec(base, gray, clamp01((medium-0.35)*0.9))
			base = lerpVec(base, ochre, strata*0.18)
			base = lerpVec(base, light, math.Pow(strata, 4)*0.10)
			base = base.Scale(0.95 + 0.025*fine + 0.11*grain)
			base = base.Mul(V(0.90+0.18*vertical, 0.89+0.16*vertical, 0.86+0.13*vertical))

			ridge := 1 - math.Abs(2*periodicFBM(u+0.31, v+0.57, 9, 4, seed+91)-1)
			vein := smoothstep(0.91, 0.995, ridge)
			base = lerpVec(base, dark.Scale(0.42), vein*0.34)

			pits := smoothstep(0.68, 0.94, periodicFBM(u+0.61, v+0.13, 42, 3, seed+113))
			base = base.Scale(1 - pits*0.22)

			tex.Pixels[tex.index(x, y)] = V(
				clamp01(base.X),
				clamp01(base.Y),
				clamp01(base.Z),
			)
		}
	}
	return tex
}

type ConstantTexture struct {
	Color Vec3
}

func (t ConstantTexture) Sample(float64, float64) Vec3 { return t.Color }
