package main

import (
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestHeadUVHardConstraints(t *testing.T) {
	m := NewOrganicHead()
	cfg := DefaultUnwrapConfig()
	cfg.AOSamples = 6
	atlas, sig, err := Unwrap(m, cfg)
	if err != nil {
		t.Fatalf("Unwrap returned error: %v", err)
	}
	rep := BuildReport(m, sig, atlas)

	if rep.FlippedFaces != 0 {
		t.Fatalf("found %d flipped faces; every island must keep orientation", rep.FlippedFaces)
	}
	if rep.ProtectedSeamEdges != 0 {
		t.Fatalf("protected face has %d seam edges", rep.ProtectedSeamEdges)
	}
	if rep.VisibleFrontSeamEdges != 0 {
		t.Fatalf("visible front has %d seam edges (length %.4f)",
			rep.VisibleFrontSeamEdges, rep.VisibleFrontSeamLength)
	}
	if rep.IslandCount < 1 || rep.IslandCount > 4 {
		t.Fatalf("expected a small number of charts, got %d", rep.IslandCount)
	}
	if rep.SeamEdges <= 20 || rep.SeamEdges > 240 {
		t.Fatalf("unexpected seam edge count %d", rep.SeamEdges)
	}

	for i, uv := range atlas.WedgeUV {
		if math.IsNaN(uv.X) || math.IsNaN(uv.Y) || math.IsInf(uv.X, 0) || math.IsInf(uv.Y, 0) {
			t.Fatalf("wedge %d has non-finite UV %+v", i, uv)
		}
		if uv.X < -0.001 || uv.X > 1.001 || uv.Y < -0.001 || uv.Y > 1.001 {
			t.Fatalf("wedge %d UV outside atlas: %+v", i, uv)
		}
	}
	for f, met := range atlas.Metrics {
		if met.Flipped || met.AreaRatio <= 0 || math.IsNaN(met.L2) || met.LInf <= 0 {
			t.Fatalf("bad metric for face %d: %+v", f, met)
		}
	}
}

func TestRunWritesValidRenders(t *testing.T) {
	dir := t.TempDir()
	if err := run(dir, 4); err != nil {
		t.Fatalf("run failed: %v", err)
	}
	for _, name := range []string{
		"uv_islands.png", "checker_model.png", "stretch_heatmap.png",
		"seam_plan.png", "unwrapped_head.obj", "metrics.json", "report.txt",
	} {
		path := filepath.Join(dir, name)
		st, err := os.Stat(path)
		if err != nil {
			t.Fatalf("missing output %s: %v", name, err)
		}
		if st.Size() < 128 {
			t.Fatalf("output %s is unexpectedly small: %d bytes", name, st.Size())
		}
		if filepath.Ext(name) == ".png" {
			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			img, err := png.Decode(f)
			_ = f.Close()
			if err != nil {
				t.Fatalf("invalid PNG %s: %v", name, err)
			}
			if img.Bounds().Dx() < 640 || img.Bounds().Dy() < 480 {
				t.Fatalf("render %s too small: %v", name, img.Bounds())
			}
		}
	}
}

func TestProtectedFaceMask(t *testing.T) {
	if !IsProtectedFacePoint(V3(0, 0.1, 0.7)) {
		t.Fatal("central face should be protected")
	}
	if IsVisibleFrontPoint(V3(0, -0.95, 0.35)) {
		t.Fatal("under-jaw seam band should not count as visible face front")
	}
	if IsVisibleFrontPoint(V3(0, 0.1, -0.7)) {
		t.Fatal("back of head must not count as front")
	}
}
