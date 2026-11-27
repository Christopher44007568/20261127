package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
)

type ModeReport struct {
	Mode               string  `json:"mode"`
	MeanLuminance      float64 `json:"meanLuminance"`
	HighFrequencyError float64 `json:"highFrequencyError"`
	SparklePixels      int     `json:"sparklePixels"`
	Samples            int     `json:"samples"`
	MeanRoughness      float64 `json:"meanEffectiveRoughness"`
}

type ContinuityStat struct {
	Radius        float64 `json:"screenRadius"`
	MeanLuminance float64 `json:"meanLuminance"`
	MeanRoughness float64 `json:"meanRoughness"`
}

type Report struct {
	TextureSize      int              `json:"textureSize"`
	Modes            []ModeReport     `json:"modes"`
	Continuity       []ContinuityStat `json:"continuity"`
	MaxLuminanceJump float64          `json:"maxLuminanceJump"`
}

func modeName(mode RoughnessMode) string {
	switch mode {
	case ModeOriginal:
		return "original_values"
	case ModeNaiveMIP:
		return "naive_mip_mean"
	default:
		return "footprint_rms_energy_compensation"
	}
}

func AnalyzeMode(scene *Scene, mode RoughnessMode, radius float64) ModeReport {
	const w, h = 500, 640
	cx, cy := 250.0, 326.0
	frame := NewFrame(w, h)
	renderSphere(frame, scene, cx, cy, radius, mode)

	report := ModeReport{Mode: modeName(mode)}
	var roughSum float64
	display := make([]float64, w*h)
	inside := make([]bool, w*h)

	for y := 0; y < h; y++ {
		sy := (float64(y) + 0.5 - cy) / radius
		for x := 0; x < w; x++ {
			sx := (float64(x) + 0.5 - cx) / radius
			q := sx*sx + sy*sy
			if q > 0.82 {
				continue
			}
			i := y*w + x
			inside[i] = true
			c := frame.Pix[i]
			l := luminance(c)
			display[i] = float64(linearToSRGB(acesTonemap(l))) / 255
			report.MeanLuminance += l
			roughSum += effectiveRoughForMetric(scene.Texture, mode, sx, sy, radius)
			report.Samples++
		}
	}
	if report.Samples > 0 {
		report.MeanLuminance /= float64(report.Samples)
		report.MeanRoughness = roughSum / float64(report.Samples)
	}

	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			i := y*w + x
			if !inside[i] {
				continue
			}
			neighbors := (display[i-1] + display[i+1] + display[i-w] + display[i+w]) * 0.25
			report.HighFrequencyError += math.Abs(display[i] - neighbors)
			if display[i] > 0.88 && display[i] > neighbors*1.45+0.08 {
				report.SparklePixels++
			}
		}
	}
	if report.Samples > 0 {
		report.HighFrequencyError /= float64(report.Samples)
	}
	return report
}

func effectiveRoughForMetric(tex *RoughnessTexture, mode RoughnessMode, sx, sy, radius float64) float64 {
	n := normalAtSlope(sx, sy)
	u, v := sphereUV(n)
	step := 1 / radius
	uxp, vxp := sphereUV(normalAtSlope(sx+step, sy))
	uxm, vxm := sphereUV(normalAtSlope(sx-step, sy))
	uyp, vyp := sphereUV(normalAtSlope(sx, sy+step))
	uym, vym := sphereUV(normalAtSlope(sx, sy-step))
	fp := FootprintFromDerivatives(wrapDelta(uxp-uxm)/2, (vxp-vxm)/2,
		wrapDelta(uyp-uym)/2, (vyp-vym)/2, float64(tex.Size))
	rs := tex.Resolve(mode, u, v, fp)
	if mode == ModeFootprint {
		dx := normalAtSlope(sx+step, sy).Sub(normalAtSlope(sx-step, sy)).Mul(0.5)
		dy := normalAtSlope(sx, sy+step).Sub(normalAtSlope(sx, sy-step)).Mul(0.5)
		rs.AddGeometryVariance((dx.Dot(dx) + dy.Dot(dy)) / 12 * 0.35)
	}
	return rs.Roughness
}

func AnalyzeContinuity(scene *Scene) ([]ContinuityStat, float64) {
	radii := []float64{224, 204, 184, 164, 144, 124, 104, 84, 66}
	stats := make([]ContinuityStat, 0, len(radii))
	var maxJump float64
	var prev float64
	for i, radius := range radii {
		const size = 500
		center := size / 2
		frame := NewFrame(size, size)
		renderSphere(frame, scene, float64(center), float64(center), radius, ModeFootprint)
		var sum, roughSum float64
		n := 0
		for y := 0; y < size; y++ {
			sy := (float64(y) + 0.5 - float64(center)) / radius
			for x := 0; x < size; x++ {
				sx := (float64(x) + 0.5 - float64(center)) / radius
				if sx*sx+sy*sy > 0.8 {
					continue
				}
				c := frame.Pix[frame.At(x, y)]
				sum += luminance(c)
				roughSum += effectiveRoughForMetric(scene.Texture, ModeFootprint, sx, sy, radius)
				n++
			}
		}
		mean := sum / math.Max(1, float64(n))
		rough := roughSum / math.Max(1, float64(n))
		stats = append(stats, ContinuityStat{Radius: radius, MeanLuminance: mean, MeanRoughness: rough})
		if i > 0 {
			jump := math.Abs(mean-prev) / math.Max(prev, 0.025)
			maxJump = math.Max(maxJump, jump)
		}
		prev = mean
	}
	return stats, maxJump
}

func BuildReport(scene *Scene) Report {
	modes := []ModeReport{
		AnalyzeMode(scene, ModeOriginal, 208),
		AnalyzeMode(scene, ModeNaiveMIP, 208),
		AnalyzeMode(scene, ModeFootprint, 208),
	}
	continuity, jump := AnalyzeContinuity(scene)
	return Report{TextureSize: scene.Texture.Size, Modes: modes,
		Continuity: continuity, MaxLuminanceJump: jump}
}

func WriteReport(path string, report Report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
