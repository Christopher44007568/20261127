package app

import (
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestRunProducesCompleteDiagnostics(t *testing.T) {
	out := t.TempDir()
	result, err := Run(Options{
		OutDir: out, CreaseAngleDeg: 40, CollapseDeg: 18,
		Smooth: true, Width: 200, Height: 150, SuperSample: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Features.Edges) == 0 {
		t.Fatal("sample should contain feature edges")
	}
	if math.Abs(result.Remesh.FeatureLengthRatio-1) > 1e-9 {
		t.Fatalf("feature length ratio = %.9f, want 1", result.Remesh.FeatureLengthRatio)
	}
	if len(result.Analysis.Collapsed) == 0 {
		t.Fatal("sample should contain a collapsed quad")
	}
	if len(result.Quad.Faces) != len(result.Source.Faces)*3 {
		t.Fatalf("quad count = %d, want %d", len(result.Quad.Faces), len(result.Source.Faces)*3)
	}

	expected := []string{
		"source_mesh.obj", "quad_mesh.obj", "original.png", "quad_mesh.png",
		"curvature_bands.png", "local_zoom.png", "edge_length_histogram.png",
		"edge_lengths.csv", "quad_face_metrics.csv", "feature_edges.csv", "edge_length_histogram.csv",
		"report.txt",
	}
	for _, name := range expected {
		info, err := os.Stat(filepath.Join(out, name))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", name)
		}
	}
	for _, name := range []string{"original.png", "quad_mesh.png", "curvature_bands.png", "local_zoom.png"} {
		f, err := os.Open(filepath.Join(out, name))
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := png.DecodeConfig(f)
		f.Close()
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if cfg.Width != 200 || cfg.Height != 150 {
			t.Fatalf("%s dimensions = %dx%d", name, cfg.Width, cfg.Height)
		}
	}
}

func TestRunAcceptsOBJInput(t *testing.T) {
	out := t.TempDir()
	objPath := filepath.Join(out, "input.obj")
	obj := "v 0 0 0\nv 1 0 0\nv 0 1 0\nv 0 0 1\nf 1 2 3\nf 2 1 4\n"
	if err := os.WriteFile(objPath, []byte(obj), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Run(Options{
		InputPath: objPath, OutDir: filepath.Join(out, "result"),
		CreaseAngleDeg: 30, CollapseDeg: 18, Smooth: false,
		Width: 160, Height: 120, SuperSample: 1,
	}); err != nil {
		t.Fatal(err)
	}
}
