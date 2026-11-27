package main

import (
	"math"
	"os"
	"path/filepath"
	"testing"
)

func preparedScene(t *testing.T, idx int) (*SceneResult, *Octree, *Topology) {
	t.Helper()
	scene := DefaultScenes()[idx]
	o := NewOctree(scene)
	top := ExtractTopology(o)
	top.RepairCracks(o)
	res := BuildMetrics(scene, o, top)
	res.NaiveBoundary = BoundaryEdgeCount(top.NaiveTris)
	res.FinalBoundary = BoundaryEdgeCount(top.Tris)
	return res, o, top
}

func TestBoxSharpCornersAreSubVoxel(t *testing.T) {
	res, o, top := preparedScene(t, 1)
	if res.VertexCount < 100 {
		t.Fatalf("too few vertices: %d", res.VertexCount)
	}
	if got := BoundaryEdgeCount(top.Tris); got != 0 {
		t.Fatalf("box mesh is not watertight: %d boundary edges", got)
	}
	if len(res.SharpBins) != 1 {
		t.Fatalf("expected one finest-level sharp bin, got %+v", res.SharpBins)
	}
	bin := res.SharpBins[0]
	if bin.Depth != 7 || math.Abs(bin.H-o.HFinest) > 1e-9 {
		t.Fatalf("unexpected local level: depth=%d h=%f", bin.Depth, bin.H)
	}
	if bin.MaxSDF > 2e-4 {
		t.Fatalf("sharp max error %g exceeds tolerance", bin.MaxSDF)
	}
	if bin.BaselineMax < 100*bin.MaxSDF {
		t.Fatalf("normal QEF should greatly improve corner error: qef=%g baseline=%g", bin.MaxSDF, bin.BaselineMax)
	}
}

func TestThinShellHasTwoSurfacesAndRemainsWatertight(t *testing.T) {
	res, _, top := preparedScene(t, 2)
	if res.VertexCount < 1000 {
		t.Fatalf("shell collapsed: vertices=%d", res.VertexCount)
	}
	if got := BoundaryEdgeCount(top.Tris); got != 0 {
		t.Fatalf("shell has %d boundary edges after repair", got)
	}
	if len(res.SharpBins) != 1 || res.SharpBins[0].Count < 100 {
		t.Fatalf("shell sharp bins missing/too small: %+v", res.SharpBins)
	}
	if res.MaxSDF > 2e-4 {
		t.Fatalf("shell SDF error too large: %g", res.MaxSDF)
	}
	if res.SharpBins[0].BaselineMax < 100*res.SharpBins[0].MaxSDF {
		t.Fatalf("thin shell baseline error not clearly worse: %+v", res.SharpBins[0])
	}
}

func TestSphereSmoothControl(t *testing.T) {
	res, _, top := preparedScene(t, 0)
	if BoundaryEdgeCount(top.Tris) != 0 {
		t.Fatalf("sphere has boundary edges")
	}
	if len(res.SharpBins) != 0 {
		t.Fatalf("sphere must not be classified as sharp: %+v", res.SharpBins)
	}
	if res.MaxSDF > 2e-3 {
		t.Fatalf("sphere error too large: %g", res.MaxSDF)
	}
}

func TestRenderSmoke(t *testing.T) {
	dir := t.TempDir()
	var results []*SceneResult
	for i := range DefaultScenes() {
		res, _, _ := preparedScene(t, i)
		results = append(results, res)
	}
	if err := RenderAll(dir, results); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"sphere.png", "box.png", "shell.png"} {
		info, err := os.Stat(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		if info.Size() < 20_000 {
			t.Fatalf("%s is unexpectedly small: %d", name, info.Size())
		}
	}
}
