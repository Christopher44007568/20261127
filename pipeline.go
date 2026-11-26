package main

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"time"
)

type RunConfig struct {
	Input  string
	Out    string
	Cell   float64
	Seed   int64
	Yaw    float64
	Pitch  float64
	Strict bool
}

type Check struct {
	Name   string      `json:"name"`
	Pass   bool        `json:"pass"`
	Value  interface{} `json:"value,omitempty"`
	Limit  string      `json:"limit,omitempty"`
	Detail string      `json:"detail,omitempty"`
}

type RunReport struct {
	Input               string                 `json:"input"`
	Demo                bool                   `json:"demo"`
	RawPoints           int                    `json:"raw_points"`
	KeptPoints          int                    `json:"kept_points"`
	RemovedOutliers     int                    `json:"removed_outliers"`
	InjectedOutliers    int                    `json:"injected_outliers,omitempty"`
	OutlierRecall       float64                `json:"outlier_recall,omitempty"`
	SurfaceFalseRemoved int                    `json:"surface_false_removed,omitempty"`
	OutlierThreshold    float64                `json:"outlier_threshold"`
	PointSpacing        float64                `json:"point_spacing"`
	Orientation         OrientationReport      `json:"orientation"`
	Grid                map[string]interface{} `json:"grid"`
	Mesh                map[string]interface{} `json:"mesh"`
	Geometry            *GeometryScore         `json:"geometry,omitempty"`
	Curvature           *ErrorStats            `json:"curvature,omitempty"`
	Holes               []HoleScore            `json:"holes,omitempty"`
	Checks              []Check                `json:"checks"`
	Outputs             map[string]string      `json:"outputs"`
	ElapsedMS           int64                  `json:"elapsed_ms"`
}

