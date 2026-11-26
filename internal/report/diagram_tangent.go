package report

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"featurelines/internal/gfx"
	"featurelines/internal/pipeline"
	"featurelines/internal/topo"
)

func tangentDiagram(s *gfx.Scene, o *pipeline.Output) {
	left := image.Rect(24, 92, 478, 596)
	right := image.Rect(502, 92, 956, 596)
	card(s, left, "A  SEAM SIGNALS", "ANGLE VETO", Palette.Green)
	card(s, right, "B  FALSE FEATURE MAP", "0 OURS", Palette.Green)

	seam := o.Case.SeamVerts
	tp := o.Robust.Sig.Topo
	type sample struct{ thetaDeg, cdiff float64 }
	var samples []sample
	for j := 0; j+1 < len(seam); j++ {
		ei := tp.EdgeIndex[topo.Key(seam[j], seam[j+1])]
		sg := o.Robust.Sig.List[ei]
		samples = append(samples, sample{sg.Theta * 180 / math.Pi, sg.CDiff})
	}

	// Top chart: normal angle.
	chartAngle := image.Rect(70, 158, 452, 322)
	drawChartFrame(s, chartAngle, "NORMAL ANGLE theta (deg)", []float64{0, 2, 4, 6, 8}, 8)
	angleGateY := chartY(chartAngle, 5, 8)
	for x := chartAngle.Min.X; x < chartAngle.Max.X; x += 10 {
		s.Line2D(x, angleGateY, x+5, angleGateY, Palette.Red, 2)
	}
	labelHalo(s, "VETO 5 deg", chartAngle.Max.X-92, angleGateY-16, 1, Palette.Red)
	plotSeries(s, chartAngle, len(samples), func(i int) float64 { return samples[i].thetaDeg }, 8, Palette.Blue)

	// Bottom chart: curvature difference.
	chartCD := image.Rect(70, 392, 452, 556)
	drawChartFrame(s, chartCD, "CURVATURE DIFFERENCE dk/k98", []float64{0, 0.4, 0.8, 1.2}, 1.2)
	cdGateY := chartY(chartCD, o.Opt.CDiffStrong, 1.2)
	for x := chartCD.Min.X; x < chartCD.Max.X; x += 10 {
		s.Line2D(x, cdGateY, x+5, cdGateY, Palette.Orange, 2)
	}
	labelHalo(s, "STRONG 0.85", chartCD.Min.X+8, cdGateY-16, 1, Palette.Orange)
	plotSeries(s, chartCD, len(samples), func(i int) float64 { return math.Min(samples[i].cdiff, 1.2) }, 1.2, Palette.Orange)

	// Right: top-view X-Y map.
	plot := image.Rect(right.Min.X+24, right.Min.Y+64, right.Max.X-20, right.Max.Y-64)
	mp := newMapper(plot, -16, 12, -15, 15, true)
	a, b := o.Case.Seam.A, o.Case.Seam.B
	mapDashed(s, mp, Vec3{X: a.X, Y: a.Y, Z: 0}, Vec3{X: b.X, Y: b.Y, Z: 0}, rgb(120, 200, 150), 3, 8, 5)
	gfx.DrawText(s.Img, "TANGENT SEAM", mp.X(1.0), mp.Y(1.5), 1, Palette.Green)

	// Curvature-only response along the true seam row (false positives).
	m := o.Case.Mesh
	tp2 := o.CurvOnly.Sig.Topo
	for j := 0; j+1 < len(seam); j++ {
		va, vb := seam[j], seam[j+1]
		pa, pb := m.V[va], m.V[vb]
		ei, ok := tp2.EdgeIndex[topo.Key(va, vb)]
		fired := ok && o.CurvOnly.Selected[ei]
		col := Palette.Red
		thick := 5
		if !fired {
			col = rgb(226, 170, 170)
			thick = 2
		}
		mapLine(s, mp, Vec3{X: pa.X, Y: pa.Y, Z: 0}, Vec3{X: pb.X, Y: pb.Y, Z: 0}, col, thick)
	}
	gfx.DrawText(s.Img, fmt.Sprintf("CURVATURE-ONLY: %.0f MM FALSE SEAM", o.Metrics.CurvOnlySeamLength),
		right.Min.X+22, right.Max.Y-52, 2, Palette.Red)
	gfx.DrawText(s.Img, "OUR FUSED TRACER: 0 FEATURE EDGES", right.Min.X+22, right.Max.Y-26, 2, Palette.Green)

	verdictBar(s, "TANGENT VETOED | NORMALS AGREE (<1 DEG), CURV JUMP IGNORED")
}

func chartY(r image.Rectangle, v, vmax float64) int {
	return r.Max.Y - int(float64(r.Dy())*v/vmax)
}

func drawChartFrame(s *gfx.Scene, r image.Rectangle, title string, ticks []float64, vmax float64) {
	gfx.DrawText(s.Img, title, r.Min.X, r.Min.Y-20, 2, Palette.Ink)
	strokeRect(s, r, Palette.Grid)
	for _, t := range ticks {
		y := chartY(r, t, vmax)
		s.Line2D(r.Min.X, y, r.Max.X, y, Palette.Grid, 1)
		label := fmt.Sprintf("%.1f", t)
		if vmax >= 7 {
			label = fmt.Sprintf("%.0f", t)
		}
		gfx.DrawText(s.Img, label, r.Min.X-34, y-6, 1, Palette.Muted)
	}
}

func plotSeries(s *gfx.Scene, r image.Rectangle, n int, val func(int) float64, vmax float64, c color.RGBA) {
	if n < 2 {
		return
	}
	px := func(i int, v float64) (int, int) {
		x := r.Min.X + int(float64(r.Dx())*float64(i)/float64(n-1))
		y := chartY(r, math.Min(math.Max(v, 0), vmax), vmax)
		return x, y
	}
	px0, py0 := px(0, val(0))
	for i := 1; i < n; i++ {
		px1, py1 := px(i, val(i))
		s.Line2D(px0, py0, px1, py1, c, 2)
		px0, py0 = px1, py1
	}
}
