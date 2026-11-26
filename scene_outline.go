package main

import (
	"fmt"
	"image"
)

var outlineGlyphs = []rune{'永', '国', 'A', 'g', 'o', '8'}

func cardGrid(w, h float64, cols, rows int, margin, top, gapX, gapY float64) []Rect {
	cw := (w - 2*margin - float64(cols-1)*gapX) / float64(cols)
	ch := (h - top - margin - float64(rows-1)*gapY) / float64(rows)
	out := make([]Rect, 0, cols*rows)
	for r := 0; r < rows; r++ {
		for c := 0; c < cols; c++ {
			x0 := margin + float64(c)*(cw+gapX)
			y0 := top + float64(r)*(ch+gapY)
			out = append(out, Rect{MinX: x0, MinY: y0, MaxX: x0 + cw, MaxY: y0 + ch})
		}
	}
	return out
}

func cardContentArea(card Rect) Rect {
	return Rect{MinX: card.MinX + 30, MinY: card.MinY + 62,
		MaxX: card.MaxX - 30, MaxY: card.MaxY - 24}
}

// glyphFit builds font-unit outline and a fit transform into area.
func glyphFit(fs *FontSet, r rune, area Rect) (*Glyph, Outline, Fit, [][]Pt) {
	f, g := fs.Glyph(r)
	o := GlyphOutline(g, f.UnitsPerEm, float64(f.UnitsPerEm), 1.0)
	fit := FitRect(o.Bounds, area)
	raw := RawPixelContours(g, f.UnitsPerEm, float64(f.UnitsPerEm))
	return g, o, fit, raw
}

func drawBaselineGuides(sc *Scene, fit Fit, area Rect) {
	baseY := fit.P(Pt{0, 0}).Y
	penX := fit.P(Pt{0, 0}).X
	sc.HGuide(baseY, area.MinX-6, area.MaxX+6, rgbaf(0x93, 0xc5, 0xfd, 255))
	sc.VGuide(penX, area.MinY-6, area.MaxY+6, rgbaf(0xbf, 0xdb, 0xfe, 255))
}

func glyphCardChrome(sc *Scene, fs *FontSet, r rune, card Rect, detail string) Rect {
	sc.Panel(card.MinX, card.MinY, card.MaxX, card.MaxY, CPanel)
	sc.DrawText(fmt.Sprintf("%c  U+%04X", r, r), card.MinX+22, card.MinY+40, fs,
		TextStyle{Ppem: 24, Color: CInk, Bold: true})
	if detail != "" {
		st := TextStyle{Ppem: 15, Color: CMuted}
		w := fs.Measure(detail, st)
		sc.DrawText(detail, card.MaxX-22-w, card.MinY+37, fs, st)
	}
	return cardContentArea(card)
}

// RenderOriginalOutlines draws raw glyf contours with on/off-curve points.
func RenderOriginalOutlines(fs *FontSet) *image.RGBA {
	const W, H = 1680, 1180
	sc := NewScene(W, H, 3, CPaper)
	sc.Title("原始轮廓：TrueType 贝塞尔曲线与控制点",
		"程序直接解析 glyf / loca / cmap 表；方块=在曲线上点，菱形=二次曲线外控制点，灰线=控制多边形，蓝线=展平轮廓", fs)
	sc.Legend(50, 126, fs, []LegendItem{
		{Text: "在曲线点", Draw: func(s *Scene, cx, cy float64) { s.FillRect(cx-5, cy-5, cx+5, cy+5, CInk) }},
		{Text: "二次控制点", Draw: func(s *Scene, cx, cy float64) { s.Diamond(cx, cy, 7, CAmber, CWhite) }},
		{Text: "展平轮廓", Draw: func(s *Scene, cx, cy float64) { s.Line(cx-13, cy, cx+13, cy, 2.4, CBlue) }},
		{Text: "基线 / 原点", Draw: func(s *Scene, cx, cy float64) { s.DashedLine(cx-13, cy, cx+13, cy, 5, 3, 1.6, CBlue) }},
	})

	cards := cardGrid(W, H, 3, 2, 48, 158, 26, 26)
	for i, r := range outlineGlyphs {
		card := cards[i]
		_, g0 := fs.Glyph(r)
		nOn, nOff := 0, 0
		for _, rc := range g0.Contours {
			for _, rp := range rc {
				if rp.OnCur {
					nOn++
				} else {
					nOff++
				}
			}
		}
		area := glyphCardChrome(sc, fs, r, card,
			fmt.Sprintf("%d 条轮廓 · %d 在上点 · %d 控制点", len(g0.Contours), nOn, nOff))
		_, o, fit, raw := glyphFit(fs, r, area)
		drawBaselineGuides(sc, fit, area)
		fo := fit.Outline(o)
		sc.FillOutline(fo, rgbaf(0xea, 0xf1, 0xff, 255))
		sc.StrokeOutline(fo, 2.4, CBlue)

		for _, rc := range raw {
			mapped := make([]Pt, len(rc)+1)
			for j, p := range rc {
				mapped[j] = fit.P(p)
			}
			mapped[len(rc)] = mapped[0]
			sc.Polyline(mapped, 1.1, rgbaf(0xc4, 0xcd, 0xda, 255))
		}
		for ci, rc := range g0.Contours {
			for j, rp := range rc {
				p := fit.P(raw[ci][j])
				if rp.OnCur {
					sc.FillRect(p.X-4.5, p.Y-4.5, p.X+4.5, p.Y+4.5, CInk)
					sc.FrameRect(p.X-4.5, p.Y-4.5, p.X+4.5, p.Y+4.5, 1.2, CWhite)
				} else {
					sc.Diamond(p.X, p.Y, 6.5, CAmber, CWhite)
				}
			}
		}
	}
	return sc.Render()
}

