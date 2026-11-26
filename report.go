package main

import (
	"fmt"
	"strings"
)

// LayoutStats 汇总一种布局的资源占用，用于资源表。
type LayoutStats struct {
	Name       string // 布局名称
	Textures   int    // 纹理对象数
	Binds      int    // 绘制 8 种材质所需绑定次数
	Gutter     int    // 每侧 Gutter 纹素数
	PageW      int    // 页/层 宽
	PageH      int    // 页/层 高
	TotalPixel int    // 总纹素
	Note       string // 校验结论
}

func (s LayoutStats) bytes() int {
	return s.TotalPixel * 4 // RGBA8
}

func layerNote(ok bool) string {
	if ok {
		return "通过：层号钳制，双线性不跨层"
	}
	return "失败：层间隔离校验未通过"
}

func gutterNote(ok bool) string {
	if ok {
		return "Gutter 环绕拷贝校验通过"
	}
	return "Gutter 校验未通过"
}

func buildStats(array *TextureArray, atlas *Atlas, maxDiff float64, layerOK, gutterOK bool) []LayoutStats {
	w, h := MaterialSize, MaterialSize
	n := len(array.Layers)
	verdict := "通过：与数组逐像素一致，无串色、无可见 Gutter"
	if maxDiff > 1e-9 {
		verdict = fmt.Sprintf("失败：最大差值 %.3e", maxDiff)
	}
	return []LayoutStats{
		{
			Name: "独立纹理（基线）", Textures: n, Binds: n, Gutter: 0,
			PageW: w, PageH: h, TotalPixel: n * w * h,
			Note: "每种材质一次绑定，无共享 shader 参数",
		},
		{
			Name: "纹理数组", Textures: 1, Binds: 1, Gutter: 0,
			PageW: w, PageH: h, TotalPixel: n * w * h,
			Note: layerNote(layerOK),
		},
		{
			Name: "图集（环绕拷贝 Gutter）", Textures: 1, Binds: 1, Gutter: atlas.Gutter,
			PageW: atlas.Page.W, PageH: atlas.Page.H,
			TotalPixel: atlas.Page.W * atlas.Page.H,
			Note:       verdict + "；" + gutterNote(gutterOK),
		},
	}
}

// RenderResourceTable 输出 Markdown 资源表。
func RenderResourceTable(stats []LayoutStats, maxDiff float64) string {
	base := stats[0].TotalPixel
	var b strings.Builder
	b.WriteString("# 纹理布局资源表\n\n")
	b.WriteString("同一组 shader 参数（U, V, Material）驱动两种布局；")
	b.WriteString("材质 8 种，每种 128x128 RGBA8。\n\n")
	b.WriteString("| 布局 | 纹理对象 | 绑定次数 | Gutter(px) | 页尺寸 | 总纹素 | 内存 | 相对基线 | 边界校验 |\n")
	b.WriteString("| --- | ---: | ---: | ---: | --- | ---: | ---: | ---: | --- |\n")
	for _, s := range stats {
		overhead := float64(s.TotalPixel-base) / float64(base) * 100
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %dx%d | %d | %d B | %+.1f%% | %s |\n",
			s.Name, s.Textures, s.Binds, s.Gutter, s.PageW, s.PageH,
			s.TotalPixel, s.bytes(), overhead, s.Note)
	}
	fmt.Fprintf(&b, "\n数组 vs 图集逐像素最大差值：%.3e（≤1e-9 视为完全一致，仅浮点舍入噪声）\n", maxDiff)
	return b.String()
}
