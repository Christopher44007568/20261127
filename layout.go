package main

import "math"

// ShaderParams 为两种布局共享的着色参数：
// 同一份材质球 shader 只依赖这三个字段，不关心底层是数组还是图集。
type ShaderParams struct {
	U, V     float64 // 归一化 UV，允许超出 [0,1]（重复寻址）
	Material int     // 材质索引：数组下是层号，图集下是区域号
}

// Sampler 为布局抽象：shader 只面向该接口采样。
type Sampler interface {
	Sample(p ShaderParams) Color
}

// ---------- 布局一：纹理数组 ----------

// TextureArray 每层独立存储，层号越界时钳制，
// 双线性只在单层内部进行，物理上不可能串到下一个材质。
type TextureArray struct {
	Layers []*Texture
}

func NewTextureArray(layers []*Texture) *TextureArray {
	return &TextureArray{Layers: layers}
}

func (a *TextureArray) Sample(p ShaderParams) Color {
	layer := clampi(p.Material, 0, len(a.Layers)-1)
	return a.Layers[layer].SampleWrap(p.U, p.V)
}

// ---------- 布局二：带 Gutter 的纹理图集 ----------

// AtlasRegion 描述材质内容区（不含 Gutter）在图集页中的像素矩形。
type AtlasRegion struct {
	X, Y, W, H int
}

// Atlas 把所有材质打包进单张页，每个区域四周填充 Gutter。
// Gutter 不是边缘复制，而是对侧边缘的环绕拷贝（左 Gutter = 内容最右列），
// 因此双线性探针越出内容区 1 纹素时读到的仍是本材质应有的环绕值，
// 既不串到相邻材质，也不会在画面上留下可见的 Gutter 缝。
type Atlas struct {
	Page    *Texture
	Regions []AtlasRegion
	Gutter  int
}

func BuildAtlas(layers []*Texture, gutter int) *Atlas {
	n := len(layers)
	cols := int(math.Ceil(math.Sqrt(float64(n))))
	rows := (n + cols - 1) / cols
	w, h := layers[0].W, layers[0].H
	cellW, cellH := w+2*gutter, h+2*gutter
	page := NewTexture(cols*cellW, rows*cellH)
	atlas := &Atlas{Page: page, Gutter: gutter}

	for i, layer := range layers {
		col := i % cols
		row := i / cols
		ox := col*cellW + gutter // 内容区原点
		oy := row*cellH + gutter
		atlas.Regions = append(atlas.Regions, AtlasRegion{X: ox, Y: oy, W: w, H: h})

		// 一次性写入内容区 + 环绕拷贝 Gutter（含四角）：
		// 对单元格内每个偏移 (dx,dy)，取内容区中环绕后的源纹素。
		for dy := -gutter; dy < h+gutter; dy++ {
			sy := wrapIndex(dy, h)
			for dx := -gutter; dx < w+gutter; dx++ {
				sx := wrapIndex(dx, w)
				page.Set(ox+dx, oy+dy, layer.At(sx, sy))
			}
		}
	}
	return atlas
}

func (a *Atlas) Sample(p ShaderParams) Color {
	r := a.Regions[clampi(p.Material, 0, len(a.Regions)-1)]
	u := p.U - math.Floor(p.U)
	v := p.V - math.Floor(p.V)
	// 把 UV 映射为图集页上的连续纹素坐标，直接对整页双线性。
	// 边缘处探针最多越出内容区 0.5 纹素，落在 Gutter 内，
	// 而 Gutter 是环绕拷贝，结果与数组的 SampleWrap 完全一致。
	x := float64(r.X) + u*float64(r.W) - 0.5
	y := float64(r.Y) + v*float64(r.H) - 0.5
	return a.Page.SampleClamp(x, y)
}
