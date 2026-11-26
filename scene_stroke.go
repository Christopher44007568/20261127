package main

import (
	"fmt"
	"image"
)

// RenderStrokePreview shows fill, outline strokes and counter-safe bold.
func RenderStrokePreview(fs *FontSet) *image.RGBA {
	const W, H = 1680, 1080
	sc := NewScene(W, H, 3, CPaper)
	sc.Title("描边预览：从原始轮廓到字腔安全粗体",
		"粗体 = 填充 ∪ 沿轮廓居中描边（宽度 2r，单侧膨胀 r）；r 由字腔检测逐字回退，密集汉字不会被吞掉字腔", fs)

	glyphs := []rune{'A', 'g', '永', '国'}
	colX0, colX1 := 236.0, 1632.0
	cw := (colX1 - colX0) / float64(len(glyphs))
	for i, r := range glyphs {
		cx := colX0 + cw*(float64(i)+0.5)
		sc.DrawText(fmt.Sprintf("%c  U+%04X", r, r), cx-55, 158, fs,
			TextStyle{Ppem: 22, Color: CInkSoft, Bold: true})
	}

	type row struct {
		label string
		sub   string
	}
	rows := []row{
		{"常规填充", "regular fill"},
		{"细描边", "stroke 2.4 px"},
		{"粗描边", "stroke 8 px"},
		{"安全粗体", "bold + counter guard"},
	}
	rowY0 := 182.0
	rh := 205.0
	ppem := 132.0
	for ri, rd := range rows {
		y0 := rowY0 + float64(ri)*rh
		sc.Panel(48, y0, 1632, y0+rh-18, CPanel)
		sc.DrawText(rd.label, 74, y0+78, fs, TextStyle{Ppem: 24, Color: CInk, Bold: true})
		sc.DrawText(rd.sub, 75, y0+106, fs, TextStyle{Ppem: 15, Color: CMuted})
		for ci, r := range glyphs {
			cx := colX0 + cw*(float64(ci)+0.5)
			cy := y0 + 128
			ink := fs.RuneInk(r, ppem, false)
			gcx := (ink.MinX + ink.MaxX) / 2
			penX := cx - gcx
			switch ri {
			case 0:
				sc.DrawRune(fs, r, penX, cy, TextStyle{Ppem: ppem, Color: CInk})
			case 1:
				_, _, o := fs.OutlineFor(r, ppem, 0.2)
				sc.StrokeOutline(o.Translate(penX, cy), 2.4, CBlue)
			case 2:
				_, _, o := fs.OutlineFor(r, ppem, 0.2)
				sc.StrokeOutline(o.Translate(penX, cy), 8, CTeal)
			case 3:
				rep := fs.BoldReport(r)
				rad := rep.ChosenEm * ppem
				sc.DrawRuneRadius(fs, r, penX, cy,
					TextStyle{Ppem: ppem, Color: CInk}, rad)
				tag := fmt.Sprintf("r=%.3f em", rep.ChosenEm)
				if rep.Limited {
					tag += " (已回退)"
				}
				tw := fs.Measure(tag, TextStyle{Ppem: 15})
				col := CGreen
				sc.DrawText(tag, cx-tw/2, y0+rh-30, fs, TextStyle{Ppem: 15, Color: col, Bold: true})
			}
		}
	}
	return sc.Render()
}
