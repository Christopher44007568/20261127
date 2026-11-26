package qem

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
)

type RunConfig struct {
	ShadedTile      int     `json:"shaded_tile"`
	SilhouetteTile  int     `json:"silhouette_tile"`
	HighlightSize   int     `json:"highlight_size"`
	ShadedViews     int     `json:"shaded_views"`
	SilhouetteViews int     `json:"silhouette_views"`
	InitialPitch    float64 `json:"initial_pitch"`
	SaveOBJ         bool    `json:"save_obj"`
}

func DefaultRunConfig() RunConfig {
	return RunConfig{
		ShadedTile:      256,
		SilhouetteTile:  192,
		HighlightSize:   520,
		ShadedViews:     4,
		SilhouetteViews: 8,
		InitialPitch:    0.16,
		SaveOBJ:         true,
	}
}

type LevelReport struct {
	Index             int                `json:"index"`
	Prefix            string             `json:"prefix"`
	RequestedRatio    float64            `json:"requested_ratio"`
	TargetTriangles   int                `json:"target_triangles"`
	StepsThisLevel    int                `json:"steps_this_level"`
	TotalCollapses    int                `json:"total_collapses"`
	ActiveTriangles   int                `json:"active_triangles"`
	ActiveVertices    int                `json:"active_vertices"`
	TriangleRatio     float64            `json:"triangle_ratio"`
	PhysicalVolume    float64            `json:"physical_volume"`
	VolumeRatio       float64            `json:"volume_ratio"`
	StopReason        string             `json:"stop_reason,omitempty"`
	Verification      VerificationReport `json:"verification"`
	SilhouetteSummary SilhouetteSummary  `json:"silhouette_summary"`
	SilhouetteMetrics []SilhouetteMetric `json:"silhouette_metrics"`
	Files             map[string]string  `json:"files"`
}

type CaseReport struct {
	Schema           string         `json:"schema"`
	Name             string         `json:"name"`
	Options          Options        `json:"options"`
	Config           RunConfig      `json:"config"`
	InitialTriangles int            `json:"initial_triangles"`
	InitialVertices  int            `json:"initial_vertices"`
	InitialVolume    float64        `json:"initial_volume"`
	ProtectedEdges   map[string]int `json:"protected_edges"`
	Levels           []LevelReport  `json:"levels"`
	UndoExact        bool           `json:"undo_exact"`
	HistoryFile      string         `json:"history_file,omitempty"`
}

