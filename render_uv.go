package main

import (
	"fmt"
	"image/color"
	"math"
)

var islandPalette = []color.RGBA{
	rgba(56, 189, 248),
	rgba(167, 139, 250),
	rgba(52, 211, 153),
	rgba(251, 191, 36),
	rgba(244, 114, 182),
	rgba(96, 165, 250),
}

func RenderUVIslands(path string, m *Mesh, sig *Signals, atlas *Atlas, rep Report) error {
	const W, H = 1440, 1040
	c := NewCanvas(W, H)
	c.Clear(rgba(9, 14, 26))

	DrawText(c, 48, 34, 4, "UV ISLANDS", rgba(226, 232, 240))
	sub := fmt.Sprintf("LSCM  /  %d CHARTS  /  %d WEDGES  /  %d FLIPS",
		rep.IslandCount, rep.Wedges, rep.FlippedFaces)
	DrawText(c, 50, 82, 2, sub, rgba(148, 163, 184))

	chart := rectI{X: 70, Y: 140, W: 850, H: 850}
	drawUVChartBackground(c, chart)
	mapUV := func(uv Vec2) (int, int) {
		return chart.X + int(uv.X*float64(chart.W)+0.5),
			chart.Y + int((1-uv.Y)*float64(chart.H)+0.5)
	}

	faceIsland := MakeFaceIslandMap(atlas)
	for f := range m.Tri {
		island := faceIsland[f]
		base := islandPalette[island%len(islandPalette)]
		ws := atlas.FaceWedge[f]
		p0x, p0y := mapUV(atlas.WedgeUV[ws[0]])
		p1x, p1y := mapUV(atlas.WedgeUV[ws[1]])
		p2x, p2y := mapUV(atlas.WedgeUV[ws[2]])
		fillTriangle2D(c, float64(p0x), float64(p0y), float64(p1x), float64(p1y),
			float64(p2x), float64(p2y), base, 0.96)
	}
	for f := range m.Tri {
		ws := atlas.FaceWedge[f]
		x0, y0 := mapUV(atlas.WedgeUV[ws[0]])
		x1, y1 := mapUV(atlas.WedgeUV[ws[1]])
		x2, y2 := mapUV(atlas.WedgeUV[ws[2]])
		wire := rgba(7, 17, 31, 70)
		c.Line(x0, y0, x1, y1, wire)
		c.Line(x1, y1, x2, y2, wire)
		c.Line(x2, y2, x0, y0, wire)
	}
	tp := BuildTopology(m)
	drawUVSeams(c, tp, atlas, mapUV)
	c.Rect(chart.X, chart.Y, chart.W, chart.H, rgba(71, 85, 105))

	panelX := 980
	DrawText(c, panelX, 150, 3, "SEAM POLICY", rgba(226, 232, 240))
	lines := []string{
		fmt.Sprintf("SEAM EDGES     %d", rep.SeamEdges),
		fmt.Sprintf("SEAM LENGTH    %.2f", rep.SeamLength),
		fmt.Sprintf("FRONT CUTS     %d", rep.VisibleFrontSeamEdges),
		fmt.Sprintf("PROTECTED      %d", rep.ProtectedSeamEdges),
		fmt.Sprintf("FLIPPED ISLANDS %d", rep.FlippedFaces),
		"",
		"COST = CURVATURE + AO",
		"FRONT FACE = INFINITE",
		"",
		"STRETCH",
		fmt.Sprintf("AREA MED  %.2f", rep.AreaRatio.Median),
		fmt.Sprintf("L2 P95    %.2f", rep.L2Stretch.P95),
		fmt.Sprintf("LINF MAX  %.2f", rep.LInfStretch.Max),
	}
	y := 205
	for _, line := range lines {
		col := rgba(203, 213, 225)
		if line == "FRONT CUTS     0" || line == "FLIPPED ISLANDS 0" {
			col = rgba(74, 222, 128)
		}
		DrawText(c, panelX, y, 2, line, col)
		y += 34
	}

	for i, n := range rep.IslandTriangles {
		col := islandPalette[i%len(islandPalette)]
		c.FillRect(panelX, 690+i*48, 28, 28, col)
		DrawText(c, panelX+42, 696+i*48, 2,
			fmt.Sprintf("ISLAND %d  %d TRIS", i+1, n), rgba(226, 232, 240))
	}
	DrawText(c, 72, 1010, 2, "PACKED ATLAS - AMBER BORDERS ARE ACTIVE SEAM COPIES", rgba(251, 191, 36))
	return SavePNG(path, c.Img)
}

type rectI struct {
	X, Y, W, H int
}

func drawUVChartBackground(c *Canvas, r rectI) {
	c.FillRect(r.X, r.Y, r.W, r.H, rgba(15, 23, 42))
	grid := rgba(30, 41, 59)
	for i := 0; i <= 10; i++ {
		x := r.X + i*r.W/10
		y := r.Y + i*r.H/10
		c.Line(x, r.Y, x, r.Y+r.H, grid)
		c.Line(r.X, y, r.X+r.W, y, grid)
	}
}

func MakeFaceIslandMap(atlas *Atlas) []int {
	out := make([]int, len(atlas.FaceWedge))
	for id, faces := range atlas.Islands {
		for _, x := range faces {
			out[x] = id
		}
	}
	return out
}

func fillTriangle2D(c *Canvas, x0, y0, x1, y1, x2, y2 float64, col color.RGBA, alpha float64) {
	minX := int(math.Floor(math.Min(x0, math.Min(x1, x2))))
	maxX := int(math.Ceil(math.Max(x0, math.Max(x1, x2))))
	minY := int(math.Floor(math.Min(y0, math.Min(y1, y2))))
	maxY := int(math.Ceil(math.Max(y0, math.Max(y1, y2))))
	area := (x1-x0)*(y2-y0) - (y1-y0)*(x2-x0)
	if math.Abs(area) < 1e-8 {
		return
	}
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			px := float64(x) + 0.5
			py := float64(y) + 0.5
			w0 := ((x1-px)*(y2-py) - (y1-py)*(x2-px)) / area
			w1 := ((x2-px)*(y0-py) - (y2-py)*(x0-px)) / area
			w2 := 1 - w0 - w1
			if w0 >= -0.002 && w1 >= -0.002 && w2 >= -0.002 {
				c.Blend(x, y, col, alpha)
			}
		}
	}
}

func drawUVSeams(c *Canvas, tp *Topology, atlas *Atlas, mapUV func(Vec2) (int, int)) {
	seamCol := rgba(251, 191, 36)
	for eid, isSeam := range atlas.Seam {
		if !isSeam || eid >= len(tp.EdgeInc) {
			continue
		}
		for _, inc := range tp.EdgeInc[eid] {
			f := inc.Face
			for k := 0; k < 2; k++ {
				cornerA := inc.Corners[k]
				cornerB := inc.Corners[(k+1)%2]
				wa := atlas.FaceWedge[f][cornerA]
				wb := atlas.FaceWedge[f][cornerB]
				x0, y0 := mapUV(atlas.WedgeUV[wa])
				x1, y1 := mapUV(atlas.WedgeUV[wb])
				c.ThickLine(x0, y0, x1, y1, 3, seamCol)
			}
		}
	}
}