// RenderOutlineSampling draws arc-length evenly distributed points.
func RenderOutlineSampling(fs *FontSet) *image.RGBA {
	const W, H = 1680, 1180
	sc := NewScene(W, H, 3, CPaper)
	sc.Title("轮廓布点：按弧长均匀采样（含切向 / 法向）",
		"展平曲线后累计弧长，再按等弧长间隔反解采样位置；红点=布点，紫线=单位法向，绿色双圈=每条轮廓首点", fs)
	sc.Legend(50, 126, fs, []LegendItem{
		{Text: "等弧长采样点", Draw: func(s *Scene, cx, cy float64) {
			s.Disc(cx, cy, 6, CRed)
			s.Disc(cx, cy, 2.4, CWhite)
		}},
		{Text: "单位法向", Draw: func(s *Scene, cx, cy float64) { s.Line(cx, cy+9, cx, cy-9, 2, CPurple) }},
		{Text: "轮廓", Draw: func(s *Scene, cx, cy float64) { s.Line(cx-13, cy, cx+13, cy, 2.2, CMuted) }},
	})

	cards := cardGrid(W, H, 3, 2, 48, 158, 26, 26)
	for i, r := range outlineGlyphs {
		card := cards[i]
		area0 := cardContentArea(card)
		_, o0, fit0, _ := glyphFit(fs, r, area0)
		totalPts := 0
		for _, c := range fit0.Outline(o0).Contours {
			totalPts += len(c.SampleSpacing(30))
		}
		area := glyphCardChrome(sc, fs, r, card,
			fmt.Sprintf("%d 条轮廓 · %d 个等弧长布点", len(o0.Contours), totalPts))
		_, o, fit, _ := glyphFit(fs, r, area)
		fo := fit.Outline(o)
		drawBaselineGuides(sc, fit, area)
		sc.FillOutline(fo, rgbaf(0xf1, 0xf5, 0xf9, 255))
		sc.StrokeOutline(fo, 2.0, CBorder)

		for _, c := range fo.Contours {
			samples := c.SampleSpacing(30)
			for k, smp := range samples {
				if k%6 == 2 {
					sc.Line(smp.Pos.X, smp.Pos.Y,
						smp.Pos.X+smp.Normal.X*16, smp.Pos.Y+smp.Normal.Y*16, 1.7, CPurple)
				}
				sc.Disc(smp.Pos.X, smp.Pos.Y, 6.2, CWhite)
				sc.Disc(smp.Pos.X, smp.Pos.Y, 3.7, CRed)
			}
			if len(samples) > 0 {
				p0 := samples[0]
				sc.Disc(p0.Pos.X, p0.Pos.Y, 9, CGreen)
				sc.Disc(p0.Pos.X, p0.Pos.Y, 4, CWhite)
			}
		}
	}
	return sc.Render()
}
