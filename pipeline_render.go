package main

import (
	"image"
	"math"
	"os"
	"path/filepath"
)

func prepareDemoCurvatureColors(mesh *Mesh, field *FieldGrid, scene *DemoScene, mask []bool) (ErrorStats, float64) {
	score := ScoreDemoFieldCurvature(mesh, field, scene, mask)
	scale := math.Max(score.Stats.P95*1.8, score.Stats.RMSE*3.0)
	scale = math.Max(scale, 0.35)
	mesh.C = make([]RGB, len(mesh.P))
	mesh.V = make([]float64, len(mesh.P))
	invalid := RGB{48, 56, 66}
	for i := range mesh.P {
		mesh.C[i] = invalid
		mesh.V[i] = -1
		if score.Valid[i] {
			av := absFloat(score.Error[i])
			mesh.V[i] = av
			mesh.C[i] = HeatColor(av / scale)
		}
	}
	return score.Stats, scale
}

func prepareGenericCurvatureColors(mesh *Mesh) ErrorStats {
	values := EstimateMeshCurvature(mesh, 32)
	absValues := make([]float64, 0, len(values))
	for _, v := range values {
		absValues = append(absValues, absFloat(v))
	}
	stats := statsFromValues(absValues)
	scale := stats.P95 * 1.2
	if scale < 0.5 {
		scale = 0.5
	}
	mesh.C = make([]RGB, len(mesh.P))
	mesh.V = make([]float64, len(mesh.P))
	for i, v := range values {
		mesh.V[i] = absFloat(v)
		mesh.C[i] = HeatColor(absFloat(v) / scale)
	}
	return stats
}

func renderPanels(out string, raw *Cloud, removed map[int]bool, filtered *Cloud, mesh *Mesh,
	field *FieldGrid, scene *DemoScene, boundaryMask []bool, cam Camera, report *RunReport) (*ErrorStats, error) {
	p1 := filepath.Join(out, "01_input_points.png")
	p2 := filepath.Join(out, "02_normal_color.png")
	p3 := filepath.Join(out, "03_reconstructed_mesh.png")
	p4 := filepath.Join(out, "04_curvature_error.png")
	rawN := filepath.Join(out, "diagnostics", "raw_inconsistent_normals.png")
	sheet := filepath.Join(out, "panel.png")

	if err := RenderInputPanel(p1, raw, removed, cam); err != nil {
		return nil, err
	}
	if err := RenderNormalPanel(p2, filtered.P, filtered.N, cam,
		"2 CONSISTENT NORMALS", "PCA + KNN MINIMUM SPANNING TREE ORIENTATION"); err != nil {
		return nil, err
	}
	if raw.HasN && len(raw.N) == len(raw.P) {
		if err := RenderNormalPanel(rawN, raw.P, raw.N, cam,
			"2A RAW NORMALS: FLIPPED", "INPUT SIGNS ARE LOCALLY INCONSISTENT"); err != nil {
			return nil, err
		}
		report.Outputs["raw_normals_diagnostic"] = rawN
	}
	if err := RenderMeshPanel(p3, mesh, cam); err != nil {
		return nil, err
	}
	if scene != nil && len(scene.Holes) > 0 {
		h := scene.Holes[0]
		closeBox := Box{
			Min: V3{h.X - 0.48, h.Y - 0.40, -0.34},
			Max: V3{h.X + 0.48, h.Y + 0.40, 0.34},
		}
		closeCam := FitCamera(closeBox, panelW, panelH, -0.52, 0.76, 0.86)
		closePath := filepath.Join(out, "diagnostics", "hole_boundary_closeup.png")
		if err := RenderMeshPanel(closePath, mesh, closeCam); err != nil {
			return nil, err
		}
		report.Outputs["hole_closeup"] = closePath
	}

	var curvStats *ErrorStats
	if scene != nil {
		stats, scale := prepareDemoCurvatureColors(mesh, field, scene, boundaryMask)
		curvStats = &stats
		if err := RenderCurvaturePanel(p4, mesh, cam, scale, stats); err != nil {
			return nil, err
		}
	} else {
		stats := prepareGenericCurvatureColors(mesh)
		curvStats = &stats
		c := NewCanvas(panelW, panelH, RGB{14, 18, 24})
		c.DrawMesh(cam, mesh, true, RGB{200, 200, 200}, V3{-0.35, -0.45, 0.82})
		c.DrawBoundaryEdges(cam, mesh, RGB{116, 235, 255})
		c.DrawTitleBar("4 CURVATURE MAGNITUDE", "NO GROUND TRUTH FOR EXTERNAL CLOUD; |H| SHOWN")
		if err := c.SavePNG(p4); err != nil {
			return nil, err
		}
	}

	imgs := [4]image.Image{}
	paths := [4]string{p1, p2, p3, p4}
	for i, path := range paths {
		img, err := loadPNGImage(path)
		if err != nil {
			return nil, err
		}
		imgs[i] = img
	}
	if err := SaveQuadSheet(sheet, imgs); err != nil {
		return nil, err
	}

	report.Outputs["contact_sheet"] = sheet
	report.Outputs["input_points"] = p1
	report.Outputs["normal_color"] = p2
	report.Outputs["mesh"] = p3
	report.Outputs["curvature_error"] = p4
	return curvStats, nil
}

func loadPNGImage(path string) (image.Image, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	return img, err
}
