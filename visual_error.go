package main

import (
	"math"
)

var viewColors = []RGB{
	{0.95, 0.25, 0.25},
	{0.10, 0.85, 0.95},
	{0.95, 0.75, 0.10},
	{0.55, 0.95, 0.35},
	{0.85, 0.45, 0.95},
	{1.00, 0.55, 0.15},
	{0.35, 0.55, 1.00},
	{0.95, 0.95, 0.95},
}

func WriteErrorHeatmapPNG(path string, scene Scene, grid *VoxelGrid, iz int, segments []Segment3, stats ErrorStats) error {
	scale := 5
	panelW := grid.NX * scale
	panelH := grid.NY * scale
	pad := 20
	sideW := 150
	width := pad + panelW + 30 + sideW + pad
	height := 70 + panelH + 50
	cv := NewCanvas(width, height, RGB{0.96, 0.97, 0.98})
	cv.DrawText(pad, 14, scene.ID+" / cross-section error heatmap", 2, RGB{0.04, 0.06, 0.09})
	cv.DrawText(pad, 38, "COLOR = ABS(FUSED METRIC TSDF - ANALYTIC SDF), BAND +-100 MM", 1, RGB{0.20, 0.24, 0.30})

	ox, oy := pad, 64
	cv.FillRect(ox, oy, panelW, panelH, RGB{0.99, 0.99, 0.98})
	m := SectionMapper{Grid: grid, OX: ox, OY: oy, Scale: scale}

	for _, b := range scene.Geometry {
		if scene.SliceZ < b.Center.Z-b.Half.Z || scene.SliceZ > b.Center.Z+b.Half.Z {
			continue
		}
		topLeftX, topY := m.XY(V3(b.Center.X-b.Half.X, b.Center.Y+b.Half.Y, scene.SliceZ))
		_, bottomY := m.XY(V3(b.Center.X-b.Half.X, b.Center.Y-b.Half.Y, scene.SliceZ))
		rightX, _ := m.XY(V3(b.Center.X+b.Half.X, b.Center.Y+b.Half.Y, scene.SliceZ))
		cv.FillRect(topLeftX, topY, rightX-topLeftX, bottomY-topY, RGB{0.88, 0.91, 0.96})
	}

	const errorScale = 0.03
	for iy := 0; iy < grid.NY; iy++ {
		for ix := 0; ix < grid.NX; ix++ {
			id := grid.Index(ix, iy, iz)
			if grid.Weight[id] < 0.2 {
				continue
			}
			p := grid.Center(ix, iy, iz)
			gt := scene.Geometry.SDF(p)
			if math.Abs(gt) > 0.10 {
				continue
			}
			err := math.Abs(grid.MetricSDF(id) - gt)
			col := HeatColor(err / errorScale)
			x, y, ww, hh := m.CellRect(ix, iy)
			cv.FillRect(x, y, ww, hh, col)
		}
	}

	drawGeometryOutline(cv, m, scene.Geometry, scene.SliceZ, RGB{0.05, 0.05, 0.05}, 3)
	drawContour(cv, m, segments, RGB{1, 1, 1}, RGB{0.00, 0.30, 0.95})
	cv.FrameRect(ox, oy, panelW, panelH, 2, RGB{0.12, 0.14, 0.18})

	barX := ox + panelW + 42
	barY := oy + 10
	barH := 300
	barW := 22
	for y := 0; y < barH; y++ {
		v := 1 - float64(y)/float64(barH)
		cv.FillRect(barX, barY+y, barW, 1, HeatColor(v))
	}
	cv.FrameRect(barX, barY, barW, barH, 1, RGB{0.05, 0.05, 0.05})
	for _, mm := range []float64{0, 10, 20, 30} {
		y := barY + barH - int(mm/30*float64(barH))
		cv.Line(barX+barW, y, barX+barW+6, y, 1, RGB{0.05, 0.05, 0.05})
		cv.DrawText(barX+barW+10, y-3, formatFloat(mm, 0), 1, RGB{0.05, 0.05, 0.05})
	}
	cv.DrawText(barX, barY-14, "MM", 1, RGB{0.05, 0.05, 0.05})

	tx := barX
	ty := barY + barH + 34
	lines := []string{
		"CONTOUR N=" + stringInt(stats.Count),
		"MEAN " + formatFloat(stats.Mean*1000, 1) + " MM",
		"RMSE " + formatFloat(stats.RMSE*1000, 1) + " MM",
		"P95  " + formatFloat(stats.P95*1000, 1) + " MM",
		"MAX  " + formatFloat(stats.Max*1000, 1) + " MM",
	}
	for i, line := range lines {
		cv.DrawText(tx, ty+i*18, line, 1, RGB{0.05, 0.07, 0.10})
	}
	cv.DrawText(ox, oy+panelH+14, "BLUE=FUSED ZERO CONTOUR   BLACK=ANALYTIC GROUND TRUTH", 1, RGB{0.08, 0.10, 0.14})
	return SaveCanvasPNG(path, cv)
}

func WriteRawCloudSectionPNG(path string, scene Scene, grid *VoxelGrid, frames []*DepthFrame, rawStats ErrorStats) error {
	scale := 5
	panelW := grid.NX * scale
	panelH := grid.NY * scale
	pad := 20
	width := pad + panelW + pad
	height := 70 + panelH + 54
	cv := NewCanvas(width, height, RGB{0.96, 0.97, 0.98})
	cv.DrawText(pad, 14, scene.ID+" / direct unprojected cloud slice", 2, RGB{0.04, 0.06, 0.09})
	cv.DrawText(pad, 38, "EACH VIEW USES ITS PERTURBED FUSION POSE; COLORED BANDS ARE STACKED LAYERS", 1, RGB{0.20, 0.24, 0.30})

	ox, oy := pad, 64
	cv.FillRect(ox, oy, panelW, panelH, RGB{0.05, 0.06, 0.09})
	m := SectionMapper{Grid: grid, OX: ox, OY: oy, Scale: scale}
	for fi, f := range frames {
		col := viewColors[fi%len(viewColors)]
		for y := 0; y < f.Height; y++ {
			for x := 0; x < f.Width; x++ {
				d := f.Depth[y*f.Width+x]
				if d <= 0 {
					continue
				}
				p := f.Fusion.UnprojectDepth(x, y, d)
				if math.Abs(p.Z-scene.SliceZ) > 0.045 {
					continue
				}
				px, py := m.XY(p)
				cv.FillRect(px-1, py-1, 2, 2, col)
			}
		}
	}
	drawGeometryOutline(cv, m, scene.Geometry, scene.SliceZ, RGB{1, 1, 1}, 2)
	cv.FrameRect(ox, oy, panelW, panelH, 2, RGB{0.12, 0.14, 0.18})

	for i, f := range frames {
		col := viewColors[i%len(viewColors)]
		lx := ox + 10 + (i%4)*150
		ly := oy + panelH + 12 + (i/4)*18
		cv.FillRect(lx, ly, 10, 10, col)
		cv.DrawText(lx+16, ly, f.Label+" AZ "+formatFloat(f.True.Azimuth, 0), 1, RGB{0.08, 0.09, 0.12})
	}
	info := "RAW CLOUD RMSE " + formatFloat(rawStats.RMSE*1000, 1) + " MM, P95 " + formatFloat(rawStats.P95*1000, 1) + " MM"
	cv.DrawText(ox+panelW-TextWidth(info, 1), oy+panelH+13, info, 1, RGB{0.35, 0.05, 0.05})
	return SaveCanvasPNG(path, cv)
}