// RunCase simplifies one mesh at each ratio and writes all requested images.
// The input mesh is mutated; an internal clone supplies the fixed original
// silhouette frame and verification baseline.
func RunCase(mesh *Mesh, name, outputDir string, ratios []float64, opt Options, cfg RunConfig) (*CaseReport, error) {
	if len(ratios) == 0 {
		ratios = []float64{1, 0.75, 0.5, 0.35, 0.22}
	}
	if cfg.ShadedTile <= 0 {
		cfg = DefaultRunConfig()
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "ppm"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "silhouette"), 0o755); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Join(outputDir, "highlights"), 0o755); err != nil {
		return nil, err
	}
	if cfg.SaveOBJ {
		if err := os.MkdirAll(filepath.Join(outputDir, "obj"), 0o755); err != nil {
			return nil, err
		}
	}

	original := mesh.Clone()
	sim, err := NewSimplifier(mesh, opt)
	if err != nil {
		return nil, err
	}
	initialTris := mesh.ActiveTriangleCount()
	initialVerts := mesh.ActiveVertexCount()
	report := &CaseReport{
		Schema:           "qemsimplify/v1",
		Name:             name,
		Options:          opt,
		Config:           cfg,
		InitialTriangles: initialTris,
		InitialVertices:  initialVerts,
		InitialVolume:    sim.initialVolume,
		ProtectedEdges:   sim.ProtectedEdgeCounts(),
	}

	stopped := false
	for levelIndex, ratio := range ratios {
		prefix := fmt.Sprintf("level-%02d", levelIndex)
		target := int(math.Round(float64(initialTris) * ratio))
		if target < 1 {
			target = 1
		}
		before := len(sim.Records)
		stopReason := ""
		if !stopped {
			_, collapseErr := sim.CollapseToTriangles(target)
			if collapseErr != nil {
				if !errors.Is(collapseErr, ErrNoCollapsibleEdge) {
					return nil, collapseErr
				}
				stopReason = collapseErr.Error()
				stopped = true
			}
		}

		level := LevelReport{
			Index:           levelIndex,
			Prefix:          prefix,
			RequestedRatio:  ratio,
			TargetTriangles: target,
			StepsThisLevel:  len(sim.Records) - before,
			TotalCollapses:  len(sim.Records),
			ActiveTriangles: mesh.ActiveTriangleCount(),
			ActiveVertices:  mesh.ActiveVertexCount(),
			TriangleRatio:   float64(mesh.ActiveTriangleCount()) / float64(initialTris),
			PhysicalVolume:  sim.volumeSign * sim.volumeSigned6 / 6,
			StopReason:      stopReason,
			Verification:    sim.Verify(original),
			Files:           map[string]string{},
		}
		if sim.initialVolume > 1e-12*sim.scale*sim.scale*sim.scale {
			level.VolumeRatio = level.PhysicalVolume / sim.initialVolume
		}

		shadedPath, err := renderShadedLevel(original, mesh, outputDir, prefix, cfg)
		if err != nil {
			return nil, err
		}
		level.Files["turntable_ppm"] = shadedPath

		silPath, metrics, summary, err := renderSilhouetteLevel(original, mesh, outputDir, prefix, cfg)
		if err != nil {
			return nil, err
		}
		level.Files["silhouette_ppm"] = silPath
		level.SilhouetteMetrics = metrics
		level.SilhouetteSummary = summary

		highlightStart := level.TotalCollapses - level.StepsThisLevel
		highlightPath, err := renderHighlightLevel(original, mesh, sim.Records, highlightStart, outputDir, prefix, cfg)
		if err != nil {
			return nil, err
		}
		level.Files["highlight_ppm"] = highlightPath

		if cfg.SaveOBJ {
			objPath := filepath.Join(outputDir, "obj", prefix+".obj")
			if err := mesh.WriteOBJ(objPath); err != nil {
				return nil, err
			}
			level.Files["obj"] = objPath
		}
		report.Levels = append(report.Levels, level)
	}

	historyPath := filepath.Join(outputDir, "collapse-history.json")
	if err := SaveJSON(historyPath, sim.Records); err != nil {
		return nil, err
	}
	report.HistoryFile = historyPath

	if err := sim.UndoAll(); err != nil {
		return nil, err
	}
	report.UndoExact = reflect.DeepEqual(original, mesh)
	if cfg.SaveOBJ {
		restoredPath := filepath.Join(outputDir, "obj", "restored-after-undo.obj")
		if err := mesh.WriteOBJ(restoredPath); err != nil {
			return nil, err
		}
	}
	return report, nil
}

func renderShadedLevel(original, mesh *Mesh, outputDir, prefix string, cfg RunConfig) (string, error) {
	frame := NewViewFrame(original, cfg.ShadedTile, cfg.ShadedTile)
	views := TurntableViews(cfg.ShadedViews, cfg.InitialPitch)
	tiles := make([]*PPM, 0, len(views))
	for _, view := range views {
		tiles = append(tiles, RenderShaded(mesh, frame, view))
	}
	sheet := ContactSheet(tiles, len(tiles), ColorBackground)
	path := filepath.Join(outputDir, "ppm", prefix+"_turntable.ppm")
	return path, SavePPM(path, sheet)
}

func renderSilhouetteLevel(original, mesh *Mesh, outputDir, prefix string, cfg RunConfig) (string, []SilhouetteMetric, SilhouetteSummary, error) {
	frame := NewViewFrame(original, cfg.SilhouetteTile, cfg.SilhouetteTile)
	views := TurntableViews(cfg.SilhouetteViews, 0)
	metrics, overlays, summary := CompareSilhouettes(original, mesh, frame, views)
	sheet := ContactSheet(overlays, 4, RGB{235, 239, 245})
	path := filepath.Join(outputDir, "silhouette", prefix+"_deviation.ppm")
	return path, metrics, summary, SavePPM(path, sheet)
}

func renderHighlightLevel(original, mesh *Mesh, records []*CollapseRecord, start int, outputDir, prefix string, cfg RunConfig) (string, error) {
	frame := NewViewFrame(original, cfg.HighlightSize, cfg.HighlightSize)
	view := View{Yaw: 0.72, Pitch: 0.34}
	img := RenderCollapseHighlights(mesh, frame, view, records, start)
	path := filepath.Join(outputDir, "highlights", prefix+"_collapsed_edges.ppm")
	return path, SavePPM(path, img)
}
