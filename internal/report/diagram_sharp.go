package report

import (
	"fmt"
	"image"

	"featurelines/internal/eval"
	"featurelines/internal/gfx"
	"featurelines/internal/pipeline"
)

func sharpDiagram(s *gfx.Scene, o *pipeline.Output) {
	m := o.Metrics
	a, b := o.Case.Segment.A, o.Case.Segment.B

	left := image.Rect(24, 92, 478, 596)
	right := image.Rect(502, 92, 956, 596)
	card(s, left, "A  CURVATURE-ONLY", "BROKEN", Palette.Red)
	card(s, right, "B  FUSED TRACING", "2 TRUE ENDS", Palette.Green)
	gfx.DrawText(s.Img, "HIGH THRESHOLD SPLITS THE CREASE", left.Min.X+22, left.Min.Y+40, 1, Palette.Muted)
	gfx.DrawText(s.Img, "ONE CONTINUOUS OPEN LINE", right.Min.X+22, right.Min.Y+40, 1, Palette.Muted)

	plotL := image.Rect(left.Min.X+30, left.Min.Y+70, left.Max.X-30, left.Max.Y-70)
	plotR := image.Rect(right.Min.X+30, right.Min.Y+70, right.Max.X-30, right.Max.Y-70)
	// Horizontal axis is world Y; vertical axis is world Z (amplified layout).
	mpL := newMapper(plotL, -17, 17, -3.2, 4.5, false)
	mpR := newMapper(plotR, -17, 17, -3.2, 4.5, false)

	drawTruth := func(mp *mapper, withLabels bool) {
		aa := Vec3{X: a.Y, Y: a.Z, Z: 0}
		bb := Vec3{X: b.Y, Y: b.Z, Z: 0}
		mapDashed(s, mp, aa, bb, rgb(170, 180, 192), 2, 7, 5)
		for _, t := range []Vec3{aa, bb} {
			s.Marker2D(mp.X(t.X), mp.Y(t.Y), gfx.MarkerDiamond, 8, Palette.Green)
		}
		if withLabels {
			gfx.DrawText(s.Img, "TRUE END A", mp.X(aa.X)-34, mp.Y(aa.Y)+20, 1, Palette.Muted)
			gfx.DrawText(s.Img, "TRUE END B", mp.X(bb.X)-34, mp.Y(bb.Y)+20, 1, Palette.Muted)
		}
	}
	drawTruth(mpL, true)
	drawTruth(mpR, false)

	yz := func(p Vec3) Vec3 { return Vec3{X: p.Y, Y: p.Z, Z: 0} }

	// Left: naive fragments and their false middle endpoints.
	var falseEnds []Vec3
	for _, ch := range o.Naive.Creases {
		mean := 0.0
		for _, v := range ch.Vertices {
			mean += eval.PointSegDist(o.Case.Mesh.V[v], a, b)
		}
		mean /= float64(len(ch.Vertices))
		if mean > 2.0 {
			continue
		}
		pts := make([]Vec3, len(ch.Vertices))
		for i, v := range ch.Vertices {
			pts[i] = yz(o.Case.Mesh.V[v])
		}
		mapChain(s, mpL, pts, false, Palette.Orange, 5)
		for _, vi := range []int{ch.Vertices[0], ch.Vertices[len(ch.Vertices)-1]} {
			p := o.Case.Mesh.V[vi]
			if absF(p.Y) < 13.5 {
				falseEnds = append(falseEnds, yz(p))
				s.Marker2D(mpL.X(p.Y), mpL.Y(p.Z), gfx.MarkerCross, 9, Palette.Red)
			}
		}
	}
	if len(falseEnds) == 2 {
		mapDashed(s, mpL, falseEnds[0], falseEnds[1], Palette.Red, 2, 4, 4)
		mid := falseEnds[0].Add(falseEnds[1]).Scale(0.5)
		gap := falseEnds[0].Dist(falseEnds[1])
		labelHalo(s, fmt.Sprintf("%.2f MM BREAK", gap), mpL.X(mid.X)-70, mpL.Y(mid.Y)-30, 2, Palette.Red)
	}
	gfx.DrawText(s.Img, "RED X = SPURIOUS ENDPOINT ERRORS", left.Min.X+22, left.Max.Y-30, 1, Palette.Muted)

	// Right: robust chain.
	var bestPts []Vec3
	bestMean := 1e9
	var bestGaps [][]Vec3
	for _, ch := range o.Robust.Creases {
		mean := 0.0
		for _, v := range ch.Vertices {
			mean += eval.PointSegDist(o.Case.Mesh.V[v], a, b)
		}
		mean /= float64(len(ch.Vertices))
		if mean < bestMean {
			bestMean = mean
			bestPts = make([]Vec3, len(ch.Vertices))
			for i, v := range ch.Vertices {
				bestPts[i] = yz(o.Case.Mesh.V[v])
			}
			bestGaps = nil
			for _, gp := range ch.Gaps {
				gp2 := make([]Vec3, len(gp.Points))
				for i, p := range gp.Points {
					gp2[i] = yz(p)
				}
				bestGaps = append(bestGaps, gp2)
			}
		}
	}
	mapChain(s, mpR, bestPts, false, Palette.Blue, 5)
	for _, gp := range bestGaps {
		mapChain(s, mpR, gp, false, Palette.Green, 7)
	}
	fp := yz(o.Case.Mesh.V[o.Case.FailVertex])
	s.Marker2D(mpR.X(fp.X), mpR.Y(fp.Y), gfx.MarkerStar, 11, Palette.Amber)
	for _, mk := range m.Endpoints {
		s.Marker2D(mpR.X(mk.Y), mpR.Y(mk.Z), gfx.MarkerRing, 9, Palette.Green)
		labelHalo(s, fmt.Sprintf("E=%.2f", mk.Error), mpR.X(mk.Y)-26, mpR.Y(mk.Z)+20, 2, Palette.Green)
	}
	gfx.DrawText(s.Img, "FAILED MID-VERTEX, TRACE HOLDS", right.Min.X+24, right.Max.Y-30, 2, Palette.Green)

	verdictBar(s, fmt.Sprintf("ONE OPEN CREASE | 2 ENDS | MAX ERR %.2f MM | COV %.0f%%",
		m.MaxEndpointError, m.Coverage*100))
}

func absF(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
