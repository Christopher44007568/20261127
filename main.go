package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	out := flag.String("out", "artifacts", "directory for PNG, OBJ, JSON and text outputs")
	samples := flag.Int("ao-samples", 18, "ambient-occlusion rays per vertex")
	flag.Parse()

	if err := run(*out, *samples); err != nil {
		fmt.Fprintln(os.Stderr, "uv unwrap failed:", err)
		os.Exit(1)
	}
}

func run(outDir string, aoSamples int) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	m := NewOrganicHead()
	cfg := DefaultUnwrapConfig()
	cfg.AOSamples = aoSamples
	atlas, sig, err := Unwrap(m, cfg)
	if err != nil {
		return err
	}
	report := BuildReport(m, sig, atlas)
	objPath := filepath.Join(outDir, "unwrapped_head.obj")
	jsonPath := filepath.Join(outDir, "metrics.json")
	txtPath := filepath.Join(outDir, "report.txt")
	uvPath := filepath.Join(outDir, "uv_islands.png")
	checkerPath := filepath.Join(outDir, "checker_model.png")
	heatPath := filepath.Join(outDir, "stretch_heatmap.png")
	seamPath := filepath.Join(outDir, "seam_plan.png")
	if err := WriteOBJ(objPath, m, atlas.WedgeUV, atlas.FaceWedge); err != nil {
		return err
	}
	if err := SaveJSON(jsonPath, report); err != nil {
		return err
	}
	if err := writeTextReport(txtPath, report); err != nil {
		return err
	}
	if err := RenderUVIslands(uvPath, m, sig, atlas, report); err != nil {
		return err
	}
	if err := RenderCheckerSheet(checkerPath, m, sig, atlas, report); err != nil {
		return err
	}
	if err := RenderHeatmapSheet(heatPath, m, sig, atlas, report); err != nil {
		return err
	}
	if err := RenderSeamSheet(seamPath, m, sig, atlas, report); err != nil {
		return err
	}
	report.Outputs["uv_islands"] = uvPath
	report.Outputs["checker_model"] = checkerPath
	report.Outputs["stretch_heatmap"] = heatPath
	report.Outputs["seam_plan"] = seamPath
	if err := SaveJSON(jsonPath, report); err != nil {
		return err
	}
	fmt.Printf("islands=%d tris=%d wedges=%d seams=%d frontSeams=%d flips=%d repairs=%d\n",
		report.IslandCount, report.Triangles, report.Wedges, report.SeamEdges,
		report.VisibleFrontSeamEdges, report.FlippedFaces, report.RepairRounds)
	fmt.Printf("area median=%.3f p95=%.3f | L2 median=%.3f p95=%.3f max=%.3f\n",
		report.AreaRatio.Median, report.AreaRatio.P95,
		report.L2Stretch.Median, report.L2Stretch.P95, report.L2Stretch.Max)
	fmt.Println("wrote PNG renders, OBJ and", jsonPath)
	return nil
}

func writeTextReport(path string, r Report) error {
	text := fmt.Sprintf(`Organic UV seam / LSCM report

mesh: %d vertices, %d triangles, %d UV wedges
islands: %d, triangle counts=%v
seam edges: %d, world length %.3f
protected face seam edges: %d
visible front seam edges: %d (length %.3f)
flipped faces: %d
adaptive repair rounds: %d
solver iterations: %v
mean vertex occlusion: %.3f

area ratio (UV area / 3D area, globally normalized):
  mean %.3f median %.3f p95 %.3f max %.3f
L2 stretch:
  mean %.3f median %.3f p95 %.3f max %.3f
L-infinity stretch:
  mean %.3f median %.3f p95 %.3f max %.3f
`, r.Vertices, r.Triangles, r.Wedges, r.IslandCount, r.IslandTriangles,
		r.SeamEdges, r.SeamLength, r.ProtectedSeamEdges,
		r.VisibleFrontSeamEdges, r.VisibleFrontSeamLength,
		r.FlippedFaces, r.RepairRounds, r.SolverIterations, r.MeanAO,
		r.AreaRatio.Mean, r.AreaRatio.Median, r.AreaRatio.P95, r.AreaRatio.Max,
		r.L2Stretch.Mean, r.L2Stretch.Median, r.L2Stretch.P95, r.L2Stretch.Max,
		r.LInfStretch.Mean, r.LInfStretch.Median, r.LInfStretch.P95, r.LInfStretch.Max)
	return os.WriteFile(path, []byte(text), 0o644)
}
