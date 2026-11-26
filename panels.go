package main

import (
	"fmt"
	"image"
	"image/draw"
)

const panelW = 800
const panelH = 600

func RenderInputPanel(path string, raw *Cloud, removed map[int]bool, cam Camera) error {
	c := NewCanvas(panelW, panelH, RGB{13, 19, 27})
	zMin, zMax := 1e9, -1e9
	for i, p := range raw.P {
		if !removed[i] {
			zMin = minFloat(zMin, p.Z)
			zMax = maxFloat(zMax, p.Z)
		}
	}
	if zMax-zMin < 1e-9 {
		zMin -= 1
		zMax += 1
	}
	for i, p := range raw.P {
		if removed[i] {
			continue
		}
		t := (p.Z - zMin) / (zMax - zMin)
		c.DrawPoint3D(cam, p, 1, ElevationColor(t))
	}
	for i, p := range raw.P {
		if removed[i] {
			pr := cam.Project(p)
			x, y := int(pr.X), int(pr.Y)
			for oy := -2; oy <= 2; oy++ {
				for ox := -2; ox <= 2; ox++ {
					if ox*ox+oy*oy <= 5 {
						c.SetPixel(x+ox, y+oy, RGB{255, 75, 92})
					}
				}
			}
		}
	}
	removedCount := len(removed)
	c.DrawTitleBar("1 INPUT POINT CLOUD", fmt.Sprintf("height color; %d red sparse outliers rejected", removedCount))
	c.FillRect(14, panelH-38, 330, 24, RGB{20, 29, 40})
	c.DrawText(24, panelH-30, "COARSE NOISE + MISSING HOLES", 1, RGB{210, 222, 232})
	return c.SavePNG(path)
}

func RenderNormalPanel(path string, points, normals []V3, cam Camera, title, subtitle string) error {
	c := NewCanvas(panelW, panelH, RGB{15, 18, 28})
	colors := make([]RGB, len(points))
	for i := range points {
		n := V3{0, 0, 1}
		if i < len(normals) {
			n = normals[i]
		}
		colors[i] = NormalColor(n)
	}
	c.DrawCloud(cam, points, colors, 1)
	c.DrawTitleBar(title, subtitle)
	c.FillRect(14, panelH-38, 360, 24, RGB{24, 28, 42})
	c.DrawText(24, panelH-30, "RGB = 0.5 + 0.5 * NORMAL XYZ", 1, RGB{226, 232, 240})
	return c.SavePNG(path)
}

func RenderMeshPanel(path string, m *Mesh, cam Camera) error {
	c := NewCanvas(panelW, panelH, RGB{238, 241, 244})
	c.DrawMesh(cam, m, false, RGB{184, 201, 214}, V3{-0.38, -0.46, 0.80})
	c.DrawBoundaryEdges(cam, m, RGB{0, 157, 196})
	c.DrawTitleBar("3 IMPLICIT MESH", fmt.Sprintf("weighted quadratic field; %d triangles; %d open edges", len(m.Tri), m.BoundaryEdgeCount()))
	c.FillRect(14, panelH-38, 286, 24, RGB{255, 255, 255})
	c.DrawText(24, panelH-30, "CYAN = HOLE / SHELL BOUNDARIES", 1, RGB{0, 104, 132})
	return c.SavePNG(path)
}

func RenderCurvaturePanel(path string, m *Mesh, cam Camera, errScale float64, stats ErrorStats) error {
	c := NewCanvas(panelW, panelH, RGB{14, 18, 24})
	c.DrawMesh(cam, m, true, RGB{200, 200, 200}, V3{-0.35, -0.45, 0.82})
	c.DrawBoundaryEdges(cam, m, RGB{116, 235, 255})
	c.DrawTitleBar("4 CURVATURE ERROR", fmt.Sprintf("interior |H-Htrue|  RMSE %.4f  P95 %.4f", stats.RMSE, stats.P95))

	barX, barY, barW, barH := 24, panelH-58, 300, 14
	for x := 0; x < barW; x++ {
		col := HeatColor(float64(x) / float64(barW-1))
		c.FillRect(barX+x, barY, 1, barH, col)
	}
	c.DrawText(barX, barY+barH+6, "0", 1, RGB{226, 232, 240})
	label := fmt.Sprintf("%.4f", errScale)
	c.DrawText(barX+barW-TextWidth(label, 1), barY+barH+6, label, 1, RGB{226, 232, 240})
	c.DrawText(barX+70, barY-14, "MEAN CURVATURE ERROR", 1, RGB{226, 232, 240})

	info := fmt.Sprintf("MAE %.4f  MAX %.4f  N %d", stats.MAE, stats.Max, stats.Count)
	c.FillRect(430, panelH-58, 340, 34, RGB{24, 31, 42})
	c.DrawText(442, panelH-48, info, 1, RGB{226, 232, 240})
	c.DrawText(442, panelH-34, "BOUNDARY RING EXCLUDED", 1, RGB{148, 163, 184})
	return c.SavePNG(path)
}

func SaveQuadSheet(path string, panels [4]image.Image) error {
	sheet := image.NewRGBA(image.Rect(0, 0, panelW*2, panelH*2))
	for idx, img := range panels {
		if img == nil {
			continue
		}
		x := (idx % 2) * panelW
		y := (idx / 2) * panelH
		r := image.Rect(x, y, x+panelW, y+panelH)
		draw.Draw(sheet, r, img, image.Point{}, draw.Src)
	}
	return SavePNG(path, sheet)
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
