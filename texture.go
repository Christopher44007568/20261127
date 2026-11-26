package main

import "math"

// Color 为线性浮点 RGB，分量范围 [0,1]。
type Color struct {
	R, G, B float64
}

func clampf(x, lo, hi float64) float64 {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func clampi(x, lo, hi int) int {
	if x < lo {
		return lo
	}
	if x > hi {
		return hi
	}
	return x
}

func (c Color) clamp01() Color {
	return Color{clampf(c.R, 0, 1), clampf(c.G, 0, 1), clampf(c.B, 0, 1)}
}

func lerpColor(a, b Color, t float64) Color {
	return Color{
		a.R + (b.R-a.R)*t,
		a.G + (b.G-a.G)*t,
		a.B + (b.B-a.B)*t,
	}
}

func diffColor(a, b Color) Color {
	return Color{math.Abs(a.R - b.R), math.Abs(a.G - b.G), math.Abs(a.B - b.B)}
}

// Texture 为等尺寸 RGB 纹理，行主序存储。
type Texture struct {
	W, H int
	Pix  []Color
}

func NewTexture(w, h int) *Texture {
	return &Texture{W: w, H: h, Pix: make([]Color, w*h)}
}

func (t *Texture) At(x, y int) Color {
	return t.Pix[y*t.W+x]
}

func (t *Texture) Set(x, y int, c Color) {
	t.Pix[y*t.W+x] = c
}

func wrapIndex(i, n int) int {
	i %= n
	if i < 0 {
		i += n
	}
	return i
}

// SampleWrap 以重复寻址做双线性采样，u/v 为归一化 UV。
// 纹理数组的每一层独立存储，采样永远不会跨层。
func (t *Texture) SampleWrap(u, v float64) Color {
	u -= math.Floor(u)
	v -= math.Floor(v)
	x := u*float64(t.W) - 0.5
	y := v*float64(t.H) - 0.5
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	fx := x - float64(x0)
	fy := y - float64(y0)
	c00 := t.At(wrapIndex(x0, t.W), wrapIndex(y0, t.H))
	c10 := t.At(wrapIndex(x0+1, t.W), wrapIndex(y0, t.H))
	c01 := t.At(wrapIndex(x0, t.W), wrapIndex(y0+1, t.H))
	c11 := t.At(wrapIndex(x0+1, t.W), wrapIndex(y0+1, t.H))
	return lerpColor(lerpColor(c00, c10, fx), lerpColor(c01, c11, fx), fy)
}

// SampleClamp 以连续纹素坐标做双线性采样，坐标钳制在纹理内。
// 图集页采样使用它：边缘外 1 纹素由 Gutter 的环绕拷贝保证正确。
func (t *Texture) SampleClamp(x, y float64) Color {
	x0 := int(math.Floor(x))
	y0 := int(math.Floor(y))
	fx := x - float64(x0)
	fy := y - float64(y0)
	c00 := t.At(clampi(x0, 0, t.W-1), clampi(y0, 0, t.H-1))
	c10 := t.At(clampi(x0+1, 0, t.W-1), clampi(y0, 0, t.H-1))
	c01 := t.At(clampi(x0, 0, t.W-1), clampi(y0+1, 0, t.H-1))
	c11 := t.At(clampi(x0+1, 0, t.W-1), clampi(y0+1, 0, t.H-1))
	return lerpColor(lerpColor(c00, c10, fx), lerpColor(c01, c11, fx), fy)
}
