package main

import "math"

type SectionMapper struct {
	Grid  *VoxelGrid
	OX    int
	OY    int
	Scale int
}

func (m SectionMapper) XY(p Vec3) (int, int) {
	x := m.OX + int((p.X-m.Grid.Min.X)/m.Grid.Voxel*float64(m.Scale)+0.5)
	maxY := m.Grid.Min.Y + float64(m.Grid.NY)*m.Grid.Voxel
	y := m.OY + int((maxY-p.Y)/m.Grid.Voxel*float64(m.Scale)+0.5)
	return x, y
}

func (m SectionMapper) CellRect(ix, iy int) (int, int, int, int) {
	x := m.OX + ix*m.Scale
	maxY := m.Grid.Min.Y + float64(m.Grid.NY)*m.Grid.Voxel
	yWorld := m.Grid.Min.Y + float64(iy+1)*m.Grid.Voxel
	y := m.OY + int((maxY-yWorld)/m.Grid.Voxel*float64(m.Scale)+0.5)
	return x, y, m.Scale, m.Scale
}

func drawGeometryOutline(c *Canvas, m SectionMapper, geom Geometry, z float64, col RGB, thick int) {
	for _, b := range geom {
		if z < b.Center.Z-b.Half.Z || z > b.Center.Z+b.Half.Z {
			continue
		}
		p00 := V3(b.Center.X-b.Half.X, b.Center.Y-b.Half.Y, z)
		p10 := V3(b.Center.X+b.Half.X, b.Center.Y-b.Half.Y, z)
		p11 := V3(b.Center.X+b.Half.X, b.Center.Y+b.Half.Y, z)
		p01 := V3(b.Center.X-b.Half.X, b.Center.Y+b.Half.Y, z)
		pts := []Vec3{p00, p10, p11, p01, p00}
		for i := 0; i < 4; i++ {
			x0, y0 := m.XY(pts[i])
			x1, y1 := m.XY(pts[i+1])
			c.Line(x0, y0, x1, y1, thick, col)
		}
	}
}

func drawContour(c *Canvas, m SectionMapper, segments []Segment3, halo, line RGB) {
	for _, s := range segments {
		x0, y0 := m.XY(s.A)
		x1, y1 := m.XY(s.B)
		c.Line(x0, y0, x1, y1, 4, halo)
	}
	for _, s := range segments {
		x0, y0 := m.XY(s.A)
		x1, y1 := m.XY(s.B)
		c.Line(x0, y0, x1, y1, 2, line)
	}
}

func WriteVoxelSectionPNG(path string, scene Scene, grid *VoxelGrid, iz int, segments []Segment3, maxWeight float32) error {
	scale := 3
	panelW := grid.NX * scale
	panelH := grid.NY * scale
	pad := 18
	header := 42
	width := pad + 3*panelW + 2*pad + pad
	height := header + panelH + 70
	cv := NewCanvas(width, height, RGB{0.94, 0.95, 0.97})
	cv.DrawText(pad, 12, scene.ID+" / voxel section z="+formatFloat(scene.SliceZ, 2)+"m / h="+formatFloat(grid.Voxel, 3), 2, RGB{0.05, 0.07, 0.10})

	titles := []string{"TRUNCATED SDF", "CUMULATIVE WEIGHT", "THRESHOLD BAND + ZERO SURFACE"}
	for panel := 0; panel < 3; panel++ {
		ox := pad + panel*(panelW+pad)
		oy := header + 18
		cv.FillRect(ox, oy, panelW, panelH, RGB{0.07, 0.08, 0.11})
		m := SectionMapper{Grid: grid, OX: ox, OY: oy, Scale: scale}

		for iy := 0; iy < grid.NY; iy++ {
			for ix := 0; ix < grid.NX; ix++ {
				id := grid.Index(ix, iy, iz)
				if grid.Weight[id] < 0.05 {
					if (ix+iy)%17 == 0 {
						x, y, ww, hh := m.CellRect(ix, iy)
						cv.FillRect(x, y, ww, hh, RGB{0.11, 0.12, 0.15})
					}
					continue
				}
				var col RGB
				switch panel {
				case 0:
					col = TSDFColor(float64(grid.Value[id]))
				case 1:
					wn := clampFloat(float64(grid.Weight[id]/maxWeight), 0, 1)
					col = Viridis(0.08 + 0.9*sqrtFloat(wn))
				default:
					col = RGB{0.10, 0.16, 0.28}
					if abs32(grid.Value[id]) <= 0.35 {
						col = RGB{1.0, 0.82, 0.12}
					}
				}
				x, y, ww, hh := m.CellRect(ix, iy)
				cv.FillRect(x, y, ww, hh, col)
			}
		}

		drawGeometryOutline(cv, m, scene.Geometry, scene.SliceZ, RGB{0.95, 0.95, 0.95}, 2)
		drawContour(cv, m, segments, RGB{0.02, 0.05, 0.08}, RGB{0.10, 0.95, 0.95})
		cv.FrameRect(ox, oy, panelW, panelH, 2, RGB{0.15, 0.17, 0.22})
		cv.DrawText(ox, oy-15, titles[panel], 1, RGB{0.05, 0.07, 0.10})

		barW := 180
		barX := ox
		barY := oy + panelH + 12
		for x := 0; x < barW; x++ {
			var col RGB
			if panel == 1 {
				col = Viridis(float64(x) / float64(barW))
			} else {
				v := -1 + 2*float64(x)/float64(barW)
				col = TSDFColor(v)
			}
			cv.FillRect(barX+x, barY, 1, 10, col)
		}
		cv.FrameRect(barX, barY, barW, 10, 1, RGB{0.1, 0.1, 0.1})
		if panel == 2 {
			cv.FillRect(barX+barW+18, barY, 12, 10, RGB{1, 0.82, 0.12})
			cv.DrawText(barX+barW+34, barY-1, "ABS(TSDF)<=0.35", 1, RGB{0.05, 0.05, 0.05})
		}
	}

	return SaveCanvasPNG(path, cv)
}

func abs32(v float32) float32 {
	if v < 0 {
		return -v
	}
	return v
}

func sqrtFloat(v float64) float64 {
	if v <= 0 {
		return 0
	}
	return math.Sqrt(v)
}
