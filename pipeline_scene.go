package main

import (
	"path/filepath"
)

func runScene(opts RunOptions, cfg FusionConfig, scene Scene) (*SceneReport, error) {
	frames := scene.Frames(opts.DepthWidth, opts.DepthHeight)
	grid := NewVoxelGrid(cfg)

	totalUpdates := 0
	holePixels := 0
	for _, frame := range frames {
		totalUpdates += FuseFrame(grid, frame, cfg)
		holePixels += frame.HolePixels
	}

	maxWeight := float32(0.0001)
	for _, w := range grid.Weight {
		if w > maxWeight {
			maxWeight = w
		}
	}

	iz := SliceIndexForZ(grid, scene.SliceZ)
	segments := ExtractSliceContour(grid, iz, 0.12)
	thresholdCells := ExtractThresholdCells(grid, 0.35, 0.18)
	mesh := ExtractMarchingTetrahedra(grid, 0.03)
	for i := range mesh.Vertices {
		mesh.Vertices[i].Error = absFloat(scene.Geometry.SDF(mesh.Vertices[i].Pos))
	}

	contourPoints := SegmentPoints(segments)
	contourStats := DistanceStats(contourPoints, scene.Geometry)
	meshPoints := make([]Vec3, len(mesh.Vertices))
	for i, v := range mesh.Vertices {
		meshPoints[i] = v.Pos
	}
	meshStats := DistanceStats(meshPoints, scene.Geometry)
	rawCloud := BuildRawCloud(frames, 2)
	rawStats := DistanceStats(rawCloud, scene.Geometry)
	truthSamples := DepthTruthSamples(frames, 4)
	coverage := MeasureMeshCoverage(mesh, truthSamples, opts.VoxelSize*2.25)

	sr := &SceneReport{
		ID:             scene.ID,
		Title:          scene.Title,
		ViewCount:      len(frames),
		HolePixels:     holePixels,
		VoxelCount:     grid.NX * grid.NY * grid.NZ,
		VoxelSize:      grid.Voxel,
		Truncation:     grid.Truncation,
		UpdatedVoxels:  totalUpdates,
		MaxWeight:      maxWeight,
		ThresholdCells: len(thresholdCells),
		MeshVertices:   len(mesh.Vertices),
		MeshTriangles:  len(mesh.Triangles),
		RawCloud:       rawStats,
		Contour:        contourStats,
		Mesh:           meshStats,
		Coverage:       coverage,
		Outputs:        map[string]string{},
	}

	thin := MeasureThinSheet(grid, iz, 0.045)
	if scene.ID == "thin_sheet" {
		sr.Thin = &thin
	}
	expectedCorner := V3(-0.39, 0.42, scene.SliceZ)
	probe := V3(-0.363, 0.393, scene.SliceZ)
	corner := MeasureCorner(grid, segments, iz, expectedCorner, probe)
	if scene.ID == "concave_corner" {
		sr.Corner = &corner
	}

	coarseH := 0.05
	coarseCfg := cfg
	coarseCfg.VoxelSize = coarseH
	coarseCfg.NX = 54
	coarseCfg.NY = 46
	coarseCfg.NZ = 28
	coarseCfg.Truncation = coarseH * 2
	coarseGrid := NewVoxelGrid(coarseCfg)
	for _, frame := range frames {
		FuseFrame(coarseGrid, frame, coarseCfg)
	}
	coarseIZ := SliceIndexForZ(coarseGrid, scene.SliceZ)
	coarseSegments := ExtractSliceContour(coarseGrid, coarseIZ, 0.12)
	coarsePoints := SegmentPoints(coarseSegments)
	coarseContour := DistanceStats(coarsePoints, scene.Geometry)
	coarseThin := MeasureThinSheet(coarseGrid, coarseIZ, 0.045)
	coarseCorner := MeasureCorner(coarseGrid, coarseSegments, coarseIZ, expectedCorner, probe)
	sr.CoarseControl = &CoarseControl{
		VoxelSize:  coarseH,
		Truncation: coarseCfg.Truncation,
		Contour:    coarseContour,
	}
	if scene.ID == "thin_sheet" {
		sr.CoarseControl.Thin = &coarseThin
	}
	if scene.ID == "concave_corner" {
		sr.CoarseControl.Corner = &coarseCorner
	}
	coarseMax := float32(0.0001)
	for _, w := range coarseGrid.Weight {
		if w > coarseMax {
			coarseMax = w
		}
	}
	coarsePath := filepath.Join(opts.OutDir, "section", scene.ID+"__coarse_control_section.png")
	if err := WriteVoxelSectionPNG(coarsePath, scene, coarseGrid, coarseIZ, coarseSegments, coarseMax); err != nil {
		return nil, err
	}
	sr.Outputs["coarse_section"] = coarsePath

	addCheck(sr, "fine_voxel", "voxel size stays fine", grid.Voxel, 0.025, "m", grid.Voxel <= 0.025)
	addCheck(sr, "contour_rmse", "slice contour RMSE", contourStats.RMSE, 0.018, "m", contourStats.Count > 0 && contourStats.RMSE <= 0.018)
	addCheck(sr, "mesh_rmse", "mesh vertex RMSE", meshStats.RMSE, 0.020, "m", meshStats.Count > 100 && meshStats.RMSE <= 0.020)
	addCheck(sr, "coverage", "ground-truth surface coverage", coverage.CoverageRatio, 0.90, "ratio", coverage.CoverageRatio >= 0.90)
	addCheck(sr, "fusion_beats_cloud", "TSDF contour RMSE lower than raw cloud", rawStats.RMSE, contourStats.RMSE*1.10, "m", contourStats.Count > 0 && contourStats.RMSE*1.10 < rawStats.RMSE)

	if scene.ID == "thin_sheet" {
		pass := len(thin.ThicknessValues) >= 4 && thin.ThicknessError <= 0.014 && thin.MeanThickness > 0.025
		addCheck(sr, "thin_thickness", "thin sheet thickness preserved", thin.ThicknessError, 0.014, "m", pass)
	}
	if scene.ID == "concave_corner" {
		addCheck(sr, "corner_position", "concave corner position preserved", corner.Error, 0.026, "m", corner.Error > 0 && corner.Error <= 0.026)
		addCheck(sr, "empty_bisector", "concave bisector is not filled", corner.ProbeMetricSDF, 0.008, "m", corner.ProbeMetricSDF >= 0.008 && corner.ProbeTSDF > 0)
	}
	if scene.ID == "wide_baseline" {
		addCheck(sr, "wide_p95", "wide-baseline p95 error bounded", contourStats.P95, 0.038, "m", contourStats.P95 <= 0.038)
	}
	if coarseContour.Count > 0 {
		addCheck(sr, "fine_beats_coarse", "15mm contour beats 50mm smoothing control", contourStats.RMSE, coarseContour.RMSE, "m", contourStats.RMSE < coarseContour.RMSE)
	}

	if err := writeSceneOutputs(opts.OutDir, scene, frames, grid, iz, segments, thresholdCells, mesh, rawCloud, *sr); err != nil {
		return nil, err
	}
	return sr, nil
}