func Run(cfg RunConfig) (*RunReport, error) {
	started := time.Now()
	if cfg.Out == "" {
		cfg.Out = "out"
	}
	if cfg.Seed == 0 {
		cfg.Seed = 20261127
	}
	if cfg.Yaw == 0 && cfg.Pitch == 0 {
		cfg.Yaw = -0.68
		cfg.Pitch = 0.58
	}
	report := &RunReport{Input: cfg.Input, Outputs: map[string]string{}}

	var raw *Cloud
	var scene *DemoScene
	var err error
	if cfg.Input != "" {
		raw, err = ReadCloud(cfg.Input)
		if err != nil {
			return nil, err
		}
	} else {
		scene = NewDemoScene(cfg.Seed)
		raw = scene.Cloud
		report.Demo = true
	}
	report.RawPoints = len(raw.P)

	filterReport := FilterOutliers(raw.P, 12, 3.0)
	report.RemovedOutliers = filterReport.Removed
	report.KeptPoints = len(filterReport.Keep)
	report.OutlierThreshold = filterReport.Threshold
	report.PointSpacing = filterReport.Spacing

	removedSet := map[int]bool{}
	keepSet := map[int]bool{}
	for i := range raw.P {
		removedSet[i] = true
	}
	filtered := &Cloud{HasN: true}
	for _, oldID := range filterReport.Keep {
		keepSet[oldID] = true
		removedSet[oldID] = false
		filtered.P = append(filtered.P, raw.P[oldID])
	}
	if scene != nil {
		outlierTotal := len(raw.P) - scene.OutlierStart
		removedOutlier := 0
		surfaceRemoved := 0
		for id, removed := range removedSet {
			if !removed {
				continue
			}
			if id >= scene.OutlierStart {
				removedOutlier++
			} else {
				surfaceRemoved++
			}
		}
		report.InjectedOutliers = outlierTotal
		report.OutlierRecall = float64(removedOutlier) / float64(maxInt(1, outlierTotal))
		report.SurfaceFalseRemoved = surfaceRemoved
	}

	normals, _ := EstimateNormals(filtered.P, 26)
	orientation := OrientNormalsMST(filtered.P, normals, 10, V3{0.08, -0.18, 1})
	filtered.N = normals
	report.Orientation = orientation

	opts := DefaultImplicitOptions(filterReport.Spacing)
	if cfg.Cell > 0 {
		opts.Cell = cfg.Cell
		opts.FitRadius = cfg.Cell * 2.9
		opts.TrimRadius = cfg.Cell * 1.25
	}
	field := BuildImplicitField(filtered.P, normals, filterReport.Spacing, opts)
	mesh := ExtractSurface(field)
	mesh.ComputeNormals()
	ApplyFieldNormals(mesh, field, opts.FitRadius*1.05)
	boundaryMask := BoundaryVertexMask(mesh, 3)
	components, largestFraction := MeshComponentStats(mesh)

	report.Grid = map[string]interface{}{
		"nx": field.nx, "ny": field.ny, "nz": field.nz,
		"cell": opts.Cell, "fit_radius": opts.FitRadius,
		"trim_radius":   opts.TrimRadius,
		"valid_corners": countValid(field.valid),
	}
	report.Mesh = map[string]interface{}{
		"vertices": len(mesh.P), "triangles": len(mesh.Tri),
		"boundary_edges": mesh.BoundaryEdgeCount(),
		"area":           MeshArea(mesh), "components": components,
		"largest_component_fraction": largestFraction,
	}

	viewBox := NewBox()
	if scene != nil {
		viewBox.Min = V3{scene.XMin - 0.18, scene.YMin - 0.16, -0.46}
		viewBox.Max = V3{scene.XMax + 0.18, scene.YMax + 0.16, 0.46}
	} else {
		for _, p := range filtered.P {
			viewBox.Add(p)
		}
		size := viewBox.Size()
		pad := math.Max(math.Max(size.X, size.Y), size.Z) * 0.10
		viewBox = viewBox.Expanded(pad)
	}
	cam := FitCamera(viewBox, panelW, panelH, cfg.Yaw, cfg.Pitch, 0.80)

	curvStats, err := renderPanels(cfg.Out, raw, removedSet, filtered, mesh, field, scene, boundaryMask, cam, report)
	if err != nil {
		return nil, err
	}
	report.Curvature = curvStats
	if err := writeCloudOutputs(cfg.Out, raw, filtered, mesh); err != nil {
		return nil, err
	}

	if scene != nil {
		geom := ScoreDemoGeometry(mesh, scene, boundaryMask)
		report.Geometry = &geom
		holes := ScoreHoles(mesh, scene.Holes)
		report.Holes = holes
		report.Checks = buildDemoChecks(report, geom, holes, components, largestFraction)
	} else {
		report.Checks = buildGenericChecks(report, components, largestFraction)
	}

	report.ElapsedMS = time.Since(started).Milliseconds()
	if err := WriteReport(filepath.Join(cfg.Out, "report.json"), report); err != nil {
		return nil, err
	}
	if cfg.Strict {
		for _, chk := range report.Checks {
			if !chk.Pass {
				return report, fmt.Errorf("严格检查失败: %s (%s)", chk.Name, chk.Detail)
			}
		}
	}
	return report, nil
}

func countValid(vs []bool) int {
	n := 0
	for _, v := range vs {
		if v {
			n++
		}
	}
	return n
}

func writeCloudOutputs(out string, raw, filtered *Cloud, mesh *Mesh) error {
	if err := os.MkdirAll(filepath.Join(out, "cloud"), 0o755); err != nil {
		return err
	}
	if err := WriteCloudPLY(filepath.Join(out, "cloud", "raw_input.ply"), raw); err != nil {
		return err
	}
	if err := WriteCloudPLY(filepath.Join(out, "cloud", "filtered_oriented.ply"), filtered); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(out, "mesh"), 0o755); err != nil {
		return err
	}
	if err := WriteMeshPLY(filepath.Join(out, "mesh", "reconstructed.ply"), mesh, len(mesh.C) == len(mesh.P)); err != nil {
		return err
	}
	return WriteMeshOBJ(filepath.Join(out, "mesh", "reconstructed.obj"), mesh)
}

func WriteReport(path string, report *RunReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
