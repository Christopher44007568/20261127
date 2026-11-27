package main

import "math"

type LODTexture interface {
	Texture
	SampleLOD(u, v, lod float64) Vec3
}

type MipmapTexture struct {
	Levels []*ImageTexture
}

func NewMipmapTexture(base *ImageTexture) *MipmapTexture {
	mips := []*ImageTexture{base}
	for current := base; current.Size > 2; {
		nextSize := current.Size / 2
		if nextSize < 1 {
			nextSize = 1
		}
		next := NewImageTexture(nextSize)
		for y := 0; y < nextSize; y++ {
			for x := 0; x < nextSize; x++ {
				x0 := 2 * x
				y0 := 2 * y
				c0 := current.Pixels[current.index(x0, y0)]
				c1 := current.Pixels[current.index(minInt(x0+1, current.Size-1), y0)]
				c2 := current.Pixels[current.index(x0, minInt(y0+1, current.Size-1))]
				c3 := current.Pixels[current.index(minInt(x0+1, current.Size-1), minInt(y0+1, current.Size-1))]
				next.Pixels[next.index(x, y)] = c0.Add(c1).Add(c2).Add(c3).Scale(0.25)
			}
		}
		mips = append(mips, next)
		current = next
		if nextSize == 1 {
			break
		}
	}
	return &MipmapTexture{Levels: mips}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m *MipmapTexture) Sample(u, v float64) Vec3 {
	return m.SampleLOD(u, v, 0)
}

func (m *MipmapTexture) BaseSize() int {
	return m.Levels[0].Size
}

func (m *MipmapTexture) SampleLOD(u, v, lod float64) Vec3 {
	if len(m.Levels) == 0 {
		return V(0, 0, 0)
	}
	if lod <= 0 {
		return m.Levels[0].Sample(u, v)
	}
	maxLOD := float64(len(m.Levels) - 1)
	if lod >= maxLOD {
		return m.Levels[int(maxLOD)].Sample(u, v)
	}
	level0 := int(math.Floor(lod))
	fraction := lod - math.Floor(lod)
	c0 := m.Levels[level0].Sample(u, v)
	c1 := m.Levels[level0+1].Sample(u, v)
	return lerpVec(c0, c1, fraction)
}

func sampleTextureLOD(texture Texture, u, v, lod float64) Vec3 {
	if lodTexture, ok := texture.(LODTexture); ok {
		return lodTexture.SampleLOD(u, v, lod)
	}
	return texture.Sample(u, v)
}
