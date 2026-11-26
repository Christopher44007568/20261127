package main

import (
	"image"
)

var overviewNames = []struct{ file, title string }{
	{"01_original_outlines.png", "01 原始轮廓 / 控制点"},
	{"02_outline_points.png", "02 轮廓布点 / 法向"},
	{"03_layout_result.png", "03 排版结果（几何标注）"},
	{"04_layout_clean.png", "04 干净排版输出"},
	{"05_stroke_preview.png", "05 描边预览 / 粗体"},
	{"06_zoom_linegap.png", "06 放大：上升/下降部间隙"},
	{"07_zoom_tracking.png", "07 放大：字距笔位"},
	{"08_zoom_counters.png", "08 放大：粗体字腔保护"},
	{"09_zoom_points.png", "09 放大：布点细节"},
}

// ThumbFit downscales with repeated box halving then nearest fit.
func ThumbFit(src *image.RGBA, boxW, boxH int) *image.RGBA {
	cur := src
	for cur.Bounds().Dx()/2 >= boxW*5/4 && cur.Bounds().Dy()/2 >= boxH*5/4 {
		cur = DownscaleAverage(cur, 2)
	}
	b := cur.Bounds()
	aspect := float64(b.Dy()) / float64(b.Dx())
	w := boxW
	h := int(float64(w) * aspect)
	if h > boxH {
		h = boxH
		w = int(float64(h) / aspect)
	}
	out := image.NewRGBA(image.Rect(0, 0, w, h))
	PasteNearest(out, cur, 0, 0, w, h)
	return out
}

// RenderOverview builds a contact sheet linking every deliverable.
func RenderOverview(fs *FontSet, imgs map[string]*image.RGBA) *image.RGBA {
	const W = 1760
	cols := 3
	margin := 40.0
	gap := 22.0
	cardW := (W - 2*margin - float64(cols-1)*gap) / 3
	cardH := 420.0
	top := 140.0
	rows := 3
	H := int(top + float64(rows)*(cardH+gap) + margin)
	sc := NewScene(W, H, 2, CPaper)
	sc.Title("总览：轮廓生成 · 布点 · 行布局 · 描边 · 局部放大",
		"纯 Go 标准库实现；以下九张图即 artifacts/ 中的 01–09 号产物", fs)

	for i, item := range overviewNames {
		r := i / cols
		c := i % cols
		x0 := margin + float64(c)*(cardW+gap)
		y0 := top + float64(r)*(cardH+gap)
		sc.Panel(x0, y0, x0+cardW, y0+cardH, CPanel)
		sc.DrawText(item.title, x0+18, y0+34, fs, TextStyle{Ppem: 19, Color: CInk, Bold: true})
		src := imgs[item.file]
		if src == nil {
			continue
		}
		thumb := ThumbFit(src, int(cardW-36), int(cardH-78))
		tb := thumb.Bounds()
		tx := int(x0 + (cardW-36-float64(tb.Dx()))/2 + 18)
		ty := int(y0 + 52 + (cardH-78-float64(tb.Dy()))/2)
		PasteRGBA(sc.img, thumbAtDeviceScale(thumb, sc.S), tx*sc.S, ty*sc.S)
	}
	return sc.Render()
}

// thumbAtDeviceScale duplicates a final-res thumb into the SS device buffer.
func thumbAtDeviceScale(thumb *image.RGBA, ss int) *image.RGBA {
	if ss == 1 {
		return thumb
	}
	b := thumb.Bounds()
	out := image.NewRGBA(image.Rect(0, 0, b.Dx()*ss, b.Dy()*ss))
	PasteNearest(out, thumb, 0, 0, b.Dx()*ss, b.Dy()*ss)
	return out
}
