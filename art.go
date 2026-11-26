package main

import (
	"image/color"
	"math"
)

// Visual palette for the diagnostic sheets.
var (
	CWhite     = color.RGBA{0xff, 0xff, 0xff, 0xff}
	CPaper     = color.RGBA{0xf4, 0xf6, 0xfa, 0xff}
	CPanel     = color.RGBA{0xff, 0xff, 0xff, 0xff}
	CInk       = color.RGBA{0x1f, 0x29, 0x37, 0xff}
	CInkSoft   = color.RGBA{0x4b, 0x55, 0x63, 0xff}
	CMuted     = color.RGBA{0x8a, 0x97, 0xa8, 0xff}
	CBorder    = color.RGBA{0xdd, 0xe3, 0xec, 0xff}
	CBlue      = color.RGBA{0x25, 0x63, 0xeb, 0xff}
	CBlueSoft  = color.RGBA{0xdb, 0xea, 0xfe, 0xff}
	CRed       = color.RGBA{0xdc, 0x26, 0x26, 0xff}
	CRedSoft   = color.RGBA{0xfe, 0xe2, 0xe2, 0xff}
	CGreen     = color.RGBA{0x15, 0x80, 0x3d, 0xff}
	CGreenSoft = color.RGBA{0xdc, 0xfc, 0xe7, 0xff}
	CAmber     = color.RGBA{0xb4, 0x53, 0x09, 0xff}
	CAmberSoft = color.RGBA{0xfe, 0xf3, 0xc7, 0xff}
	CPurple    = color.RGBA{0x7c, 0x3a, 0xed, 0xff}
	CTeal      = color.RGBA{0x0e, 0x74, 0x90, 0xff}
)

func rgbaf(r, g, b uint8, a uint8) color.RGBA { return color.RGBA{r, g, b, a} }

// Panel draws a white card with a hairline border.
func (sc *Scene) Panel(x0, y0, x1, y1 float64, fill color.RGBA) {
	sc.FillRect(x0, y0, x1, y1, fill)
	sc.FrameRect(x0+0.5, y0+0.5, x1-0.5, y1-0.5, 1.4, CBorder)
}

// DashedLine strokes a dashed segment.
func (sc *Scene) DashedLine(x0, y0, x1, y1, dash, gap, width float64, c color.RGBA) {
	dx, dy := x1-x0, y1-y0
	total := math.Hypot(dx, dy)
	if total < 1e-6 {
		return
	}
	ux, uy := dx/total, dy/total
	for d := 0.0; d < total; d += dash + gap {
		e := math.Min(d+dash, total)
		sc.Line(x0+ux*d, y0+uy*d, x0+ux*e, y0+uy*e, width, c)
	}
}

// HGuide draws a horizontal guide spanning x0..x1.
func (sc *Scene) HGuide(y, x0, x1 float64, c color.RGBA) {
	sc.DashedLine(x0, y, x1, y, 7, 5, 1.3, c)
}

// VGuide draws a vertical guide.
func (sc *Scene) VGuide(x, y0, y1 float64, c color.RGBA) {
	sc.DashedLine(x, y0, x, y1, 7, 5, 1.3, c)
}

// Arrow draws a line with a triangular head at the (x1,y1) end.
func (sc *Scene) Arrow(x0, y0, x1, y1, width, head float64, c color.RGBA) {
	sc.Line(x0, y0, x1, y1, width, c)
	ang := math.Atan2(y1-y0, x1-x0)
	for _, s := range []float64{ang + 2.6, ang - 2.6} {
		sc.Line(x1, y1, x1+head*math.Cos(s), y1+head*math.Sin(s), width, c)
	}
}

// DimH draws a horizontal dimension with end ticks and a centered label.
func (sc *Scene) DimH(x0, x1, y float64, label string, fs *FontSet, c color.RGBA) {
	sc.Line(x0, y, x1, y, 1.6, c)
	sc.Line(x0, y-6, x0, y+6, 1.8, c)
	sc.Line(x1, y-6, x1, y+6, 1.8, c)
	if label != "" {
		st := TextStyle{Ppem: 17, Color: c}
		w := fs.Measure(label, st)
		sc.FillRect((x0+x1)/2-w/2-6, y-22, (x0+x1)/2+w/2+6, y-3, CWhite)
		sc.DrawText(label, (x0+x1)/2-w/2, y-8, fs, st)
	}
}

// DimV draws a vertical dimension with end ticks and a label to the right.
func (sc *Scene) DimV(y0, y1, x float64, label string, fs *FontSet, c color.RGBA) {
	sc.Line(x, y0, x, y1, 1.6, c)
	sc.Line(x-6, y0, x+6, y0, 1.8, c)
	sc.Line(x-6, y1, x+6, y1, 1.8, c)
	if label != "" {
		st := TextStyle{Ppem: 17, Color: c}
		sc.DrawText(label, x+10, (y0+y1)/2+6, fs, st)
	}
}

// Pill draws a rounded-ish label chip (square corners at this scale are fine).
func (sc *Scene) Pill(x, y float64, text string, fs *FontSet, fg, bg color.RGBA) float64 {
	st := TextStyle{Ppem: 16, Color: fg}
	w := fs.Measure(text, st)
	sc.FillRect(x, y-19, x+w+18, y+7, bg)
	sc.DrawText(text, x+9, y-4, fs, st)
	return x + w + 18
}

// Title draws the sheet title and subtitle.
func (sc *Scene) Title(title, sub string, fs *FontSet) {
	sc.DrawText(title, 48, 62, fs, TextStyle{Ppem: 34, Color: CInk})
	if sub != "" {
		sc.DrawText(sub, 50, 96, fs, TextStyle{Ppem: 17, Color: CMuted})
	}
}

// Diamond strokes a diamond marker with a filled center.
func (sc *Scene) Diamond(cx, cy, r float64, edge, center color.RGBA) {
	pts := []Pt{{cx, cy - r}, {cx + r, cy}, {cx, cy + r}, {cx - r, cy}}
	for i := 0; i < 4; i++ {
		a, b := pts[i], pts[(i+1)%4]
		sc.Line(a.X, a.Y, b.X, b.Y, 2.4, edge)
	}
	sc.Disc(cx, cy, r*0.4, center)
}

// LegendItem is one marker+text entry.
type LegendItem struct {
	Text string
	Draw func(sc *Scene, cx, cy float64)
}

// Legend draws items in a row and returns the ending x.
func (sc *Scene) Legend(x, y float64, fs *FontSet, items []LegendItem) float64 {
	st := TextStyle{Ppem: 16, Color: CInkSoft}
	cx := x
	for _, it := range items {
		if it.Draw != nil {
			it.Draw(sc, cx+10, y-5)
		}
		sc.DrawText(it.Text, cx+28, y, fs, st)
		cx += 28 + fs.Measure(it.Text, st) + 30
	}
	return cx
}
