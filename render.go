package main

import (
	"image"
	"image/color"
	"math"
)

func toRGBA(c Color) color.RGBA {
	c = c.clamp01()
	return color.RGBA{
		R: uint8(c.R*255 + 0.5),
		G: uint8(c.G*255 + 0.5),
		B: uint8(c.B*255 + 0.5),
		A: 255,
	}
}

// shadeBall 用共享的 ShaderParams 路径渲染一颗材质球。
// 球面 UV 在经度方向跨 0/1 缝，极点处 UV 剧烈压缩，
// 正好反复考验两种布局的边界采样。
func shadeBall(s Sampler, material int, img *image.RGBA, ox, oy, size int) {
	radius := float64(size)/2 - 4
	cx := float64(ox) + float64(size)/2
	cy := float64(oy) + float64(size)/2
	light := [3]float64{-0.45, -0.65, 0.62}
	il := 1.0 / math.Sqrt(light[0]*light[0]+light[1]*light[1]+light[2]*light[2])
	for i := range light {
		light[i] *= il
	}

	for py := 0; py < size; py++ {
		for px := 0; px < size; px++ {
			nx := (float64(ox+px) + 0.5 - cx) / radius
			ny := (float64(oy+py) + 0.5 - cy) / radius
			r2 := nx*nx + ny*ny
			if r2 > 1 {
				continue
			}
			nz := math.Sqrt(1 - r2)
			u := math.Atan2(nx, nz)/(2*math.Pi) + 0.5
			v := math.Acos(clampf(ny, -1, 1)) / math.Pi
			tex := s.Sample(ShaderParams{U: u * 2, V: v * 2, Material: material})

			diff := math.Max(0, nx*light[0]+ny*light[1]+nz*light[2])
			half := [3]float64{light[0], light[1], light[2] + 1}
			ih := 1.0 / math.Sqrt(half[0]*half[0]+half[1]*half[1]+half[2]*half[2])
			ndh := nx*half[0]*ih + ny*half[1]*ih + nz*half[2]*ih
			spec := math.Pow(math.Max(0, ndh), 48) * 0.35
			lum := 0.25 + 0.75*diff
			c := Color{tex.R*lum + spec, tex.G*lum + spec, tex.B*lum + spec}
			img.SetRGBA(ox+px, oy+py, toRGBA(c))
		}
	}
}

// RenderBallCollage 生成材质球拼图：上排纹理数组、下排图集，
// 8 种材质一一对应，肉眼即可对比两种布局输出是否一致。
func RenderBallCollage(array, atlas Sampler, materials int) *image.RGBA {
	cell := 160
	img := image.NewRGBA(image.Rect(0, 0, cell*materials, cell*2))
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			g := uint8(30)
			if (x/10+y/10)%2 == 0 {
				g = 38
			}
			img.SetRGBA(x, y, color.RGBA{g, g, g, 255})
		}
	}
	for m := 0; m < materials; m++ {
		shadeBall(array, m, img, m*cell, 0, cell)
		shadeBall(atlas, m, img, m*cell, cell, cell)
	}
	return img
}
