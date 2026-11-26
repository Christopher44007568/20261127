package report

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
)

// WriteCSV dumps the full sampled dataset for all methods.
func WriteCSV(w io.Writer, all []Series) error {
	cw := csv.NewWriter(w)
	header := []string{
		"method", "time", "quat_w", "quat_x", "quat_y", "quat_z",
		"blend_norm", "omega_x", "omega_y", "omega_z",
		"angular_speed", "ortho_error", "degenerate",
	}
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, s := range all {
		for i, sample := range s.Samples {
			row := []string{
				string(s.Method),
				strconv.FormatFloat(sample.T, 'f', 3, 64),
				strconv.FormatFloat(sample.Quat.W, 'g', 10, 64),
				strconv.FormatFloat(sample.Quat.X, 'g', 10, 64),
				strconv.FormatFloat(sample.Quat.Y, 'g', 10, 64),
				strconv.FormatFloat(sample.Quat.Z, 'g', 10, 64),
				strconv.FormatFloat(s.BlendNorm[i], 'g', 10, 64),
				num(s.OmegaX[i]), num(s.OmegaY[i]), num(s.OmegaZ[i]),
				num(s.Speed[i]), num(s.Ortho[i]),
				strconv.FormatBool(sample.Degenerate),
			}
			if err := cw.Write(row); err != nil {
				return err
			}
		}
	}
	cw.Flush()
	return cw.Error()
}

func num(v float64) string {
	if math.IsNaN(v) {
		return "NaN"
	}
	return strconv.FormatFloat(v, 'g', 10, 64)
}

// Summary returns a human-readable comparison table.
func Summary(all []Series, arcBudget float64) string {
	out := fmt.Sprintf("minimum shortest-arc budget across all segments: %.4f rad (%.2f deg)\n\n",
		arcBudget, arcBudget*180/math.Pi)
	out += fmt.Sprintf("%-26s %14s %14s %14s %12s\n",
		"method", "swept rad", "swept deg", "max |ω| rad/s", "degenerate")
	for _, s := range all {
		out += fmt.Sprintf("%-26s %14.4f %14.2f %14.3f %12d\n",
			string(s.Method), s.SweptAngle, s.SweptAngle*180/math.Pi,
			s.MaxSpeed, s.Degenerate)
	}
	out += "\nmax orthogonality error per method:\n"
	for _, s := range all {
		out += fmt.Sprintf("  %-26s %.3e\n", string(s.Method), s.MaxOrtho)
	}
	return out
}
