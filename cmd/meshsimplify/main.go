package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	qem "qemsimplify"
)

func main() {
	var (
		input         = flag.String("input", "", "input OBJ; when empty, built-in orb and boundary patch demos are generated")
		outDir        = flag.String("out", "output", "output directory")
		name          = flag.String("name", "", "case name for a single OBJ input")
		ratiosText    = flag.String("ratios", "1,0.75,0.5,0.35,0.22", "comma-separated retained triangle ratios")
		maxVolumeLoss = flag.Float64("max-volume-loss", qem.DefaultOptions().MaxVolumeLoss, "maximum physical volume loss fraction")
		sharpAngle    = flag.Float64("sharp-angle", qem.DefaultOptions().SharpNormalAngleDegrees, "sharp normal edge lock angle in degrees")
		noSharp       = flag.Bool("no-sharp-lock", false, "do not lock sharp dihedral/normal-detail edges")
		noFreeze      = flag.Bool("no-freeze", false, "allow locked protected vertices to move while keeping their IDs")
		noLockVerts   = flag.Bool("no-lock-vertices", false, "only reject the protected edge itself; do not lock endpoints")
		noOBJ         = flag.Bool("no-obj", false, "do not write progressive OBJ files")
	)
	flag.Parse()

	ratios, err := parseRatios(*ratiosText)
	if err != nil {
		fatal(err)
	}
	opt := qem.DefaultOptions()
	opt.MaxVolumeLoss = *maxVolumeLoss
	opt.SharpNormalAngleDegrees = *sharpAngle
	opt.ProtectSharpNormals = !*noSharp
	opt.FreezeLockedPositions = !*noFreeze
	opt.LockProtectedVertices = !*noLockVerts
	cfg := qem.DefaultRunConfig()
	cfg.SaveOBJ = !*noOBJ

	if *input != "" {
		mesh, err := qem.LoadOBJ(*input)
		if err != nil {
			fatal(err)
		}
		caseName := *name
		if caseName == "" {
			base := filepath.Base(*input)
			caseName = strings.TrimSuffix(base, filepath.Ext(base))
		}
		if err := runAndReport(mesh, caseName, *outDir, ratios, opt, cfg); err != nil {
			fatal(err)
		}
		return
	}

	orb := qem.NewDetailOrb(4)
	patch := qem.NewWavyPatch(33, 33)
	modelDir := filepath.Join(*outDir, "models")
	if err := os.MkdirAll(modelDir, 0o755); err != nil {
		fatal(err)
	}
	if err := orb.WriteOBJ(filepath.Join(modelDir, "detail-orb.obj")); err != nil {
		fatal(err)
	}
	if err := patch.WriteOBJ(filepath.Join(modelDir, "wavy-boundary-patch.obj")); err != nil {
		fatal(err)
	}
	if err := runAndReport(orb, "detail_orb", filepath.Join(*outDir, "detail_orb"), ratios, opt, cfg); err != nil {
		fatal(err)
	}
	if err := runAndReport(patch, "wavy_boundary_patch", filepath.Join(*outDir, "wavy_boundary_patch"), ratios, opt, cfg); err != nil {
		fatal(err)
	}
}

func runAndReport(mesh *qem.Mesh, name, dir string, ratios []float64, opt qem.Options, cfg qem.RunConfig) error {
	report, err := qem.RunCase(mesh, name, dir, ratios, opt, cfg)
	if err != nil {
		return err
	}
	if err := qem.SaveJSON(filepath.Join(dir, "report.json"), report); err != nil {
		return err
	}
	if err := qem.WriteMetricsCSV(filepath.Join(dir, "silhouette-metrics.csv"), report); err != nil {
		return err
	}

	fmt.Printf("\n[%s] initial triangles=%d vertices=%d protected=%v undoExact=%v\n",
		name, report.InitialTriangles, report.InitialVertices, report.ProtectedEdges, report.UndoExact)
	for _, level := range report.Levels {
		bad := len(level.Verification.FlippedFaces) + len(level.Verification.DegenerateFaces) +
			len(level.Verification.MissingProtectedEdges) + len(level.Verification.InactiveProtectedVertices) +
			len(level.Verification.FrozenPositionViolations)
		fmt.Printf("  %s tris=%5d (%5.1f%%) collapses=%4d areaRatio=%.4f minView=%.4f IoU=%.4f chamfer=%.5f volRatio=%.5f violations=%d\n",
			level.Prefix, level.ActiveTriangles, 100*level.TriangleRatio,
			level.TotalCollapses, level.SilhouetteSummary.MeanAreaRatio,
			level.SilhouetteSummary.MinAreaRatio, level.SilhouetteSummary.MeanIoU,
			level.SilhouetteSummary.MeanChamferUnits, level.VolumeRatio, bad)
		if level.StopReason != "" {
			fmt.Printf("    stop: %s\n", level.StopReason)
		}
	}
	fmt.Printf("  outputs: %s\n", dir)
	return nil
}

func parseRatios(text string) ([]float64, error) {
	parts := strings.Split(text, ",")
	out := make([]float64, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		v, err := strconv.ParseFloat(part, 64)
		if err != nil || v <= 0 || v > 1 {
			return nil, fmt.Errorf("invalid ratio %q: must be in (0,1]", part)
		}
		out = append(out, v)
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("at least one ratio is required")
	}
	return out, nil
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
