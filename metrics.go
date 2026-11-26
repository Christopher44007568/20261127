package qem

import "math"

type SilhouetteMetric struct {
	ViewIndex        int     `json:"view_index"`
	YawDegrees       float64 `json:"yaw_degrees"`
	PitchDegrees     float64 `json:"pitch_degrees"`
	OriginalPixels   int     `json:"original_pixels"`
	SimplifiedPixels int     `json:"simplified_pixels"`
	AreaRatio        float64 `json:"area_ratio"`
	IoU              float64 `json:"iou"`
	ChamferPixels    float64 `json:"chamfer_pixels"`
	HausdorffPixels  float64 `json:"hausdorff_pixels"`
	ChamferUnits     float64 `json:"chamfer_units"`
	HausdorffUnits   float64 `json:"hausdorff_units"`
}

type SilhouetteSummary struct {
	ViewCount          int     `json:"view_count"`
	MeanAreaRatio      float64 `json:"mean_area_ratio"`
	MinAreaRatio       float64 `json:"min_area_ratio"`
	MeanIoU            float64 `json:"mean_iou"`
	MeanChamferUnits   float64 `json:"mean_chamfer_units"`
	MaxHausdorffUnits  float64 `json:"max_hausdorff_units"`
	MeanChamferPixels  float64 `json:"mean_chamfer_pixels"`
	MaxHausdorffPixels float64 `json:"max_hausdorff_pixels"`
}

func CompareSilhouetteAtView(originalMesh, simplifiedMesh *Mesh, frame ViewFrame, view View, viewIndex int) (SilhouetteMetric, *PPM) {
	original := RenderMask(originalMesh, frame, view)
	simplified := RenderMask(simplifiedMesh, frame, view)
	overlay := SilhouetteOverlay(original, simplified)

	originalCount := original.PixelCount()
	simplifiedCount := simplified.PixelCount()
	intersection := 0
	for i := range original.Inside {
		if original.Inside[i] && simplified.Inside[i] {
			intersection++
		}
	}
	union := originalCount + simplifiedCount - intersection
	areaRatio := 0.0
	if originalCount > 0 {
		areaRatio = float64(simplifiedCount) / float64(originalCount)
	}
	iou := 0.0
	if union > 0 {
		iou = float64(intersection) / float64(union)
	}

	distOriginal := original.DistanceMap()
	distSimplified := simplified.DistanceMap()
	meanOrig, maxOrig, _ := meanMaxBoundaryDistance(original, distSimplified)
	meanSimp, maxSimp, _ := meanMaxBoundaryDistance(simplified, distOriginal)
	chamferPx := (meanOrig + meanSimp) * 0.5
	hausdorffPx := math.Max(maxOrig, maxSimp)

	return SilhouetteMetric{
		ViewIndex:        viewIndex,
		YawDegrees:       math.Mod(view.Yaw*180/math.Pi, 360),
		PitchDegrees:     view.Pitch * 180 / math.Pi,
		OriginalPixels:   originalCount,
		SimplifiedPixels: simplifiedCount,
		AreaRatio:        areaRatio,
		IoU:              iou,
		ChamferPixels:    chamferPx,
		HausdorffPixels:  hausdorffPx,
		ChamferUnits:     frame.UnprojectPixelLength(chamferPx),
		HausdorffUnits:   frame.UnprojectPixelLength(hausdorffPx),
	}, overlay
}

func CompareSilhouettes(originalMesh, simplifiedMesh *Mesh, frame ViewFrame, views []View) ([]SilhouetteMetric, []*PPM, SilhouetteSummary) {
	rows := make([]SilhouetteMetric, 0, len(views))
	overlays := make([]*PPM, 0, len(views))
	summary := SilhouetteSummary{ViewCount: len(views), MinAreaRatio: math.Inf(1)}
	var sumArea, sumIoU, sumChamfer, sumChamferPx float64
	for i, view := range views {
		row, overlay := CompareSilhouetteAtView(originalMesh, simplifiedMesh, frame, view, i)
		rows = append(rows, row)
		overlays = append(overlays, overlay)
		sumArea += row.AreaRatio
		sumIoU += row.IoU
		sumChamfer += row.ChamferUnits
		sumChamferPx += row.ChamferPixels
		summary.MinAreaRatio = math.Min(summary.MinAreaRatio, row.AreaRatio)
		summary.MaxHausdorffUnits = math.Max(summary.MaxHausdorffUnits, row.HausdorffUnits)
		summary.MaxHausdorffPixels = math.Max(summary.MaxHausdorffPixels, row.HausdorffPixels)
	}
	n := float64(len(views))
	if n > 0 {
		summary.MeanAreaRatio = sumArea / n
		summary.MeanIoU = sumIoU / n
		summary.MeanChamferUnits = sumChamfer / n
		summary.MeanChamferPixels = sumChamferPx / n
	}
	if math.IsInf(summary.MinAreaRatio, 1) {
		summary.MinAreaRatio = 0
	}
	return rows, overlays, summary
}
