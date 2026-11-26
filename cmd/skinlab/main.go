package main

import (
	"flag"
	"fmt"
	"log"
	"math"
	"path/filepath"

	"skinlab/internal/report"
	"skinlab/internal/skin"
)

func main() {
	bendDeg := flag.Float64("bend", 90, "final elbow flexion in degrees")
	twistDeg := flag.Float64("twist", 120, "final forearm axial twist in degrees")
	sweepSteps := flag.Int("sweep-steps", 31, "number of samples in the trajectory sweep")
	outDir := flag.String("out", "artifacts", "directory for the HTML report and CSV files")
	flag.Parse()

	if *sweepSteps < 2 {
		log.Fatal("sweep-steps must be at least 2")
	}
	if math.Abs(*bendDeg) >= 180 {
		log.Fatal("bend must be between -180 and 180 degrees (exclusive)")
	}

	mesh := skin.BuildTube()
	identity := skin.ElbowPose("bind", 0, 0)
	bind := skin.Analyze(mesh, identity)
	bindDistance, bindExact := skin.BindPoseMismatch(bind.LBS, bind.DQ)

	bend := *bendDeg * math.Pi / 180
	twist := *twistDeg * math.Pi / 180
	final := skin.Analyze(mesh, skin.ElbowPose("final", bend, twist))
	markers := skin.MarkerVertexIndices(mesh)
	trajectories := skin.BuildTrajectories(mesh, markers, bend, twist, *sweepSteps)

	if _, err := report.Write(*outDir, report.Dataset{
		Mesh:            mesh,
		Bind:            bind,
		Final:           final,
		Trajectory:      trajectories,
		Markers:         markers,
		BindExact:       bindExact,
		BindMaxDistance: bindDistance,
	}); err != nil {
		log.Fatal(err)
	}

	reportPath, err := filepath.Abs(filepath.Join(*outDir, "index.html"))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("LBS 截面积 %.1f%%–%.1f%%\n", final.MinLBSRatio*100, final.MaxLBSRatio*100)
	fmt.Printf("DQ  截面积 %.1f%%–%.1f%%\n", final.MinDQRatio*100, final.MaxDQRatio*100)
	fmt.Printf("绑定姿态逐位一致: %t (max error %.3g)\n", bindExact, bindDistance)
	fmt.Printf("报告: %s\n", reportPath)
}
