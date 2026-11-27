package main

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
)

const defaultBudget = 128

type metricEntry struct {
	Mode              string  `json:"mode"`
	MaxTapsBudget     int     `json:"max_taps_budget"`
	BoardPixels       int     `json:"board_pixels"`
	RMSE              float64 `json:"rmse"`
	MAE               float64 `json:"mae"`
	MaxLumaError      float64 `json:"max_luma_error"`
	PSNR              float64 `json:"psnr_db"`
	LongAxisVariation float64 `json:"long_axis_variation"`
	CrossAxisEnergy   float64 `json:"cross_axis_energy"`
	DirectionPixels   int     `json:"direction_pixels"`
	TotalTaps         int     `json:"total_taps"`
	MaxProbes         int     `json:"max_probes"`
	MaxAnisotropy     float64 `json:"max_anisotropy"`
	MaxMajorTexels    float64 `json:"max_major_texels"`
	MaxMinorTexels    float64 `json:"max_minor_texels"`
	DegradedPixels    int     `json:"degraded_pixels,omitempty"`
}

type metricsFile struct {
	Anisotropic   metricEntry   `json:"anisotropic"`
	BudgetLadder  []metricEntry `json:"budget_ladder"`
	DefaultBudget int           `json:"default_budget"`
	FixedFour     metricEntry   `json:"fixed_four"`
	Height        int           `json:"height"`
	Reference     metricEntry   `json:"reference"`
	TextureSize   int           `json:"texture_size"`
	Width         int           `json:"width"`
}

func metricFromStats(name string, stats *RenderStats, err ErrorStats) metricEntry {
	return metricEntry{
		Mode:              name,
		MaxTapsBudget:     stats.Budget,
		BoardPixels:       stats.BoardPixels,
		RMSE:              err.RMSE,
		MAE:               err.MAE,
		MaxLumaError:      err.MaxLumaError,
		PSNR:              err.PSNR,
		LongAxisVariation: stats.LongAxisVariation,
		CrossAxisEnergy:   stats.CrossAxisEnergy,
		DirectionPixels:   stats.DirectionPixels,
		TotalTaps:         stats.TotalTaps,
		MaxProbes:         stats.MaxProbes,
		MaxAnisotropy:     stats.MaxAnisotropy,
		MaxMajorTexels:    stats.MaxMajorTexels,
		MaxMinorTexels:    stats.MaxMinorTexels,
		DegradedPixels:    stats.DegradedPixels,
	}
}

func main() {
	artifactDir := "artifacts"
	if err := os.MkdirAll(artifactDir, 0o755); err != nil {
		fatal(err)
	}

	fmt.Println("building procedural texture and mip chain")
	pattern := NewStripePattern()
	tex := NewTexture(pattern, textureSize)

	fmt.Println("computing screen-ray differential footprints")
	scene := NewScene()
	fps := BuildFootprints(scene)
	fmt.Printf("board pixels: %d\n", len(fps.Pixels))

	fmt.Println("rendering elliptical reference")
	refImg, refStats := renderBoard(scene, tex, pattern, fps, ModeReference, 0)
	addDirectionalMetrics(refStats, refImg, fps, scene)

	fmt.Println("rendering fixed-four baseline")
	fixedImg, fixedStats := renderBoard(scene, tex, pattern, fps, ModeFixed, 4)
	addDirectionalMetrics(fixedStats, fixedImg, fps, scene)
	fixedErr := compareImages(fixedImg, refImg, fps)

	fmt.Println("rendering anisotropic mode")
	anisoImg, anisoStats := renderBoard(scene, tex, pattern, fps, ModeAniso, defaultBudget)
	addDirectionalMetrics(anisoStats, anisoImg, fps, scene)
	anisoErr := compareImages(anisoImg, refImg, fps)

	budgets := []int{4, 8, 16, 32, 64, 128}
	budgetImages := make(map[int]*image.RGBA)
	budgetEntries := make([]metricEntry, 0, len(budgets))
	for _, budget := range budgets {
		fmt.Printf("rendering budget %d\n", budget)
		var img *image.RGBA
		var stats *RenderStats
		if budget == defaultBudget {
			img, stats = anisoImg, anisoStats
		} else {
			img, stats = renderBoard(scene, tex, pattern, fps, ModeAniso, budget)
			addDirectionalMetrics(stats, img, fps, scene)
		}
		errStats := compareImages(img, refImg, fps)
		budgetImages[budget] = img
		budgetEntries = append(budgetEntries, metricFromStats("anisotropic", stats, errStats))
	}

	anisoHeat := errorHeatmap(scene, anisoImg, refImg, fps, 20)
	fixedHeat := errorHeatmap(scene, fixedImg, refImg, fps, 20)
	overlay := renderFootprintOverlay(scene, fps, anisoImg)

	fmt.Println("writing image artifacts")
	outputs := map[string]*image.RGBA{
		"board_anisotropic.png":        anisoImg,
		"board_fixed_four.png":         fixedImg,
		"board_reference.png":          refImg,
		"error_heatmap.png":            anisoHeat,
		"fixed_four_error_heatmap.png": fixedHeat,
		"footprint_overlay.png":        overlay,
		"tilted_texture.png":           textureImage(tex),
		"budget_degradation.png":       makeBudgetStrip(budgetImages, budgets),
		"comparison.png":               makeComparison(fixedImg, anisoImg, refImg, anisoHeat),
		"detail_moire.png":             makeDetail(fixedImg, anisoImg, fixedHeat),
	}
	for name, img := range outputs {
		if err := savePNG(filepath.Join(artifactDir, name), img); err != nil {
			fatal(err)
		}
	}

	metrics := metricsFile{
		Anisotropic:   metricFromStats("anisotropic", anisoStats, anisoErr),
		BudgetLadder:  budgetEntries,
		DefaultBudget: defaultBudget,
		FixedFour:     metricFromStats("fixed_four", fixedStats, fixedErr),
		Height:        scene.Height,
		Reference:     metricFromStats("elliptical_reference", refStats, ErrorStats{}),
		TextureSize:   textureSize,
		Width:         scene.Width,
	}
	metrics.Reference.MaxTapsBudget = 0
	metrics.Reference.TotalTaps = 0
	if err := writeJSON(filepath.Join(artifactDir, "metrics.json"), &metrics); err != nil {
		fatal(err)
	}
	fmt.Printf("done: aniso RMSE %.6f, fixed-four RMSE %.6f\n", anisoErr.RMSE, fixedErr.RMSE)
}

func writeJSON(path string, v any) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	return f.Sync()
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
