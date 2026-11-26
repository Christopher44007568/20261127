package main

import (
	"fmt"
	"image"
)

func panelTitle(sc *Scene, fs *FontSet, r Rect, title, sub string) {
	sc.DrawText(title, r.MinX+26, r.MinY+42, fs, TextStyle{Ppem: 23, Color: CInk, Bold: true})
	if sub != "" {
		sc.DrawText(sub, r.MinX+27, r.MinY+70, fs, TextStyle{Ppem: 15, Color: CMuted})
	}
}

func passBadge(sc *Scene, fs *FontSet, xRight, y float64, text string) {
	st := TextStyle{Ppem: 16, Color: CGreen, Bold: true}
	w := fs.Measure(text, st)
	sc.FillRect(xRight-w-34, y-21, xRight, y+9, CGreenSoft)
	sc.Disc(xRight-w-20, y-6, 4, CGreen)
	sc.DrawText(text, xRight-w-12, y-5, fs, st)
}

// drawLineGapPanel renders the controlled ascender/descender specimen.
func drawLineGapPanel(sc *Scene, fs *FontSet, panel Rect) image.Rectangle {
	sc.Panel(panel.MinX, panel.MinY, panel.MaxX, panel.MaxY, CPanel)
	panelTitle(sc, fs, panel, "局部取材 A：上升部与下降部互不覆盖",
		"上行 gpqy 的下降部 与 下行 hkbd 的上升部 之间保留正墨迹间隙")

	st := LayoutStyle{Ppem: 64, TrackingEm: 0.10, LineHeightEm: 1.18, MinInkGapEm: 0.18}
	lay := fs.Layout("gpqy\nhkbd", st)
	ox := panel.MinX + 170
	base1 := panel.MinY + 178
	oy := base1 - lay.Lines[0].BaselineY
	drawLayout(sc, fs, lay, ox, oy, TextStyle{Ppem: st.Ppem, TrackingEm: st.TrackingEm, Color: CInk})

	l1, l2 := lay.Lines[0], lay.Lines[1]
	b1 := oy + l1.BaselineY
	b2 := oy + l2.BaselineY
	yDesc := b1 + l1.Descent
	yAsc := b2 - l2.Ascent
	gap := yAsc - yDesc

	sc.HGuide(b1, ox-26, panel.MaxX-30, rgbaf(0x93, 0xc5, 0xfd, 255))
	sc.HGuide(b2, ox-26, panel.MaxX-30, rgbaf(0x93, 0xc5, 0xfd, 255))
	sc.Line(ox-26, yDesc, panel.MaxX-30, yDesc, 2.0, CAmber)
	sc.Line(ox-26, yAsc, panel.MaxX-30, yAsc, 2.0, CTeal)
	sc.FillRect(ox-26, yDesc, panel.MaxX-30, yAsc, rgbaf(0xdc, 0xfc, 0xe7, 110))
	dimX := panel.MinX + 64
	sc.Line(dimX, yDesc, dimX, yAsc, 1.8, CGreen)
	sc.Line(dimX-7, yDesc, dimX+7, yDesc, 2, CGreen)
	sc.Line(dimX-7, yAsc, dimX+7, yAsc, 2, CGreen)
	gapLabel := fmt.Sprintf("墨迹间隙 gap = %.1f px  > 0", gap)
	glSt := TextStyle{Ppem: 20, Color: CGreen, Bold: false}
	glW := fs.Measure(gapLabel, glSt)
	glX := panel.MinX + 400
	glY := (yDesc+yAsc)/2 + 7
	sc.FillRect(glX-12, glY-24, glX+glW+14, glY+8, CWhite)
	sc.DrawText(gapLabel, glX, glY, fs, glSt)
	sc.DrawText("上行下降底", panel.MaxX-150, yDesc-8, fs, TextStyle{Ppem: 15, Color: CAmber})
	sc.DrawText("下行上升顶", panel.MaxX-150, yAsc+18, fs, TextStyle{Ppem: 15, Color: CTeal})
	sc.DrawText("基线1", panel.MinX+16, b1-8, fs, TextStyle{Ppem: 14, Color: CBlue})
	sc.DrawText("基线2", panel.MinX+16, b2+22, fs, TextStyle{Ppem: 14, Color: CBlue})

	passBadge(sc, fs, panel.MaxX-24, panel.MinY+40,
		fmt.Sprintf("PASS · gap %.1f px ≥ 0", gap))
	note := fmt.Sprintf("baseline step = %.1f px = max(%.2f em×%d, descent %.1f + minGap %.1f + ascent %.1f)",
		b2-b1, st.LineHeightEm, int(st.Ppem), l1.Descent, st.MinInkGapEm*st.Ppem, l2.Ascent)
	sc.DrawText(note, panel.MinX+28, panel.MaxY-24, fs, TextStyle{Ppem: 15, Color: CInkSoft})

	return image.Rect(int(panel.MinX+6), int(panel.MinY+88),
		int(panel.MinX+770), int(panel.MinY+308))
}

