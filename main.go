package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	input := flag.String("input", "", "input .ply/.xyz/.txt point cloud (default: built-in noisy holed shell)")
	out := flag.String("out", "out", "output directory")
	cell := flag.Float64("cell", 0, "implicit grid cell size (default: adaptive from point spacing)")
	seed := flag.Int64("seed", 20261127, "deterministic demo random seed")
	yaw := flag.Float64("yaw", -0.68, "view yaw in radians")
	pitch := flag.Float64("pitch", 0.58, "view pitch in radians")
	strict := flag.Bool("strict", false, "exit non-zero if geometric quality checks fail")
	flag.Parse()

	report, err := Run(RunConfig{
		Input:  *input,
		Out:    *out,
		Cell:   *cell,
		Seed:   *seed,
		Yaw:    *yaw,
		Pitch:  *pitch,
		Strict: *strict,
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	fmt.Printf("points raw=%d kept=%d outliers=%d spacing=%.5f\n",
		report.RawPoints, report.KeptPoints, report.RemovedOutliers, report.PointSpacing)
	fmt.Printf("normals flips=%d components=%d\n", report.Orientation.Flips, report.Orientation.Components)
	fmt.Printf("mesh vertices=%v triangles=%v boundaryEdges=%v\n",
		report.Mesh["vertices"], report.Mesh["triangles"], report.Mesh["boundary_edges"])
	if report.Geometry != nil {
		fmt.Printf("geometry RMSE=%.5f signedBias=%.5f\n", report.Geometry.RMSE, report.Geometry.SignedMean)
	}
	if report.Curvature != nil {
		fmt.Printf("curvature RMSE=%.5f P95=%.5f\n", report.Curvature.RMSE, report.Curvature.P95)
	}
	failed := 0
	for _, chk := range report.Checks {
		fmt.Println(chk.Name, "-", chk.Detail)
		if !chk.Pass {
			failed++
		}
	}
	fmt.Println("report:", report.Outputs["contact_sheet"])
	if *strict && failed > 0 {
		os.Exit(2)
	}
}