func absFloat(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}

func writeSceneOutputs(
	outDir string,
	scene Scene,
	frames []*DepthFrame,
	grid *VoxelGrid,
	iz int,
	segments []Segment3,
	thresholdCells []ThresholdCell,
	mesh *Mesh,
	rawCloud []Vec3,
	sr SceneReport,
) error {
	setPath := func(key, category, suffix string) string {
		p := filepath.Join(outDir, category, scene.ID+"__"+suffix)
		sr.Outputs[key] = p
		return p
	}

	if err := WriteDepthMosaicPNG(setPath("depth_mosaic", "depth", "raw_depth_mosaic.png"), scene, frames); err != nil {
		return err
	}
	if err := WriteRawCloudSectionPNG(setPath("raw_cloud", "cloud", "raw_cloud_layers.png"), scene, grid, frames, sr.RawCloud); err != nil {
		return err
	}
	if err := WriteVoxelSectionPNG(setPath("voxel_section", "section", "voxel_section.png"), scene, grid, iz, segments, sr.MaxWeight); err != nil {
		return err
	}
	if err := WriteErrorHeatmapPNG(setPath("error_heatmap", "heatmap", "error_heatmap.png"), scene, grid, iz, segments, sr.Contour); err != nil {
		return err
	}
	if err := WriteMeshRenderPNG(setPath("surface_png", "surface", "surface.png"), mesh, scene, false); err != nil {
		return err
	}
	if err := WriteMeshRenderPNG(setPath("surface_error_png", "surface", "surface_error.png"), mesh, scene, true); err != nil {
		return err
	}
	if err := WriteMeshPLY(setPath("surface_ply", "mesh", "surface.ply"), mesh); err != nil {
		return err
	}
	if err := WriteThresholdPLY(setPath("threshold_ply", "points", "threshold_cells.ply"), thresholdCells); err != nil {
		return err
	}
	if err := WritePointsPLY(setPath("raw_cloud_ply", "points", "raw_cloud.ply"), rawCloud, nil); err != nil {
		return err
	}
	return nil
}
