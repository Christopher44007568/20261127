package main

import (
	"fmt"
	"image"
	"image/draw"
)

const (
	sheetW           = 1920
	sheetH           = 980
	panelW           = 600
	panelH           = 760
	panelY           = 150
	heatDisplayScale = 1.65
)

func RenderCheckerSheet(path string, m *Mesh, sig *Signals, atlas *Atlas, rep Report) error {
	sheet := NewCanvas(sheetW, sheetH)
	sheet.Clear(rgba(2, 6, 23))
	DrawText(sheet, 44, 38, 4, "CHECKERBOARD MODEL", rgba(226, 232, 240))
	DrawText(sheet, 48, 92, 2,
		fmt.Sprintf("PERSPECTIVE-CORRECT UV CHECK  /  %d ISLANDS  /  %d FLIPS  /  FRONT SEAMS %d",
			rep.IslandCount, rep.FlippedFaces, rep.VisibleFrontSeamEdges),
		rgba(148, 163, 184))

	cams := []Camera{
		{Target: V3(0, -0.10, 0), Yaw: 0, Pitch: 0.02, Dist: 5.55, FovY: 32},
		{Target: V3(0, -0.10, 0), Yaw: 0.72, Pitch: 0.12, Dist: 5.65, FovY: 32},
		{Target: V3(0, -0.10, 0), Yaw: 3.14159265, Pitch: 0.06, Dist: 5.55, FovY: 32},
	}
	labels := []string{"01 FRONT - ZERO FACE CUTS", "02 THREE-QUARTER", "03 BACK - LOW-COST SEAM"}
	xs := []int{40, 660, 1280}
	for i, cam := range cams {
		view := RenderModelView(panelW, panelH, m, atlas, sig, shadeChecker, cam)
		blitCanvas(sheet.Img, view, xs[i], panelY)
		sheet.Rect(xs[i], panelY, panelW, panelH, rgba(51, 65, 85))
		drawLabel(sheet, xs[i]+22, panelY+24, labels[i], 2)
	}
	DrawText(sheet, 48, 944, 2,
		"UNIFORM GLOBAL TEXEL SCALE; SQUARE CHECKS SHOULD STAY SQUARE. AMBER LINES ARE HIDDEN/BACK SEAMS.",
		rgba(251, 191, 36))
	return SavePNG(path, sheet.Img)
}

func RenderHeatmapSheet(path string, m *Mesh, sig *Signals, atlas *Atlas, rep Report) error {
	sheet := NewCanvas(sheetW, sheetH)
	sheet.Clear(rgba(2, 6, 23))
	DrawText(sheet, 44, 38, 4, "TEXTURE STRETCH HEATMAP", rgba(226, 232, 240))
	DrawText(sheet, 48, 92, 2,
		fmt.Sprintf("COMBINED AREA + ANGLE DISTORTION  /  L2 P95 %.2f  /  LINF MAX %.2f",
			rep.L2Stretch.P95, rep.LInfStretch.Max),
		rgba(148, 163, 184))

	cams := []Camera{
		{Target: V3(0, -0.10, 0), Yaw: 0, Pitch: 0.02, Dist: 5.55, FovY: 32},
		{Target: V3(0, -0.10, 0), Yaw: 2.55, Pitch: 0.08, Dist: 5.65, FovY: 32},
	}
	xs := []int{40, 660}
	labels := []string{"01 FRONT STRETCH", "02 REAR / OCCLUDED CUTS"}
	for i, cam := range cams {
		view := RenderModelView(panelW, panelH, m, atlas, sig, shadeHeat, cam)
		blitCanvas(sheet.Img, view, xs[i], panelY)
		sheet.Rect(xs[i], panelY, panelW, panelH, rgba(51, 65, 85))
		drawLabel(sheet, xs[i]+22, panelY+24, labels[i], 2)
	}
	uvPanel := RenderUVHeatView(panelW, panelH, m, atlas)
	blitCanvas(sheet.Img, uvPanel, 1280, panelY)
	sheet.Rect(1280, panelY, panelW, panelH, rgba(51, 65, 85))
	drawLabel(sheet, 1302, panelY+24, "03 UV SPACE DISTORTION", 2)

	barX, barY, barW, barH := 520, 925, 880, 24
	for x := 0; x < barW; x++ {
		col := heatRamp(float64(x) / float64(barW-1))
		sheet.FillRect(barX+x, barY, 1, barH, col)
	}
	sheet.Rect(barX, barY, barW, barH, rgba(226, 232, 240, 180))
	DrawText(sheet, barX, barY+34, 2, "0", rgba(203, 213, 225))
	DrawText(sheet, barX+barW/2-24, barY+34, 2, "0.8", rgba(203, 213, 225))
	DrawText(sheet, barX+barW-128, barY+34, 2, "1.65+", rgba(203, 213, 225))
	return SavePNG(path, sheet.Img)
}

