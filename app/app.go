// Package app orchestrates scene rendering, image output and the machine
// readable summary. It is invoked by both main entry points.
package app

import (
	"encoding/json"
	"os"
	"path/filepath"

	"coverlab/cover"
	"coverlab/scenes"
)

// Method names shared across artifacts.
const (
	mAnalytic = "analytic"
	mSS2      = "ss2"
	mSS3      = "ss3"
	mBlur     = "blur"
)

// BlurSigma is the radius of the demonstration "cover it with blur" baseline.
const BlurSigma = 1.15

// MethodStat records error statistics for one method vs analytic alpha.
type MethodStat struct {
	Method string `json:"method"`
	cover.ErrStats
}

// SceneSummary is the per-scene section of summary.json.
type SceneSummary struct {
	Name    string                `json:"name"`
	Width   int                   `json:"width"`
	Height  int                   `json:"height"`
	Methods map[string]MethodStat `json:"methods"`
}

// Point is one offset/coverage sample.
type Point struct {
	Dy       float64 `json:"dy"`
	Coverage float64 `json:"coverage"`
	Gray255  float64 `json:"gray255"`
}

// SweepSummary documents sub-pixel brightness continuity.
type SweepSummary struct {
	PixelX   int                `json:"pixelX"`
	PixelY   int                `json:"pixelY"`
	Step     float64            `json:"step"`
	Analytic []Point            `json:"analytic"`
	SS2      []Point            `json:"ss2"`
	SS3      []Point            `json:"ss3"`
	MaxJump  map[string]float64 `json:"maxAdjacentJump"`
}

// Summary is the top-level summary.json document.
type Summary struct {
	BlurSigma float64                      `json:"blurSigma"`
	Scenes    []SceneSummary               `json:"scenes"`
	Sweep     SweepSummary                 `json:"sweep"`
	Stains    map[string]cover.StainReport `json:"stains"`
	Guard     GuardReport                  `json:"guardPixel"`
	Checks    map[string]bool              `json:"checks"`
}

// GuardReport records the pure-white gap pixel check.
type GuardReport struct {
	X        int       `json:"x"`
	Y        int       `json:"y"`
	Analytic cover.RGB `json:"analyticRGB255"`
	SS2      cover.RGB `json:"ss2RGB255"`
	SS3      cover.RGB `json:"ss3RGB255"`
	Blur     cover.RGB `json:"blurRGB255"`
}

// Run generates every artifact into outDir and writes summary.json.
func Run(outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	sum := Summary{
		BlurSigma: BlurSigma,
		Stains:    map[string]cover.StainReport{},
		Checks:    map[string]bool{},
	}

	for _, scene := range scenes.All() {
		st, err := renderScene(outDir, scene)
		if err != nil {
			return err
		}
		sum.Scenes = append(sum.Scenes, st)
	}

	sweep, err := renderSweep(outDir)
	if err != nil {
		return err
	}
	sum.Sweep = sweep

	stains, guard, checks, err := renderStain(outDir)
	if err != nil {
		return err
	}
	sum.Stains = stains
	sum.Guard = guard
	for k, v := range checks {
		sum.Checks[k] = v
	}
	sum.Checks["analytic_continuous_0p1"] = sweep.MaxJump[mAnalytic] <= 0.1+1e-9
	sum.Checks["ss2_shows_quantization"] = sweep.MaxJump[mSS2] >= 0.24
	sum.Checks["ss3_shows_quantization"] = sweep.MaxJump[mSS3] >= 1.0/9.0-1e-9

	b, err := json.MarshalIndent(sum, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "summary.json"), b, 0o644)
}
