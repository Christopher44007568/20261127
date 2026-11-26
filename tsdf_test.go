package main

import (
	"math"
	"testing"
)

func TestUpdateWeightAverageAndCap(t *testing.T) {
	w, v := UpdateWeight(0, 0, 0.4, 2, 10)
	if w != 2 || v != 0.4 {
		t.Fatalf("first update = (%v,%v), want (2,0.4)", w, v)
	}
	w, v = UpdateWeight(w, v, 0.8, 2, 10)
	if math.Abs(float64(w-4)) > 1e-6 || math.Abs(float64(v-0.6)) > 1e-6 {
		t.Fatalf("average update = (%v,%v), want (4,0.6)", w, v)
	}
	w, _ = UpdateWeight(w, v, 1, 100, 6)
	if w != 6 {
		t.Fatalf("capped weight = %v, want 6", w)
	}
	beforeW, beforeV := w, v
	w, v = UpdateWeight(w, v, 1, 0, 6)
	if w != beforeW || v != beforeV {
		t.Fatalf("zero weight changed state: (%v,%v)", w, v)
	}
}

func TestBoxSDFSign(t *testing.T) {
	b := Box{Center: V3(0, 0, 0), Half: V3(1, 1, 1)}
	if d := b.SDF(V3(2, 0, 0)); math.Abs(d-1) > 1e-9 {
		t.Fatalf("outside distance = %v, want 1", d)
	}
	if d := b.SDF(V3(0.5, 0, 0)); math.Abs(d+0.5) > 1e-9 {
		t.Fatalf("inside distance = %v, want -0.5", d)
	}
	if d := b.SDF(V3(1, 0, 0)); math.Abs(d) > 1e-9 {
		t.Fatalf("surface distance = %v, want 0", d)
	}
}

func TestSyntheticPlaneExtraction(t *testing.T) {
	g := NewVoxelGrid(FusionConfig{
		VoxelSize: .05, NX: 24, NY: 24, NZ: 12,
		Min: V3(-.6, -.6, -.3), Truncation: .1,
	})
	for iz := 0; iz < g.NZ; iz++ {
		for iy := 0; iy < g.NY; iy++ {
			for ix := 0; ix < g.NX; ix++ {
				id := g.Index(ix, iy, iz)
				p := g.Center(ix, iy, iz)
				g.Value[id] = float32(clampFloat(p.Y/g.Truncation, -1, 1))
				g.Weight[id] = 2
			}
		}
	}

	cells := ExtractThresholdCells(g, .25, 1)
	if len(cells) == 0 {
		t.Fatal("expected threshold cells around y=0")
	}
	segments := ExtractSliceContour(g, g.NZ/2, 1)
	if len(segments) < 10 {
		t.Fatalf("contour segments = %d, want a continuous line", len(segments))
	}
	for _, s := range segments {
		if math.Abs(s.A.Y) > .04 || math.Abs(s.B.Y) > .04 {
			t.Fatalf("zero crossing too far from plane: %+v", s)
		}
	}

	mesh := ExtractMarchingTetrahedra(g, 1)
	if len(mesh.Vertices) < 100 || len(mesh.Triangles) < 100 {
		t.Fatalf("mesh too small: %d verts %d tris", len(mesh.Vertices), len(mesh.Triangles))
	}
}

func TestRunAllScenesQuality(t *testing.T) {
	opts := DefaultRunOptions()
	opts.OutDir = t.TempDir()
	report, err := Run(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Scenes) != 3 {
		t.Fatalf("scenes = %d, want 3", len(report.Scenes))
	}
	for _, scene := range report.Scenes {
		if scene.Contour.RMSE > .018 {
			t.Fatalf("%s contour RMSE %.4f exceeds 18 mm", scene.ID, scene.Contour.RMSE)
		}
		if scene.CoarseControl == nil {
			t.Fatalf("%s missing coarse control", scene.ID)
		}
		for _, check := range scene.Checks {
			if !check.Pass {
				t.Fatalf("%s check %s failed: %.4g vs %.4g %s", scene.ID, check.ID, check.Value, check.Threshold, check.Unit)
			}
		}
	}
	if !report.Pass {
		t.Fatal("report marked as failed")
	}
}
