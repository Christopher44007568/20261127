package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for PNG/CSV/TXT outputs")
	flag.Parse()
	if err := run(*outDir); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	cfg := DefaultConfig()

	sheet := RenderSheet(cfg)
	if err := savePNG(sheet, outDir+"/walk_frames_sheet.png"); err != nil {
		return err
	}
	fmt.Println("wrote", outDir+"/walk_frames_sheet.png")

	rows := CollectMetrics(cfg, 2, 120)
	if err := WriteMetricsCSV(outDir+"/metrics.csv", rows); err != nil {
		return err
	}
	fmt.Println("wrote", outDir+"/metrics.csv")

	sweep := ReachSweep(cfg)
	if err := WriteSweepCSV(outDir+"/reach_sweep.csv", sweep); err != nil {
		return err
	}
	fmt.Println("wrote", outDir+"/reach_sweep.csv")

	summ := Summarize(cfg, rows, sweep)

	errImg := RenderErrorCharts(cfg, rows, sweep, summ)
	if err := savePNG(errImg, outDir+"/target_end_error.png"); err != nil {
		return err
	}
	fmt.Println("wrote", outDir+"/target_end_error.png")

	angImg := RenderJointAngles(cfg)
	if err := savePNG(angImg, outDir+"/joint_angle_curves.png"); err != nil {
		return err
	}
	fmt.Println("wrote", outDir+"/joint_angle_curves.png")

	if err := WriteWalkGIF(outDir+"/walk.gif", cfg, 24); err != nil {
		return err
	}
	fmt.Println("wrote", outDir+"/walk.gif")

	if err := WriteSummary(outDir+"/summary.txt", cfg, summ); err != nil {
		return err
	}
	if err := WriteHTML(outDir + "/index.html"); err != nil {
		return err
	}
	fmt.Println("wrote", outDir+"/summary.txt", "and", outDir+"/index.html")

	if summ.MaxIKError > 1e-9 {
		return fmt.Errorf("IK error too large: %e", summ.MaxIKError)
	}
	if summ.MaxLockDrift > 1e-9 {
		return fmt.Errorf("sole lock drift too large: %e", summ.MaxLockDrift)
	}
	if summ.KneeMin < -1e-9 {
		return fmt.Errorf("knee hyperextension detected: %.3f deg", summ.KneeMin)
	}
	fmt.Printf("checks ok: maxErr=%.2e maxLock=%.2e kneeRange=[%.1f,%.1f]\n",
		summ.MaxIKError, summ.MaxLockDrift, summ.KneeMin, summ.KneeMax)
	return nil
}
