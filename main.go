package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"path/filepath"

	wf "whitefurnace/furnace"
)

type outputReport struct {
	ZeroDependencies bool               `json:"zero_dependencies"`
	AllPass          bool               `json:"all_pass"`
	Tolerance        float64            `json:"tolerance"`
	Materials        []wf.FurnaceReport `json:"materials"`
	Outputs          map[string]string  `json:"outputs"`
}

func main() {
	outDir := flag.String("out", "out", "directory for PNG and JSON outputs")
	panel := flag.Int("panel", 380, "material panel size in pixels")
	flag.Parse()

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		log.Fatal(err)
	}

	reports := wf.RunAllWhiteFurnace()
	allPass := wf.AllPass(reports)

	outputs := map[string]string{
		"original":     filepath.Join(*outDir, "original_materials.png"),
		"corrected":    filepath.Join(*outDir, "corrected_materials.png"),
		"heatmap":      filepath.Join(*outDir, "energy_heatmap.png"),
		"whiteFurnace": filepath.Join(*outDir, "white_furnace.png"),
		"report":       filepath.Join(*outDir, "energy_report.json"),
	}

	original := wf.RenderMaterialSheet(false, *panel, reports)
	if err := wf.SavePNG(outputs["original"], original); err != nil {
		log.Fatal(err)
	}
	corrected := wf.RenderMaterialSheet(true, *panel, reports)
	if err := wf.SavePNG(outputs["corrected"], corrected); err != nil {
		log.Fatal(err)
	}
	heatmap := wf.RenderEnergyHeatmap(reports)
	if err := wf.SavePNG(outputs["heatmap"], heatmap); err != nil {
		log.Fatal(err)
	}
	furnaceSheet := wf.RenderWhiteFurnaceSheet(reports)
	if err := wf.SavePNG(outputs["whiteFurnace"], furnaceSheet); err != nil {
		log.Fatal(err)
	}

	report := outputReport{
		ZeroDependencies: true,
		AllPass:          allPass,
		Tolerance:        0.025,
		Materials:        reports,
		Outputs:          outputs,
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	if err := os.WriteFile(outputs["report"], data, 0o644); err != nil {
		log.Fatal(err)
	}
	if err := os.MkdirAll("artifacts", 0o755); err == nil {
		_ = os.WriteFile(filepath.Join("artifacts", "energy_report.json"), data, 0o644)
	}

	if !allPass {
		log.Printf("white furnace failed for at least one material; see %s", outputs["report"])
		os.Exit(2)
	}
	log.Printf("all three materials pass white furnace; outputs in %s", *outDir)
}
