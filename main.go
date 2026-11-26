package main

import (
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	opts := DefaultRunOptions()
	flag.StringVar(&opts.OutDir, "out", "out", "directory for PNG, PLY and report.json outputs")
	flag.IntVar(&opts.DepthWidth, "width", opts.DepthWidth, "synthetic depth image width")
	flag.IntVar(&opts.DepthHeight, "height", opts.DepthHeight, "synthetic depth image height")
	flag.Float64Var(&opts.VoxelSize, "voxel", opts.VoxelSize, "fine TSDF voxel size in metres")
	strict := flag.Bool("strict", false, "exit non-zero if geometric quality checks fail")
	flag.Parse()

	report, err := Run(opts)
	if err != nil {
		log.Fatalf("TSDF lab failed: %v", err)
	}

	fmt.Printf("TSDF lab finished: %d scenes, pass=%v\n", len(report.Scenes), report.Pass)
	for _, s := range report.Scenes {
		fmt.Printf("  %-16s contour RMSE=%6.2f mm  mesh RMSE=%6.2f mm  coverage=%.3f  tris=%d\n",
			s.ID, s.Contour.RMSE*1000, s.Mesh.RMSE*1000, s.Coverage.CoverageRatio, s.MeshTriangles)
		for _, chk := range s.Checks {
			status := "PASS"
			if !chk.Pass {
				status = "FAIL"
			}
			fmt.Printf("    [%s] %-24s value=%.4g threshold=%.4g %s\n", status, chk.ID, chk.Value, chk.Threshold, chk.Unit)
		}
	}
	fmt.Printf("Report: %s/report.json\n", opts.OutDir)
	if *strict && !report.Pass {
		os.Exit(2)
	}
}
