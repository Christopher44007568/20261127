package qem

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
)

func SaveJSON(path string, value any) error {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return err
		}
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	return enc.Encode(value)
}

func WriteMetricsCSV(path string, report *CaseReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	header := []string{
		"case", "level", "prefix", "requested_ratio", "triangle_ratio",
		"active_triangles", "total_collapses", "view", "yaw_degrees",
		"original_pixels", "simplified_pixels", "area_ratio", "iou",
		"chamfer_pixels", "hausdorff_pixels", "chamfer_units", "hausdorff_units",
		"mean_area_ratio_all_views", "min_area_ratio_all_views",
		"mean_iou_all_views", "mean_chamfer_units_all_views",
		"max_hausdorff_units_all_views", "volume_ratio", "flip_count",
		"missing_protected_edge_count",
	}
	if err := w.Write(header); err != nil {
		return err
	}
	for _, level := range report.Levels {
		flipCount := len(level.Verification.FlippedFaces) + len(level.Verification.DegenerateFaces)
		missingLocks := len(level.Verification.MissingProtectedEdges) +
			len(level.Verification.InactiveProtectedVertices) +
			len(level.Verification.FrozenPositionViolations)
		for _, m := range level.SilhouetteMetrics {
			row := []string{
				report.Name,
				strconv.Itoa(level.Index),
				level.Prefix,
				strconv.FormatFloat(level.RequestedRatio, 'f', 4, 64),
				strconv.FormatFloat(level.TriangleRatio, 'f', 6, 64),
				strconv.Itoa(level.ActiveTriangles),
				strconv.Itoa(level.TotalCollapses),
				strconv.Itoa(m.ViewIndex),
				strconv.FormatFloat(m.YawDegrees, 'f', 1, 64),
				strconv.Itoa(m.OriginalPixels),
				strconv.Itoa(m.SimplifiedPixels),
				strconv.FormatFloat(m.AreaRatio, 'f', 6, 64),
				strconv.FormatFloat(m.IoU, 'f', 6, 64),
				strconv.FormatFloat(m.ChamferPixels, 'f', 4, 64),
				strconv.FormatFloat(m.HausdorffPixels, 'f', 4, 64),
				strconv.FormatFloat(m.ChamferUnits, 'f', 6, 64),
				strconv.FormatFloat(m.HausdorffUnits, 'f', 6, 64),
				strconv.FormatFloat(level.SilhouetteSummary.MeanAreaRatio, 'f', 6, 64),
				strconv.FormatFloat(level.SilhouetteSummary.MinAreaRatio, 'f', 6, 64),
				strconv.FormatFloat(level.SilhouetteSummary.MeanIoU, 'f', 6, 64),
				strconv.FormatFloat(level.SilhouetteSummary.MeanChamferUnits, 'f', 6, 64),
				strconv.FormatFloat(level.SilhouetteSummary.MaxHausdorffUnits, 'f', 6, 64),
				strconv.FormatFloat(level.VolumeRatio, 'f', 8, 64),
				strconv.Itoa(flipCount),
				strconv.Itoa(missingLocks),
			}
			if err := w.Write(row); err != nil {
				return err
			}
		}
	}
	w.Flush()
	return w.Error()
}
