package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"orientation-interp/interp"
	"orientation-interp/report"
)

func main() {
	outDir := "artifacts"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatalf("create output dir: %v", err)
	}

	keys := interp.Scenario()
	methods := []interp.Method{
		interp.MethodEuler,
		interp.MethodNlerp,
		interp.MethodSlerp,
	}
	series := make([]report.Series, len(methods))
	for i, method := range methods {
		series[i] = report.BuildSeries(method, keys, interp.ScenarioStep)
	}
	keyTimes := report.KeyframeTimes(keys)

	files := []struct {
		name string
		fn   func(*os.File) error
	}{
		{"pose_sequence.svg", func(f *os.File) error { return report.WritePoseStrip(f, series, keyTimes) }},
		{"angular_speed.svg", func(f *os.File) error { return report.WriteSpeedChart(f, series, keyTimes) }},
		{"ortho_error.svg", func(f *os.File) error { return report.WriteOrthoChart(f, series, keyTimes) }},
		{"blend_norm.svg", func(f *os.File) error { return report.WriteBlendNormChart(f, series, keyTimes) }},
	}
	for _, spec := range files {
		path := filepath.Join(outDir, spec.name)
		f, err := os.Create(path)
		if err != nil {
			log.Fatalf("create %s: %v", path, err)
		}
		if err := spec.fn(f); err != nil {
			f.Close()
			log.Fatalf("write %s: %v", path, err)
		}
		if err := f.Close(); err != nil {
			log.Fatalf("close %s: %v", path, err)
		}
		fmt.Println("wrote", path)
	}

	csvPath := filepath.Join(outDir, "samples.csv")
	cf, err := os.Create(csvPath)
	if err != nil {
		log.Fatalf("create %s: %v", csvPath, err)
	}
	if err := report.WriteCSV(cf, series); err != nil {
		cf.Close()
		log.Fatalf("write %s: %v", csvPath, err)
	}
	cf.Close()
	fmt.Println("wrote", csvPath)

	fmt.Println()
	fmt.Print(report.Summary(series, report.ArcBudget(keys)))
}
