// Package pipeline wires the material model, metrics, charts, and report files.
package pipeline

import (
	"path/filepath"

	"materiallod/material"
)

const (
	CrossfadeWidth = 0.10
	ScanStep       = 0.01
	JumpThreshold  = 0.025
	DiffFullScale  = 0.28
)

type Options struct {
	Out string
}

func DefaultOptions() Options {
	return Options{Out: filepath.Join("artifacts", "material-lod")}
}

type Artifacts struct {
	Materials  []material.Material
	Spheres    map[string][3]*material.SphereBuffer
	Brightness []material.BrightnessStats
	Sweeps     []material.ViewSample
	Continuity material.ContinuityReport
}

// Generate computes every artifact and writes CSV, JSON, Markdown, and PNGs.
func Generate(opts Options) (*Artifacts, error) {
	if opts.Out == "" {
		opts = DefaultOptions()
	}
	if err := ensureDir(opts.Out); err != nil {
		return nil, err
	}

	materials := material.DefaultMaterials()
	art := &Artifacts{
		Materials:  materials,
		Spheres:    map[string][3]*material.SphereBuffer{},
		Brightness: []material.BrightnessStats{},
		Sweeps:     []material.ViewSample{},
	}

	if err := writeJSON(outputPath(opts.Out, "materials.json"), materials); err != nil {
		return nil, err
	}

	for _, mat := range materials {
		low := material.RenderSphere(mat, material.TierLow)
		medium := material.RenderSphere(mat, material.TierMedium)
		high := material.RenderSphere(mat, material.TierHigh)
		art.Spheres[mat.ID] = [3]*material.SphereBuffer{low, medium, high}

		art.Brightness = append(art.Brightness,
			material.CompareBuffers(mat.ID, "low_vs_high", low, high),
			material.CompareBuffers(mat.ID, "medium_vs_high", medium, high),
		)
		for _, tier := range []material.Tier{material.TierLow, material.TierMedium, material.TierHigh} {
			art.Sweeps = append(art.Sweeps, material.ViewSweep(mat, tier)...)
		}
	}

	if err := writeBrightnessCSV(opts.Out, art.Brightness); err != nil {
		return nil, err
	}
	if err := writeJSON(outputPath(opts.Out, "brightness_stats.json"), art.Brightness); err != nil {
		return nil, err
	}
	if err := writeViewSweepCSV(opts.Out, art.Sweeps); err != nil {
		return nil, err
	}

	art.Continuity = material.ScanContinuity(materials, ScanStep, CrossfadeWidth, JumpThreshold)
	if err := writeJSON(outputPath(opts.Out, "continuity_report.json"), art.Continuity); err != nil {
		return nil, err
	}
	if err := writeBoundaryCSV(opts.Out, art.Continuity.Rows); err != nil {
		return nil, err
	}

	if err := renderImages(opts.Out, art); err != nil {
		return nil, err
	}
	if err := writeReportMarkdown(opts.Out, art); err != nil {
		return nil, err
	}
	return art, nil
}
