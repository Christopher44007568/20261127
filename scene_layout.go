package main

import (
	"fmt"
	"image"
	"image/color"
)

func demoParagraph() string {
	return "排版必须按真实轮廓边界计算：字距 Tracking 与行高 Leading 来自 glyf 包围盒，而不是文本框。" +
		"The quick brown fox jumps over the lazy dog. " +
		"相邻行的上升部 hkbdflt 与下降部 gjpqy 不得相互覆盖；中英文 Mixed 混排 0123456789。"
}

func mainLayoutStyle() LayoutStyle {
	return LayoutStyle{
		Ppem:         31,
		TrackingEm:   0.055,
		LineHeightEm: 1.62,
		MinInkGapEm:  0.16,
		MaxWidth:     1470,
	}
}

// LayoutSheet holds the rendered sheet plus pixel rects useful for zoom crops.
type LayoutSheet struct {
	Img             *image.RGBA
	LineGapCrop     image.Rectangle
	TrackingCrop    image.Rectangle
	LineGapPngSize  image.Rectangle
	TrackingPngSize image.Rectangle
}

// RenderLayoutSheet builds the annotated typography proof sheet.
func RenderLayoutSheet(fs *FontSet) LayoutSheet {
	const W = 1680
	sty := mainLayoutStyle()
	lay := fs.Layout(demoParagraph(), sty)

	paraX := 88.0
	paraCardX0, paraCardX1 := 48.0, 1632.0
	statY0, statY1 := 118.0, 236.0
	paraCardY0 := 258.0
	textTop := paraCardY0 + 84
	paraCardY1 := textTop + lay.Height + 40

	lgRect := Rect{MinX: 48, MinY: paraCardY1 + 28, MaxX: 1010, MaxY: paraCardY1 + 470}
	tkRect := Rect{MinX: 1034, MinY: paraCardY1 + 28, MaxX: 1632, MaxY: paraCardY1 + 470}
	H := int(lgRect.MaxY) + 44

	sc := NewScene(W, H, 3, CPaper)
	sc.Title("排版结果：轮廓级字距、行高与字形边界",
		"每个字形按 glyf 真实包围盒定位；蓝虚线=基线，青线=上升墨迹顶，橙线=下降墨迹底，绿色尺寸=相邻行真实墨迹间隙", fs)
	drawStatCards(sc, fs, lay, 48, statY0, 1632, statY1)

	// Paragraph card.
	sc.Panel(paraCardX0, paraCardY0, paraCardX1, paraCardY1, CPanel)
	sc.Pill(paraCardX0+24, paraCardY0+38, "自动换行 · 中英混排 · 轮廓度量（首行显示字形包围盒）", fs, CBlue, CBlueSoft)
	drawLayout(sc, fs, lay, paraX, textTop, TextStyle{
		Ppem: sty.Ppem, TrackingEm: sty.TrackingEm, Color: CInk,
	})
	drawParagraphGuides(sc, fs, lay, paraX, textTop)

	// Specimen panels.
	lgCrop := drawLineGapPanel(sc, fs, lgRect)
	tkCrop := drawTrackingPanel(sc, fs, tkRect)

	img := sc.Render()
	return LayoutSheet{
		Img:          img,
		LineGapCrop:  lgCrop,
		TrackingCrop: tkCrop,
	}
}

func drawStatCards(sc *Scene, fs *FontSet, lay *Layout, x0, y0, x1, y1 float64) {
	sty := lay.Style
	type card struct {
		k, v, sub string
		accent    color.RGBA
	}
	cards := []card{
		{"排版行数", fmt.Sprintf("%d 行", len(lay.Lines)), "greedy wrap / CJK break", CBlue},
		{"相邻行最小墨迹间隙", fmt.Sprintf("%.1f px", lay.MinPairGap),
			fmt.Sprintf("下限 %.1f px · 无重叠", sty.MinInkGapEm*sty.Ppem), CGreen},
		{"字距 Tracking", fmt.Sprintf("+%.3f em", sty.TrackingEm),
			fmt.Sprintf("每字推进 +%.1f px", sty.TrackingEm*sty.Ppem), CPurple},
		{"最小基线步距", fmt.Sprintf("%.2f em", sty.LineHeightEm),
			"max(行高倍数, 下降+间隙+上升)", CAmber},
	}
	gap := 22.0
	cw := (x1 - x0 - gap*3) / 4
	for i, c := range cards {
		cx := x0 + float64(i)*(cw+gap)
		sc.Panel(cx, y0, cx+cw, y1, CPanel)
		sc.FillRect(cx, y0, cx+6, y1, c.accent)
		sc.DrawText(c.k, cx+22, y0+34, fs, TextStyle{Ppem: 17, Color: CMuted})
		sc.DrawText(c.v, cx+20, y0+78, fs, TextStyle{Ppem: 34, Color: CInk, Bold: true})
		sc.DrawText(c.sub, cx+22, y0+106, fs, TextStyle{Ppem: 15, Color: CInkSoft})
	}
}

func drawLayout(sc *Scene, fs *FontSet, lay *Layout, ox, oy float64, st TextStyle) {
	st.Bold = lay.Style.Bold
	for _, ln := range lay.Lines {
		for _, pg := range ln.Glyphs {
			sc.DrawRune(fs, rune(pg.Code), ox+pg.PenX, oy+ln.BaselineY, st)
		}
	}
}

func drawParagraphGuides(sc *Scene, fs *FontSet, lay *Layout, ox, oy float64) {
	x0 := ox - 18
	x1 := ox + lay.Style.MaxWidth + 18
	for i, ln := range lay.Lines {
		by := oy + ln.BaselineY
		sc.HGuide(by, x0, x1, rgbaf(0x93, 0xc5, 0xfd, 220))
		top := by - ln.Ascent
		bot := by + ln.Descent
		sc.Line(ox-8, top, x1, top, 1.1, rgbaf(0x99, 0xf6, 0xe4, 180))
		sc.Line(ox-8, bot, x1, bot, 1.1, rgbaf(0xfd, 0xba, 0x74, 180))
		if i == 0 {
			for _, pg := range ln.Glyphs {
				if isSpaceRune(rune(pg.Code)) {
					continue
				}
				c := rgbaf(0xbf, 0xdb, 0xfe, 200)
				sc.FrameRect(ox+pg.InkMinX-1, oy+pg.InkMinY-1,
					ox+pg.InkMaxX+1, oy+pg.InkMaxY+1, 1.1, c)
			}
		}
		if i > 0 {
			prev := lay.Lines[i-1]
			yDescent := oy + prev.BaselineY + prev.Descent
			yAscent := by - ln.Ascent
			gap := yAscent - yDescent
			xg := ox - 30
			sc.FillRect(xg-9, yDescent, xg+9, yAscent, rgbaf(0xdc, 0xfc, 0xe7, 160))
			sc.DimV(yDescent, yAscent, xg, fmt.Sprintf("%.1f", gap), fs, CGreen)
		}
	}
}
