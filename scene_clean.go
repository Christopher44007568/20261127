package main

import (
	"image"
)

// RenderCleanLayout is the un-annotated typographic output.
func RenderCleanLayout(fs *FontSet) *image.RGBA {
	const W = 1680
	x := 120.0
	maxW := W - 240.0

	head := fs.Layout("中英文轮廓排版 Outline Typography", LayoutStyle{
		Ppem: 58, TrackingEm: 0.01, LineHeightEm: 1.18, MaxWidth: maxW, Bold: true,
	})
	body1 := fs.Layout(demoParagraph(), LayoutStyle{
		Ppem: 29, TrackingEm: 0.03, LineHeightEm: 1.78, MinInkGapEm: 0.1, MaxWidth: maxW,
	})
	body2 := fs.Layout(
		"字距加在每一次笔位推进上，行高由上升墨迹、下降墨迹与最小间隙共同决定；"+
			"粗体经过字腔复测，例如 a / B / 8 与 国 / 圆 / 量 会使用各自最大的安全粗度。"+
			"Aa Bb Gg Pp Qq Yy 012369", LayoutStyle{
			Ppem: 29, TrackingEm: 0.03, LineHeightEm: 1.78, MinInkGapEm: 0.1, MaxWidth: maxW,
		})

	H := 150 + head.Height + 40 + 4 + 44 + body1.Height + 34 + body2.Height + 90
	sc := NewScene(W, int(H), 3, CWhite)
	sc.DrawText("OUTPUT PREVIEW", x, 92, fs, TextStyle{Ppem: 18, Color: CBlue, Bold: true})

	y := 150.0
	drawLayout(sc, fs, head, x, y, TextStyle{Ppem: 58, Color: CInk, Bold: true})
	y += head.Height + 34
	sc.FillRect(x, y, x+120, y+5, CBlue)
	y += 44
	drawLayout(sc, fs, body1, x, y, TextStyle{Ppem: 29, Color: CInk})
	y += body1.Height + 30
	drawLayout(sc, fs, body2, x, y, TextStyle{Ppem: 29, Color: CInkSoft})

	sc.DrawText("outline-lab · pure Go stdlib · TTF glyf + arc-length sampling + SSAA",
		x, H-44, fs, TextStyle{Ppem: 16, Color: CMuted})
	return sc.Render()
}
