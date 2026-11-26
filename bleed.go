package main

import (
	"image"
	"math"
)

// RenderBoundaryTest 生成边界串色测试图。
// 每种材质占一组三条横带：上为纹理数组采样、中为图集采样（同一组
// ShaderParams，U 方向每屏重复 4 次以反复跨越 0/1 缝）、下为两者
// 差值放大 16 倍的结果，纯黑表示图集没有串色、也没有可见 Gutter。
// 同时返回逐像素最大差值，供资源表记录。
func RenderBoundaryTest(array, atlas Sampler, materials int) (*image.RGBA, float64) {
	const (
		stripW = 768
		stripH = 24
		reps   = 4.0
	)
	img := image.NewRGBA(image.Rect(0, 0, stripW, stripH*3*materials))
	maxDiff := 0.0
	for m := 0; m < materials; m++ {
		base := m * 3 * stripH
		for y := 0; y < stripH; y++ {
			v := (float64(y) + 0.5) / float64(stripH)
			for x := 0; x < stripW; x++ {
				u := (float64(x) + 0.5) / float64(stripW) * reps
				p := ShaderParams{U: u, V: v, Material: m}
				ca := array.Sample(p)
				cb := atlas.Sample(p)
				d := diffColor(ca, cb)
				maxDiff = math.Max(maxDiff, math.Max(d.R, math.Max(d.G, d.B)))
				img.SetRGBA(x, base+y, toRGBA(ca))
				img.SetRGBA(x, base+stripH+y, toRGBA(cb))
				amp := Color{d.R * 16, d.G * 16, d.B * 16}
				img.SetRGBA(x, base+2*stripH+y, toRGBA(amp))
			}
		}
	}
	return img, maxDiff
}

// CheckLayerIsolation 验证纹理数组的层间隔离：
// 用越界层号采样，结果必须等于钳制后的合法层，不得读到相邻层。
func CheckLayerIsolation(a *TextureArray) bool {
	last := len(a.Layers) - 1
	uvs := [][2]float64{{0.13, 0.57}, {0.99, 0.01}, {2.7, -1.3}}
	for _, uv := range uvs {
		lo := a.Sample(ShaderParams{U: uv[0], V: uv[1], Material: -5})
		wantLo := a.Sample(ShaderParams{U: uv[0], V: uv[1], Material: 0})
		if lo != wantLo {
			return false
		}
		hi := a.Sample(ShaderParams{U: uv[0], V: uv[1], Material: last + 100})
		wantHi := a.Sample(ShaderParams{U: uv[0], V: uv[1], Material: last})
		if hi != wantHi {
			return false
		}
	}
	return true
}

// CheckAtlasGutter 验证图集 Gutter：区域内容区之外、Gutter 之内的
// 每个纹素必须等于内容区的环绕拷贝，确保双线性探针越界时不串色。
func CheckAtlasGutter(a *Atlas) bool {
	g := a.Gutter
	for _, r := range a.Regions {
		for dy := -g; dy < r.H+g; dy++ {
			for dx := -g; dx < r.W+g; dx++ {
				got := a.Page.At(r.X+dx, r.Y+dy)
				sx := r.X + wrapIndex(dx, r.W)
				sy := r.Y + wrapIndex(dy, r.H)
				if got != a.Page.At(sx, sy) {
					return false
				}
			}
		}
	}
	return true
}