func RenderSeamSheet(path string, m *Mesh, sig *Signals, atlas *Atlas, rep Report) error {
	sheet := NewCanvas(sheetW, 880)
	sheet.Clear(rgba(2, 6, 23))
	DrawText(sheet, 44, 36, 4, "CURVATURE + OCCLUSION SEAMS", rgba(226, 232, 240))
	DrawText(sheet, 48, 88, 2,
		fmt.Sprintf("AMBER CUTS ROUTE TO BACK / UNDER-JAW  /  FRONT EDGES %d  /  MEAN AO %.3f",
			rep.VisibleFrontSeamEdges, rep.MeanAO),
		rgba(148, 163, 184))
	cams := []Camera{
		{Target: V3(0, -0.10, 0), Yaw: 0, Pitch: 0.02, Dist: 5.55, FovY: 32},
		{Target: V3(0, -0.10, 0), Yaw: 1.45, Pitch: 0.08, Dist: 5.65, FovY: 32},
		{Target: V3(0, -0.10, 0), Yaw: 3.14, Pitch: 0.05, Dist: 5.55, FovY: 32},
	}
	labels := []string{"FRONT", "SIDE / EAR", "BACK"}
	for i, cam := range cams {
		view := RenderModelView(panelW, 650, m, atlas, sig, shadeAO, cam)
		blitCanvas(sheet.Img, view, xs3(i), 140)
		sheet.Rect(xs3(i), 140, panelW, 650, rgba(51, 65, 85))
		drawLabel(sheet, xs3(i)+22, 164, labels[i], 2)
	}
	DrawText(sheet, 48, 828, 2, "DARK CREASES ARE RAY-TRACED SELF OCCLUSION; HIGH-CURVATURE EDGES LOWER SEAM COST.", rgba(148, 163, 184))
	return SavePNG(path, sheet.Img)
}

func xs3(i int) int {
	return []int{40, 660, 1280}[i]
}

func blitCanvas(dst *image.RGBA, src *Canvas, x, y int) {
	r := image.Rect(x, y, x+src.W, y+src.H)
	draw.Draw(dst, r, src.Img, image.Pt(0, 0), draw.Src)
}

func drawLabel(c *Canvas, x, y int, text string, scale int) {
	DrawText(c, x+2, y+2, scale, text, rgba(2, 6, 23))
	DrawText(c, x, y, scale, text, rgba(248, 250, 252))
}

func RenderUVHeatView(w, h int, m *Mesh, atlas *Atlas) *Canvas {
	c := NewCanvas(w, h)
	c.Clear(rgba(15, 23, 42))
	chart := rectI{X: 28, Y: 100, W: w - 56, H: w - 56}
	if chart.H > h-150 {
		chart.H = h - 150
		chart.W = chart.H
		chart.X = (w - chart.W) / 2
	}
	drawUVChartBackground(c, chart)
	mapUV := func(uv Vec2) (int, int) {
		return chart.X + int(uv.X*float64(chart.W)+0.5),
			chart.Y + int((1-uv.Y)*float64(chart.H)+0.5)
	}
	for f := range m.Tri {
		heat := 0.0
		if f < len(atlas.Metrics) {
			heat = atlas.Metrics[f].Heat / heatDisplayScale
		}
		col := heatRamp(clampFloat(heat, 0, 1))
		ws := atlas.FaceWedge[f]
		x0, y0 := mapUV(atlas.WedgeUV[ws[0]])
		x1, y1 := mapUV(atlas.WedgeUV[ws[1]])
		x2, y2 := mapUV(atlas.WedgeUV[ws[2]])
		fillTriangle2D(c, float64(x0), float64(y0), float64(x1), float64(y1),
			float64(x2), float64(y2), col, 1)
	}
	for f := range m.Tri {
		ws := atlas.FaceWedge[f]
		x0, y0 := mapUV(atlas.WedgeUV[ws[0]])
		x1, y1 := mapUV(atlas.WedgeUV[ws[1]])
		x2, y2 := mapUV(atlas.WedgeUV[ws[2]])
		c.Line(x0, y0, x1, y1, rgba(15, 23, 42, 90))
		c.Line(x1, y1, x2, y2, rgba(15, 23, 42, 90))
		c.Line(x2, y2, x0, y0, rgba(15, 23, 42, 90))
	}
	c.Rect(chart.X, chart.Y, chart.W, chart.H, rgba(71, 85, 105))
	return c
}
