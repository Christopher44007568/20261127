package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

type rotationReport struct {
	MeanRatio      float64 `json:"mean_ratio"`
	MeanAbsDelta   float64 `json:"mean_abs_delta_linear"`
	MaxMeanRatio   float64 `json:"max_view_mean_ratio"`
	SharedCoverage float64 `json:"shared_coverage"`
}

type renderReport struct {
	Program         string                  `json:"program"`
	Width           int                     `json:"width"`
	Height          int                     `json:"height"`
	Antialiasing    int                     `json:"antialiasing"`
	Exposure        float64                 `json:"exposure"`
	Views           map[string]ImageMetrics `json:"views"`
	BacklitEnergy   float64                 `json:"backlit_to_front_energy_ratio"`
	FiberRotation   rotationReport          `json:"fiber_rotation"`
	OutputDirectory string                  `json:"output_directory"`
}

func main() {
	outDir := flag.String("out", "renders", "directory for PPM files and metrics")
	width := flag.Int("width", 960, "output width")
	height := flag.Int("height", 800, "output height")
	aa := flag.Int("aa", 2, "supersampling grid size per axis")
	exposure := flag.Float64("exposure", 3.15, "linear exposure before tone mapping")
	flag.Parse()

	if *width < 64 || *height < 64 || *aa < 1 || *aa > 4 || *exposure <= 0 {
		fmt.Fprintln(os.Stderr, "invalid width, height, aa, or exposure")
		os.Exit(2)
	}
	opts := RenderOptions{Width: *width, Height: *height, AA: *aa, Exposure: *exposure}
	camera := makeCamera(V(0.38, 0.16, 5.75), V(0, -0.01, 0), 39)
	material := velvetMaterial()
	views := velvetViews()

	results := make(map[string]RenderResult, len(views))
	metrics := make(map[string]ImageMetrics, len(views))
	for _, view := range views {
		fmt.Printf("rendering %-8s %dx%d, aa %dx\n", view.Name, *width, *height, *aa)
		result := renderView(camera, view, material, opts)
		path := filepath.Join(*outDir, view.Name+".ppm")
		if err := writePPM(path, result, *exposure); err != nil {
			fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
			os.Exit(1)
		}
		results[view.Name] = result
		metrics[view.Name] = result.Metrics
		pm := result.Metrics
		fmt.Printf("  coverage %.1f%%  cloth %.5f  edge/interior %.2fx  over-one %.2f%%\n",
			pm.Coverage*100, pm.MeanClothLinear, pm.EdgeInteriorRatio, pm.OverOneFraction*100)
	}

	front := metrics["front"]
	backlit := metrics["backlit"]
	energyRatio := safeRatio(backlit.MeanClothLinear, front.MeanClothLinear)
	rotation := compareFiberRotation(results["front"], results["rotation"])
	report := renderReport{
		Program:         "velvet-sheen-go",
		Width:           *width,
		Height:          *height,
		Antialiasing:    *aa,
		Exposure:        *exposure,
		Views:           metrics,
		BacklitEnergy:   energyRatio,
		FiberRotation:   rotation,
		OutputDirectory: *outDir,
	}
	reportPath := filepath.Join(*outDir, "metrics.json")
	if err := writeReport(reportPath, report); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", reportPath, err)
		os.Exit(1)
	}
	fmt.Printf("backlit/front energy %.3fx, fiber rotation mean ratio %.3fx, delta %.5f\n",
		energyRatio, rotation.MeanRatio, rotation.MeanAbsDelta)
	fmt.Printf("wrote %s, %s, and %s\n",
		filepath.Join(*outDir, "front.ppm"),
		filepath.Join(*outDir, "backlit.ppm"),
		filepath.Join(*outDir, "rotation.ppm"))
}

func compareFiberRotation(front, rotated RenderResult) rotationReport {
	var sumA, sumB, sumAbs, shared float64
	for i := range front.Pixels {
		if front.Coverage[i] < 0.5 || rotated.Coverage[i] < 0.5 {
			continue
		}
		a := front.Pixels[i].Luminance()
		b := rotated.Pixels[i].Luminance()
		sumA += a
		sumB += b
		sumAbs += math.Abs(a - b)
		shared++
	}
	meanA := safeRatio(sumA, shared)
	meanB := safeRatio(sumB, shared)
	return rotationReport{
		MeanRatio:      safeRatio(meanB, meanA),
		MeanAbsDelta:   safeRatio(sumAbs, shared),
		MaxMeanRatio:   math.Max(safeRatio(meanB, meanA), safeRatio(meanA, meanB)),
		SharedCoverage: shared,
	}
}

func writeReport(path string, report renderReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