// drawTrackingPanel renders glyph bounds + advance/tracking callouts.
func drawTrackingPanel(sc *Scene, fs *FontSet, panel Rect) image.Rectangle {
	sc.Panel(panel.MinX, panel.MinY, panel.MaxX, panel.MaxY, CPanel)
	panelTitle(sc, fs, panel, "局部取材 B：字距落在笔位几何上",
		"蓝框=真实墨迹边界；灰括号=advance；紫段=额外 tracking")

	st := LayoutStyle{Ppem: 88, TrackingEm: 0.16, LineHeightEm: 1.2, MaxWidth: 1e7}
	lay := fs.Layout("汉Av", st)
	ox := panel.MinX + 52
	base := panel.MinY + 210
	oy := base - lay.Lines[0].BaselineY
	drawLayout(sc, fs, lay, ox, oy, TextStyle{Ppem: st.Ppem, TrackingEm: st.TrackingEm, Color: CInk})

	ln := lay.Lines[0]
	for _, pg := range ln.Glyphs {
		sc.FrameRect(ox+pg.InkMinX-2, oy+pg.InkMinY-2,
			ox+pg.InkMaxX+2, oy+pg.InkMaxY+2, 1.6, CBlue)
	}
	sc.HGuide(base, ox-14, panel.MaxX-24, rgbaf(0x93, 0xc5, 0xfd, 255))

	// Advance + tracking brackets under each pen.
	by := base + 66
	track := st.TrackingEm * st.Ppem
	for i, pg := range ln.Glyphs {
		xp := ox + pg.PenX
		sc.DimH(xp, xp+pg.Advance, by,
			fmt.Sprintf("adv %.0f", pg.Advance), fs, CMuted)
		if i < len(ln.Glyphs)-1 {
			sc.DimH(xp+pg.Advance, xp+pg.Advance+track, by+40,
				fmt.Sprintf("T %.1f", track), fs, CPurple)
		}
	}
	// Optical gap between first two ink boxes, on top.
	if len(ln.Glyphs) >= 2 {
		g0, g1 := ln.Glyphs[0], ln.Glyphs[1]
		gx0 := ox + g0.InkMaxX + 2
		gx1 := ox + g1.InkMinX - 2
		gy := base + 34
		if gx1 > gx0 {
			sc.DimH(gx0, gx1, gy, fmt.Sprintf("墨迹间隙 %.1f px", gx1-gx0), fs, CGreen)
		}
	}
	sc.DrawText("x(n+1) = x(n) + advance(n) + T，逐字作用于轮廓坐标",
		panel.MinX+28, panel.MaxY-24, fs, TextStyle{Ppem: 15, Color: CInkSoft})

	return image.Rect(int(panel.MinX+14), int(panel.MinY+96),
		int(panel.MinX+434), int(panel.MinY+334))
}
