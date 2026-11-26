package bloom

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
)

type StageReport struct {
	ID           string              `json:"id"`
	Kind         string              `json:"kind"`
	Format       FormatID            `json:"format"`
	Quantization *QuantizationReport `json:"quantization"`
}

type PPMReloadReport struct {
	File          string  `json:"file"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	MaxVal        int     `json:"maxval"`
	SHA256        string  `json:"sha256"`
	MismatchCount uint64  `json:"mismatch_count"`
	MaxCodeError  float64 `json:"max_code_error"`
	MeanCodeError float64 `json:"mean_code_error"`
}

type FormatArtifactReport struct {
	Format               FormatID        `json:"format"`
	Encoding             string          `json:"encoding"`
	Directory            string          `json:"directory"`
	HDRFile              string          `json:"hdr_file"`
	ComparisonPNG        string          `json:"comparison_png"`
	ErrorHeatmapPNG      string          `json:"error_heatmap_png"`
	ReloadPPM            string          `json:"reload_ppm"`
	ErrorHeatmapScale    float64         `json:"error_heatmap_normalization"`
	FinalError           ErrorStats      `json:"final_error"`
	RangeCache           RangeCache      `json:"range_cache"`
	ClampedChannelValues uint64          `json:"clamped_channel_values"`
	PPMReload            PPMReloadReport `json:"ppm_reload"`
	Stages               []StageReport   `json:"stages"`
}

func WriteFormatArtifacts(root string, render *FormatRender) (*FormatArtifactReport, error) {
	if render == nil || render.Execution == nil || render.Execution.Final == nil {
		return nil, fmt.Errorf("nil or incomplete render")
	}
	info, err := FormatInfoFor(render.Format)
	if err != nil {
		return nil, err
	}
	dir := filepath.Join(root, string(render.Format))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	hdrPath := filepath.Join(dir, "hdr_compare.pfm")
	pngPath := filepath.Join(dir, "hdr_compare.png")
	heatPath := filepath.Join(dir, "quantization_error.png")
	ppmPath := filepath.Join(dir, "reload.ppm")
	if err := WritePFM(hdrPath, render.Execution.Final.Linear); err != nil {
		return nil, fmt.Errorf("write PFM: %w", err)
	}
	if err := WriteComparisonPNG(pngPath, render.Reference, render.Execution.Final.Linear); err != nil {
		return nil, fmt.Errorf("write comparison PNG: %w", err)
	}
	normalization := render.ErrorNormalization
	if normalization <= 0 {
		normalization = 1
	}
	if err := WriteQuantizationHeatmap(heatPath, render.ErrorImage, normalization); err != nil {
		return nil, fmt.Errorf("write heatmap: %w", err)
	}
	display, err := NewImage(render.Execution.Final.Linear.Width, render.Execution.Final.Linear.Height)
	if err != nil {
		return nil, err
	}
	for i, c := range render.Execution.Final.Linear.Pixels {
		display.Pixels[i] = ToneMapReinhard(c)
	}
	if err := WritePPM16(ppmPath, display); err != nil {
		return nil, fmt.Errorf("write PPM: %w", err)
	}
	reload, err := validatePPMReload(ppmPath, display)
	if err != nil {
		return nil, fmt.Errorf("reload PPM: %w", err)
	}
	stages := make([]StageReport, 0, len(render.Execution.Order))
	for _, id := range render.Execution.Order {
		out := render.Execution.Nodes[id]
		stages = append(stages, StageReport{
			ID: id, Kind: out.Kind, Format: out.Format, Quantization: out.Quantization,
		})
	}
	report := &FormatArtifactReport{
		Format: render.Format, Encoding: info.Encoding, Directory: string(render.Format),
		HDRFile: "hdr_compare.pfm", ComparisonPNG: "hdr_compare.png",
		ErrorHeatmapPNG: "quantization_error.png", ReloadPPM: "reload.ppm",
		ErrorHeatmapScale: normalization, FinalError: render.FinalError,
		RangeCache:           render.Execution.Range,
		ClampedChannelValues: render.Execution.Range.ClampCount(),
		PPMReload:            reload, Stages: stages,
	}
	return report, writeJSON(filepath.Join(dir, "stats.json"), report)
}

func validatePPMReload(path string, expected *Image) (PPMReloadReport, error) {
	loaded, err := ReadPPM(path)
	if err != nil {
		return PPMReloadReport{}, err
	}
	if loaded.Width != expected.Width || loaded.Height != expected.Height {
		return PPMReloadReport{}, fmt.Errorf("loaded PPM is %dx%d, want %dx%d",
			loaded.Width, loaded.Height, expected.Width, expected.Height)
	}
	if len(loaded.Pixels) != len(expected.Pixels) {
		return PPMReloadReport{}, fmt.Errorf("loaded PPM has %d pixels, want %d",
			len(loaded.Pixels), len(expected.Pixels))
	}
	var mismatches uint64
	var maxError, sumError float64
	maxVal := float64(loaded.MaxVal)
	for i := range expected.Pixels {
		want := expected.Pixels[i]
		got := loaded.Pixels[i]
		for _, pair := range [][2]float64{
			{float64(PPM16Code(want.R)), math.Round(got.R * maxVal)},
			{float64(PPM16Code(want.G)), math.Round(got.G * maxVal)},
			{float64(PPM16Code(want.B)), math.Round(got.B * maxVal)},
		} {
			d := math.Abs(pair[0] - pair[1])
			if d != 0 {
				mismatches++
			}
			sumError += d
			if d > maxError {
				maxError = d
			}
		}
	}
	hash, err := fileSHA256(path)
	if err != nil {
		return PPMReloadReport{}, err
	}
	samples := float64(len(expected.Pixels) * 3)
	return PPMReloadReport{
		File: "reload.ppm", Width: loaded.Width, Height: loaded.Height,
		MaxVal: loaded.MaxVal, SHA256: hash, MismatchCount: mismatches,
		MaxCodeError: maxError, MeanCodeError: sumError / samples,
	}, nil
}

func fileSHA256(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

func writeJSON(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
