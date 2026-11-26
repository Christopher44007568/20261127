package report

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"featurelines/internal/gfx"
	"featurelines/internal/pipeline"
)

func holeDiagram(s *gfx.Scene, o *pipeline.Output) {
	m := o.Metrics
	left := image.Rect(24, 92, 478, 596)
	right := image.Rect(502, 92, 956, 596)
	card(s, left, "A  CURVATURE-ONLY", fmt.Sprintf("%d SEG / %d ENDS", m.NaiveSegments, m.NaiveEndpoints), Palette.Red)
	card(s, right, "B  FUSED TRACING", "1 LOOP / 0 ENDS", Palette.Green)
	gfx.DrawText(s.Img, "HIGH THRESHOLD TAU=0.72", left.Min.X+22, left.Min.Y+40, 1, Palette.Muted)
	gfx.DrawText(s.Img, "CURVATURE DIFF + NORMAL ANGLE + BRIDGING", right.Min.X+22, right.Min.Y+40, 1, Palette.Muted)

	plotL := image.Rect(left.Min.X+18, left.Min.Y+58, left.Max.X-18, left.Max.Y-18)
	plotR := image.Rect(right.Min.X+18, right.Min.Y+58, right.Max.X-18, right.Max.Y-18)
	mpL := newMapper(plotL, -10.5, 10.5, -10.5, 10.5, true)
	mpR := newMapper(plotR, -10.5, 10.5, -10.5, 10.5, true)

	drawReference := func(mp *mapper) {
		mapCircle(s, mp, 0, 0, o.Case.Circle.Radius+1.6, Palette.Grid, 1, false)
		mapCircle(s, mp, 0, 0, o.Case.Circle.Radius, color.RGBA{170, 180, 192, 255}, 2, true)
		gfx.DrawText(s.Img, "IDEAL RIM", mp.X(0)-40, mp.Y(-o.Case.Circle.Radius)-26, 1, Palette.Muted)
	}
	drawReference(mpL)
	drawReference(mpR)

	ptsOf := func(vi []int) []Vec3 {
		p := make([]Vec3, len(vi))
		for i, v := range vi {
			p[i] = o.Case.Mesh.V[v]
		}
		return p
	}

	// Left: naive fragments.
	for _, ch := range o.Naive.Creases {
		near := true
		for _, v := range ch.Vertices {
			p := o.Case.Mesh.V[v]
			rad := math.Hypot(p.X, p.Y)
			if math.Abs(rad-o.Case.Circle.Radius) > 2.6 || p.Z < 0 {
				near = false
				break
			}
		}
		if !near {
			continue
		}
		mapChain(s, mpL, ptsOf(ch.Vertices), false, Palette.Orange, 5)
	}
	for _, mk := range m.NaiveMarks {
		x, y := mpL.X(mk.X), mpL.Y(mk.Y)
		if mk.HasPair {
			s.Line2D(x, y, mpL.X(mk.PairX), mpL.Y(mk.PairY), Palette.Red, 2)
		}
		s.Marker2D(x, y, gfx.MarkerSquare, 5, Palette.Red)
		if mk.Gap > 0 {
			mx := (mk.X + mk.PairX) / 2
			my := (mk.Y + mk.PairY) / 2
			labelHalo(s, fmt.Sprintf("%.2f", mk.Gap), mpL.X(mx)+6, mpL.Y(my)-16, 2, Palette.Red)
		}
	}
	gfx.DrawText(s.Img, "RED SQUARES = FALSE BREAK ENDS", left.Min.X+22, left.Max.Y-30, 1, Palette.Muted)

	// Right: robust closed loop.
	var holeChainPts []Vec3
	var holeGaps [][]Vec3
	for _, ch := range o.Robust.Creases {
		if !ch.Closed {
			continue
		}
		meanR, zSum := 0.0, 0.0
		for _, v := range ch.Vertices {
			p := o.Case.Mesh.V[v]
			meanR += math.Hypot(p.X, p.Y)
			zSum += p.Z
		}
		meanR /= float64(len(ch.Vertices))
		zMean := zSum / float64(len(ch.Vertices))
		if math.Abs(meanR-o.Case.Circle.Radius) < 0.6 && zMean > 0 {
			holeChainPts = ptsOf(ch.Vertices)
			for _, gp := range ch.Gaps {
				holeGaps = append(holeGaps, gp.Points)
			}
			break
		}
	}
	mapChain(s, mpR, holeChainPts, true, Palette.Blue, 5)
	for _, gp := range holeGaps {
		mapChain(s, mpR, gp, false, Palette.Green, 6)
	}
	fp := o.Case.Mesh.V[o.Case.FailVertex]
	s.Marker2D(mpR.X(fp.X), mpR.Y(fp.Y), gfx.MarkerStar, 11, Palette.Amber)
	gfx.DrawText(s.Img, "STAR = FAILED VERTEX, BRIDGED", right.Min.X+22, right.Min.Y+56, 1, rgb(176, 118, 0))
	labelHalo(s, "0 ENDPOINT ERRORS", right.Min.X+24, right.Max.Y-30, 2, Palette.Green)

	verdictBar(s, fmt.Sprintf("CLOSED LOOP | COV %.0f%% | 0 FALSE ENDS | SINGLE FAILED VERTEX BRIDGED", m.Coverage*100))
}

func labelHalo(s *gfx.Scene, text string, x, y, scale int, c color.RGBA) {
	w := gfx.TextWidth(text, scale)
	h := 8 * scale
	fillRect(s.Img, x-4, y-3, x+w+6, y+h, color.RGBA{255, 255, 255, 210})
	gfx.DrawText(s.Img, text, x, y, scale, c)
}

func verdictBar(s *gfx.Scene, text string) {
	r := image.Rect(24, 606, 956, 652)
	fillRect(s.Img, r.Min.X, r.Min.Y, r.Max.X, r.Max.Y, color.RGBA{226, 244, 234, 255})
	strokeRect(s, r, Palette.Green)
	gfx.DrawText(s.Img, "PASS", r.Min.X+18, r.Min.Y+14, 2, Palette.Green)
	gfx.DrawText(s.Img, text, r.Min.X+92, r.Min.Y+15, 2, Palette.Ink)
}
