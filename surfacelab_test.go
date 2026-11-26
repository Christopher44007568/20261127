package main

import (
	"encoding/binary"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func TestReadBinaryPLY(t *testing.T) {
	path := filepath.Join(t.TempDir(), "binary.ply")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	header := "ply\nformat binary_little_endian 1.0\nelement vertex 2\n" +
		"property float x\nproperty float y\nproperty float z\nend_header\n"
	if _, err := f.WriteString(header); err != nil {
		t.Fatal(err)
	}
	for _, p := range []V3{{1, 2, 3}, {-1.5, 2.25, 4}} {
		for _, v := range []float32{float32(p.X), float32(p.Y), float32(p.Z)} {
			if err := binary.Write(f, binary.LittleEndian, v); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := f.Close(); err != nil {
		t.Fatal(err)
	}
	c, err := ReadPLYFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(c.P) != 2 || math.Abs(c.P[1].X+1.5) > 1e-6 || c.P[0].Z != 3 {
		t.Fatalf("bad binary PLY points: %+v", c.P)
	}
}

func TestOrientNormalsMSTCorrectsFlips(t *testing.T) {
	var pts []V3
	var normals []V3
	for ix := 0; ix < 14; ix++ {
		for iy := 0; iy < 14; iy++ {
			pts = append(pts, V3{float64(ix) * 0.04, float64(iy) * 0.04, 0})
			n := V3{0.02 * float64(ix-7), -0.015 * float64(iy-7), 1}
			n = Normalize(n)
			if (ix*7+iy*3)%5 < 2 {
				n = n.Mul(-1)
			}
			normals = append(normals, n)
		}
	}
	rep := OrientNormalsMST(pts, normals, 8, V3{0, 0, 1})
	if rep.Components != 1 {
		t.Fatalf("components=%d, want 1", rep.Components)
	}
	for i, n := range normals {
		if n.Z < 0.85 {
			t.Fatalf("normal %d not oriented upward: %+v", i, n)
		}
	}
}

func TestFilterOutliersRejectsSparsePoints(t *testing.T) {
	var pts []V3
	for ix := 0; ix < 20; ix++ {
		for iy := 0; iy < 20; iy++ {
			pts = append(pts, V3{float64(ix) * 0.035, float64(iy) * 0.035, 0.001 * math.Sin(float64(ix))})
		}
	}
	ghost := V3{2.5, 2.5, 1.4}
	pts = append(pts, ghost)
	rep := FilterOutliers(pts, 10, 3)
	keep := map[int]bool{}
	for _, id := range rep.Keep {
		keep[id] = true
	}
	if keep[len(pts)-1] {
		t.Fatal("far outlier was retained")
	}
	if len(rep.Keep) < 390 {
		t.Fatalf("too many surface points removed: kept=%d", len(rep.Keep))
	}
}

func TestQuadricCurvatureRecovery(t *testing.T) {
	m := &Mesh{}
	for ix := -8; ix <= 8; ix++ {
		for iy := -8; iy <= 8; iy++ {
			x := float64(ix) * 0.02
			y := float64(iy) * 0.02
			z := 0.8*x*x + 0.3*y*y
			m.P = append(m.P, V3{x, y, z})
			m.N = append(m.N, Normalize(V3{-1.6 * x, -0.6 * y, 1}))
		}
	}
	hs := EstimateMeshCurvature(m, 32)
	center := 8*17 + 8
	if math.Abs(hs[center]-1.1) > 0.08 {
		t.Fatalf("H=%v, want about 1.1", hs[center])
	}
}

func TestDemoPipelineQualityAndOutputs(t *testing.T) {
	out := filepath.Join(t.TempDir(), "result")
	report, err := Run(RunConfig{Out: out, Seed: 20261127, Yaw: -0.68, Pitch: 0.58})
	if err != nil {
		t.Fatal(err)
	}
	if report.KeptPoints < 30000 || report.RemovedOutliers < 120 {
		t.Fatalf("bad filtering counts: %+v", report)
	}
	if tri := report.Mesh["triangles"].(int); tri < 50000 {
		t.Fatalf("too few triangles: %d", tri)
	}
	if report.Geometry == nil || report.Geometry.RMSE > 0.008 {
		t.Fatalf("geometry RMSE too high: %+v", report.Geometry)
	}
	if math.Abs(report.Geometry.SignedMean) > 0.004 {
		t.Fatalf("swelling bias too large: %+v", report.Geometry)
	}
	if report.Curvature == nil || report.Curvature.RMSE > 0.20 {
		t.Fatalf("curvature RMSE too high: %+v", report.Curvature)
	}
	for _, h := range report.Holes {
		if !h.OpenCore || h.CoreTriangleHits != 0 {
			t.Fatalf("hole was bridged: %+v", h)
		}
	}
	for _, name := range []string{"panel.png", "01_input_points.png", "02_normal_color.png",
		"03_reconstructed_mesh.png", "04_curvature_error.png", "mesh/reconstructed.ply", "report.json"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatalf("missing output %s: %v", name, err)
		}
	}
}
