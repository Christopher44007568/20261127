package pipeline

import (
	"encoding/csv"
	"os"
	"strconv"

	"materiallod/material"
)

func writeCSVFile(path string, header []string, rows [][]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.Write(header); err != nil {
		return err
	}
	if err := w.WriteAll(rows); err != nil {
		return err
	}
	w.Flush()
	return w.Error()
}

func writeBrightnessCSV(dir string, stats []material.BrightnessStats) error {
	rows := make([][]string, 0, len(stats))
	for _, s := range stats {
		rows = append(rows, []string{
			s.MaterialID, s.Compare, itoa(s.Pixels), f6(s.MAE), f6(s.RMSE),
			f6(s.MaxAbs), f6(s.MeanSigned),
		})
	}
	return writeCSVFile(outputPath(dir, "brightness_difference.csv"),
		[]string{"material_id", "compare", "pixels", "mae", "rmse", "max_abs", "mean_signed"}, rows)
}

func writeViewSweepCSV(dir string, samples []material.ViewSample) error {
	rows := make([][]string, 0, len(samples))
	for _, s := range samples {
		rows = append(rows, []string{
			s.MaterialID, s.MaterialKind, s.Tier, f1(s.AngleDeg), f6(s.LinearLuminance),
			f6(s.DisplayLuma), f6(s.DisplayR), f6(s.DisplayG), f6(s.DisplayB),
		})
	}
	return writeCSVFile(outputPath(dir, "view_sweep.csv"),
		[]string{
			"material_id", "material_kind", "tier", "angle_deg", "linear_luminance",
			"display_luma", "display_r", "display_g", "display_b",
		}, rows)
}

func writeBoundaryCSV(dir string, rowsIn []material.BoundaryRow) error {
	rows := make([][]string, 0, len(rowsIn))
	for _, r := range rowsIn {
		rows = append(rows, []string{
			f2(r.LOD), r.MaterialID, r.BoundaryZone, r.HardTier,
			f4(r.Weights.Low), f4(r.Weights.Medium), f4(r.Weights.High),
			f6(r.MAEvsHigh), f6(r.RMSEvsHigh), f6(r.MaxvsHigh), f6(r.MaxStepJump),
		})
	}
	return writeCSVFile(outputPath(dir, "boundary_errors.csv"),
		[]string{
			"lod", "material_id", "boundary_zone", "hard_tier", "w_low",
			"w_medium", "w_high", "mae_vs_high", "rmse_vs_high",
			"max_vs_high", "max_step_jump",
		}, rows)
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	neg := v < 0
	if neg {
		v = -v
	}
	var b [20]byte
	i := len(b)
	for v > 0 {
		i--
		b[i] = byte('0' + v%10)
		v /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

func f1(x float64) string { return formatFloat(x, 1) }
func f2(x float64) string { return formatFloat(x, 2) }
func f4(x float64) string { return formatFloat(x, 4) }

func formatFloat(x float64, precision int) string {
	return strconv.FormatFloat(x, 'f', precision, 64)
}
