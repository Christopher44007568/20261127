package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
	"path/filepath"

	"example.com/bezierstroke"
)

type Scene struct {
	Key         string
	Title       string
	Curve       bezierstroke.Cubic
	Center      bezierstroke.Point
	Scale       float64
	W, H        int
	StrokeWidth float64
	Tolerance   float64
}

type SceneMetrics struct {
	Key              string                 `json:"key"`
	Title            string                 `json:"title"`
	Curve            [4]bezierstroke.Point  `json:"curve"`
	Viewport         map[string]interface{} `json:"viewport"`
	TolerancePx      float64                `json:"tolerance_px"`
	StrokeWidthPx    float64                `json:"stroke_width_px"`
	CurveLengthPx    float64                `json:"curve_length_px"`
	AdaptiveVertices int                    `json:"adaptive_vertices"`
	AdaptiveSegments int                    `json:"adaptive_segments"`
	MaxChordErrorPx  float64                `json:"max_chord_error_px"`
	Fixed8MaxErrorPx float64                `json:"fixed_8_max_error_px"`
	MeshPolygons     int                    `json:"mesh_polygons"`
	OutputOriginal   string                 `json:"output_original"`
	OutputMesh       string                 `json:"output_mesh"`
	OutputFinal      string                 `json:"output_final"`
}

type Report struct {
	Scenes []SceneMetrics `json:"scenes"`
}

func main() {
	outDir := flag.String("out", "artifacts", "output directory for diagnostic PNGs")
	samples := flag.Int("samples", 16, "coverage samples per pixel axis")
	flag.Parse()
	if *samples < 1 {
		*samples = 1
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fatal(err)
	}

	scenes := demoScenes()
	report := Report{}
	for _, scene := range scenes {
		prepared := prepareScene(scene)
		paths := map[string]string{}
		stages := []struct {
			name string
			draw func(*CanvasRenderer)
		}{
			{"original", (*CanvasRenderer).drawOriginal},
			{"mesh", (*CanvasRenderer).drawMesh},
			{"final", (*CanvasRenderer).drawFinal},
		}
		for _, stage := range stages {
			name := scene.Key + "_" + stage.name + ".png"
			path := filepath.Join(*outDir, name)
			renderer := newCanvasRenderer(prepared, *samples)
			stage.draw(renderer)
			if err := renderer.Canvas.SavePNG(path); err != nil {
				fatal(err)
			}
			paths[stage.name] = path
		}
		metrics := prepared.metrics(paths)
		report.Scenes = append(report.Scenes, metrics)
		fmt.Printf("%s: %d adaptive vertices, max error %.4f px -> %s\n",
			scene.Key, metrics.AdaptiveVertices, metrics.MaxChordErrorPx,
			filepath.Join(*outDir, scene.Key+"_final.png"))
		if scene.Key == "near_line" {
			copyFile(filepath.Join(*outDir, "original.png"), paths["original"])
			copyFile(filepath.Join(*outDir, "mesh_edges.png"), paths["mesh"])
			copyFile(filepath.Join(*outDir, "final.png"), paths["final"])
		}
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(filepath.Join(*outDir, "metrics.json"), data, 0o644); err != nil {
		fatal(err)
	}
}

func demoScenes() []Scene {
	near := Scene{
		Key:   "near_line",
		Title: "Near-line cubic, adaptive zoom",
		Curve: bezierstroke.Cubic{
			P0: bezierstroke.Point{X: -1.2, Y: 0},
			P1: bezierstroke.Point{X: -0.4, Y: 0.05},
			P2: bezierstroke.Point{X: 0.4, Y: 0.05},
			P3: bezierstroke.Point{X: 1.2, Y: 0},
		},
		Center:      bezierstroke.Point{X: 0, Y: 0.01875},
		Scale:       640,
		W:           1600,
		H:           820,
		StrokeWidth: 11,
		Tolerance:   0.08,
	}
	sharp := Scene{
		Key:   "sharp_bend",
		Title: "Cusp / near-180-degree turn",
		Curve: bezierstroke.Cubic{
			P0: bezierstroke.Point{X: -1, Y: 0},
			P1: bezierstroke.Point{X: 1, Y: 1},
			P2: bezierstroke.Point{X: -1, Y: 1},
			P3: bezierstroke.Point{X: 1, Y: 0},
		},
		Center:      bezierstroke.Point{X: 0, Y: 0},
		Scale:       380,
		W:           1600,
		H:           820,
		StrokeWidth: 16,
		Tolerance:   0.25,
	}
	long := Scene{
		Key:   "long_curve",
		Title: "Long curve, constant-width stroke",
		Curve: bezierstroke.Cubic{
			P0: bezierstroke.Point{X: -10, Y: -1},
			P1: bezierstroke.Point{X: -3, Y: 2.2},
			P2: bezierstroke.Point{X: 3, Y: -2.2},
			P3: bezierstroke.Point{X: 10, Y: 1},
		},
		Center:      bezierstroke.Point{X: 0, Y: 0},
		Scale:       76,
		W:           1600,
		H:           820,
		StrokeWidth: 8,
		Tolerance:   0.10,
	}
	return []Scene{near, sharp, long}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func finite(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 0
	}
	return v
}

func copyFile(dst, src string) {
	data, err := os.ReadFile(src)
	if err != nil {
		fatal(err)
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		fatal(err)
	}
}
