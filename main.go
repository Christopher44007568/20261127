package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

// Scenario records one rendered expression together with the effective
// weights and the artifact paths produced for it.
type Scenario struct {
	Name             string    `json:"name"`
	RawWeights       []float64 `json:"rawWeights"`
	ClampedWeights   []float64 `json:"clampedWeights"`
	EffectiveWeights []float64 `json:"effectiveWeights"`
	WeightSum        float64   `json:"clampedWeightSum"`
	PLY              string    `json:"ply"`
	NormalRGB        string    `json:"normalDiffRGB"`
	NormalMag        string    `json:"normalDiffMagnitude"`
}

type report struct {
	TargetNames []string   `json:"targets"`
	GridU       int        `json:"gridU"`
	GridV       int        `json:"gridV"`
	Scenarios   []Scenario `json:"scenarios"`
}

func main() {
	outDir := flag.String("out", "artifacts/render", "output directory for PLY and PNG artifacts")
	imageSize := flag.Int("size", 512, "normal-difference map resolution in pixels")
	flag.Parse()

	if err := run(*outDir, *imageSize); err != nil {
		fmt.Fprintln(os.Stderr, "facemorph:", err)
		os.Exit(1)
	}
}

func run(outDir string, imageSize int) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	base := BuildBaseFace()
	targets := BuildTargets(base)
	blender := &Blender{Base: base, Targets: targets}
	names := make([]string, len(targets))
	for i, target := range targets {
		names[i] = target.Name
	}

	scenarios := []struct {
		name    string
		weights []float64
	}{
		{"single_smile", []float64{1, 0, 0, 0}},
		{"single_jaw_open", []float64{0, 0, 0, 1}},
		{"combined_smile_fold_brow", []float64{0.8, 0.8, 0.6, 0}},
		{"combined_full_face", []float64{1, 1, 1, 1}},
		{"extreme_clamped", []float64{9, -4, 0.5, 2.5}},
		{"all_off_identity", []float64{0, 0, 0, 0}},
	}

	rendered := make([]Scenario, 0, len(scenarios))
	for _, scenario := range scenarios {
		result := blender.Blend(scenario.weights)

		plyPath, err := WritePLY(outDir, scenario.name, result.Mesh)
		if err != nil {
			return err
		}
		rgbPath, magPath, err := WriteNormalDiffMap(outDir, scenario.name, base, result.Mesh, imageSize)
		if err != nil {
			return err
		}

		rendered = append(rendered, Scenario{
			Name:             scenario.name,
			RawWeights:       append([]float64(nil), scenario.weights...),
			ClampedWeights:   append([]float64(nil), result.ClampedWeights...),
			EffectiveWeights: append([]float64(nil), result.EffectiveWeights...),
			WeightSum:        result.Denominator,
			PLY:              plyPath,
			NormalRGB:        rgbPath,
			NormalMag:        magPath,
		})
	}

	summary := report{
		TargetNames: names,
		GridU:       gridU,
		GridV:       gridV,
		Scenarios:   rendered,
	}
	reportPath := filepath.Join(outDir, "report.json")
	data, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(reportPath, data, 0o644); err != nil {
		return err
	}

	fmt.Printf("rendered %d scenarios to %s\nreport: %s\n", len(rendered), outDir, reportPath)
	return nil
}
