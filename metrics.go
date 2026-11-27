package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
)

type ImageMetrics struct {
	File            string  `json:"file"`
	Width           int     `json:"width"`
	Height          int     `json:"height"`
	Coverage        float64 `json:"coverage"`
	MeanLinearLuma  float64 `json:"mean_linear_luma"`
	MaxLinearLuma   float64 `json:"max_linear_luma"`
	OverOneFraction float64 `json:"over_one_fraction"`
}

type MetricsReport struct {
	WorldScale          float64                 `json:"world_scale"`
	Hardness            float64                 `json:"hardness"`
	WeightSumMaxError   float64                 `json:"weight_sum_max_error"`
	WhiteSeamPeak       float64                 `json:"white_texture_seam_peak"`
	GravityUVScale      float64                 `json:"gravity_uv_scale"`
	ProjectionWeighting string                  `json:"projection_weighting"`
	Images              map[string]ImageMetrics `json:"images"`
}

func luma(c Vec3) float64 {
	return 0.2126*c.X + 0.7152*c.Y + 0.0722*c.Z
}

func MeasureFrame(frame *Frame) ImageMetrics {
	metrics := ImageMetrics{Width: frame.Width, Height: frame.Height}
	var lumaSum float64
	var overOne int
	for i, color := range frame.Color {
		if frame.Depth[i] <= 0 {
			continue
		}
		metrics.Coverage++
		y := luma(color)
		lumaSum += y
		metrics.MaxLinearLuma = math.Max(metrics.MaxLinearLuma, y)
		if y > 1 {
			overOne++
		}
	}
	if metrics.Coverage > 0 {
		metrics.MeanLinearLuma = lumaSum / metrics.Coverage
		metrics.OverOneFraction = float64(overOne) / metrics.Coverage
		metrics.Coverage /= float64(frame.Width * frame.Height)
	}
	return metrics
}

func CoreInvariantMetrics(scale, hardness float64) (weightError, whitePeak float64) {
	for i := 0; i < 256; i++ {
		angleA := float64(i) * 2.399963229728653
		angleB := float64(i*7+3) * 1.41421356237
		normal := V(
			math.Cos(angleA)*math.Cos(angleB),
			math.Sin(angleA)*math.Cos(angleB),
			math.Sin(angleB),
		).Normalize()
		weights := NormalWeights(normal, hardness)
		sum := weights.X + weights.Y + weights.Z
		weightError = math.Max(weightError, math.Abs(sum-1))

		tri := NewTriplanar(ConstantTexture{Color: V(1, 1, 1)}, scale, hardness)
		color := tri.Sample(
			V(float64(i)*0.13, -2+float64(i)*0.07, 4-float64(i)*0.11),
			normal,
		)
		whitePeak = math.Max(whitePeak, math.Max(color.X, math.Max(color.Y, color.Z)))
	}
	return weightError, whitePeak
}

func WriteJSON(path string, value any) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return os.WriteFile(path, data, 0o644)
}
