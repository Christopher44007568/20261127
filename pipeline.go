package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type RunOptions struct {
	OutDir      string
	DepthWidth  int
	DepthHeight int
	VoxelSize   float64
}

type CheckReport struct {
	ID        string  `json:"id"`
	Label     string  `json:"label"`
	Pass      bool    `json:"pass"`
	Value     float64 `json:"value"`
	Threshold float64 `json:"threshold"`
	Unit      string  `json:"unit"`
	Detail    string  `json:"detail,omitempty"`
}

type SceneReport struct {
	ID             string            `json:"id"`
	Title          string            `json:"title"`
	ViewCount      int               `json:"view_count"`
	HolePixels     int               `json:"hole_pixels"`
	VoxelCount     int               `json:"voxel_count"`
	VoxelSize      float64           `json:"voxel_size_m"`
	Truncation     float64           `json:"truncation_m"`
	UpdatedVoxels  int               `json:"updated_voxel_updates"`
	MaxWeight      float32           `json:"max_weight"`
	ThresholdCells int               `json:"threshold_cells"`
	MeshVertices   int               `json:"mesh_vertices"`
	MeshTriangles  int               `json:"mesh_triangles"`
	RawCloud       ErrorStats        `json:"raw_cloud_distance"`
	Contour        ErrorStats        `json:"slice_contour_distance"`
	Mesh           ErrorStats        `json:"mesh_vertex_distance"`
	Coverage       CoverageCheck     `json:"coverage"`
	Thin           *ThinCheck        `json:"thin,omitempty"`
	Corner         *CornerCheck      `json:"corner,omitempty"`
	CoarseControl  *CoarseControl    `json:"coarse_control,omitempty"`
	Checks         []CheckReport     `json:"checks"`
	Outputs        map[string]string `json:"outputs"`
}

type CoarseControl struct {
	VoxelSize  float64      `json:"voxel_size_m"`
	Truncation float64      `json:"truncation_m"`
	Contour    ErrorStats   `json:"slice_contour_distance"`
	Thin       *ThinCheck   `json:"thin,omitempty"`
	Corner     *CornerCheck `json:"corner,omitempty"`
}

type Report struct {
	Schema      string        `json:"schema"`
	GeneratedAt string        `json:"generated_at"`
	GoVersion   string        `json:"go_version"`
	Options     RunOptions    `json:"options"`
	Pass        bool          `json:"pass"`
	Scenes      []SceneReport `json:"scenes"`
}

func DefaultRunOptions() RunOptions {
	return RunOptions{
		OutDir:      "out",
		DepthWidth:  256,
		DepthHeight: 192,
		VoxelSize:   0.015,
	}
}

func addCheck(report *SceneReport, id, label string, value, threshold float64, unit string, pass bool) {
	report.Checks = append(report.Checks, CheckReport{
		ID: id, Label: label, Pass: pass, Value: value, Threshold: threshold, Unit: unit,
	})
}

func Run(opts RunOptions) (*Report, error) {
	if opts.OutDir == "" {
		opts.OutDir = "out"
	}
	if opts.DepthWidth <= 0 {
		opts.DepthWidth = 256
	}
	if opts.DepthHeight <= 0 {
		opts.DepthHeight = 192
	}
	if opts.VoxelSize <= 0 {
		opts.VoxelSize = 0.015
	}

	report := &Report{
		Schema:      "tsdf-lab/v1",
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		GoVersion:   runtime.Version(),
		Options:     opts,
		Pass:        true,
	}

	gridMin := V3(-1.35, -1.15, -0.15)
	cfg := FusionConfig{
		VoxelSize:  opts.VoxelSize,
		NX:         180,
		NY:         154,
		NZ:         94,
		Min:        gridMin,
		Truncation: opts.VoxelSize * 2.5,
		MaxWeight:  12,
		MinWeight:  0.25,
		UseRobust:  false,
	}

	for _, scene := range AllScenes() {
		sr, err := runScene(opts, cfg, scene)
		if err != nil {
			return nil, err
		}
		report.Scenes = append(report.Scenes, *sr)
		for _, chk := range sr.Checks {
			if !chk.Pass {
				report.Pass = false
			}
		}
	}

	if err := writeReport(filepath.Join(opts.OutDir, "report.json"), report); err != nil {
		return nil, err
	}
	return report, nil
}

func writeReport(path string, report *Report) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
