package main

import (
	"math"
	"math/rand"
)

const (
	MaterialSize = 128 // 每种材质 128x128
	MaterialNum  = 8
)

// 每种材质使用对比强烈的双色调色板，且 8 种材质色相互不重叠，
// 这样一旦发生跨材质串色，输出中会出现调色板之外的颜色，易于检测。

func checkerTexture() *Texture {
	t := NewTexture(MaterialSize, MaterialSize)
	a := Color{0.85, 0.10, 0.10} // 红
	b := Color{0.95, 0.92, 0.88} // 白
	for y := 0; y < t.H; y++ {
		for x := 0; x < t.W; x++ {
			if (x/16+y/16)%2 == 0 {
				t.Set(x, y, a)
			} else {
				t.Set(x, y, b)
			}
		}
	}
	return t
}

func stripesTexture() *Texture {
	t := NewTexture(MaterialSize, MaterialSize)
	a := Color{0.10, 0.70, 0.15} // 绿
	b := Color{0.05, 0.12, 0.05} // 深绿
	for y := 0; y < t.H; y++ {
		for x := 0; x < t.W; x++ {
			if (x+y)/8%2 == 0 {
				t.Set(x, y, a)
			} else {
				t.Set(x, y, b)
			}
		}
	}
	return t
}

func dotsTexture() *Texture {
	t := NewTexture(MaterialSize, MaterialSize)
	bg := Color{0.08, 0.15, 0.75} // 蓝
	fg := Color{0.55, 0.80, 1.00} // 浅蓝
	for y := 0; y < t.H; y++ {
		for x := 0; x < t.W; x++ {
			dx := float64(x%32) - 15.5
			dy := float64(y%32) - 15.5
			if dx*dx+dy*dy < 64 {
				t.Set(x, y, fg)
			} else {
				t.Set(x, y, bg)
			}
		}
	}
	return t
}

func ringsTexture() *Texture {
	t := NewTexture(MaterialSize, MaterialSize)
	a := Color{0.90, 0.80, 0.10} // 黄
	b := Color{0.45, 0.10, 0.55} // 紫
	c := float64(MaterialSize) / 2
	for y := 0; y < t.H; y++ {
		for x := 0; x < t.W; x++ {
			dx := float64(x) - c + 0.5
			dy := float64(y) - c + 0.5
			r := math.Sqrt(dx*dx + dy*dy)
			if int(r/8)%2 == 0 {
				t.Set(x, y, a)
			} else {
				t.Set(x, y, b)
			}
		}
	}
	return t
}

func noiseTexture() *Texture {
	t := NewTexture(MaterialSize, MaterialSize)
	rng := rand.New(rand.NewSource(42))
	a := Color{0.05, 0.65, 0.70} // 青
	b := Color{0.02, 0.10, 0.14} // 深青
	for y := 0; y < t.H; y++ {
		for x := 0; x < t.W; x++ {
			t.Set(x, y, lerpColor(a, b, rng.Float64()))
		}
	}
	return t
}

func bandsTexture() *Texture {
	t := NewTexture(MaterialSize, MaterialSize)
	a := Color{0.85, 0.20, 0.60} // 品红
	b := Color{0.95, 0.55, 0.10} // 橙
	for y := 0; y < t.H; y++ {
		for x := 0; x < t.W; x++ {
			if y/12%2 == 0 {
				t.Set(x, y, a)
			} else {
				t.Set(x, y, b)
			}
		}
	}
	return t
}

func gridTexture() *Texture {
	t := NewTexture(MaterialSize, MaterialSize)
	a := Color{0.80, 0.80, 0.80} // 浅灰
	b := Color{0.15, 0.15, 0.15} // 深灰
	for y := 0; y < t.H; y++ {
		for x := 0; x < t.W; x++ {
			if x%16 < 2 || y%16 < 2 {
				t.Set(x, y, b)
			} else {
				t.Set(x, y, a)
			}
		}
	}
	return t
}

func spiralTexture() *Texture {
	t := NewTexture(MaterialSize, MaterialSize)
	a := Color{0.55, 0.90, 0.20} // 黄绿
	b := Color{0.10, 0.25, 0.40} // 深青蓝
	c := float64(MaterialSize) / 2
	for y := 0; y < t.H; y++ {
		for x := 0; x < t.W; x++ {
			dx := float64(x) - c + 0.5
			dy := float64(y) - c + 0.5
			ang := math.Atan2(dy, dx)
			r := math.Sqrt(dx*dx + dy*dy)
			if math.Sin(ang*3+r*0.35) > 0 {
				t.Set(x, y, a)
			} else {
				t.Set(x, y, b)
			}
		}
	}
	return t
}

func makeMaterials() []*Texture {
	return []*Texture{
		checkerTexture(),
		stripesTexture(),
		dotsTexture(),
		ringsTexture(),
		noiseTexture(),
		bandsTexture(),
		gridTexture(),
		spiralTexture(),
	}
}
