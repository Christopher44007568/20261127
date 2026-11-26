package main

import (
	"image/png"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
)

func TestSyntheticSegmentationIsConnected(t *testing.T) {
	g := NewSyntheticGrid(96, 72)
	seeds, err := FarthestSeeds(g, 7)
	if err != nil {
		t.Fatal(err)
	}
	result, err := Segment(g, seeds, nil)
	if err != nil {
		t.Fatal(err)
	}
	report := CheckConnectivity(g, result.Labels, result.Seeds)
	if !report.Connected {
		t.Fatalf("regions not 4-connected: %v", report.Problems)
	}
	if result.RepairedCells != 0 {
		t.Fatalf("topology-safe relaxation should not need repair, repaired=%d", result.RepairedCells)
	}
}

func TestManySeedChangesRemainConnectedAroundHoles(t *testing.T) {
	g := NewSyntheticGrid(56, 42)
	openIDs := make([]int, 0, g.OpenCount())
	for id, blocked := range g.Blocked {
		if !blocked {
			openIDs = append(openIDs, id)
		}
	}
	rng := rand.New(rand.NewSource(20261127))

	for trial := 0; trial < 40; trial++ {
		count := 1 + rng.Intn(12)
		seeds := make([]Seed, 0, count)
		used := map[int]bool{}
		for len(seeds) < count {
			id := openIDs[rng.Intn(len(openIDs))]
			if used[id] {
				continue
			}
			used[id] = true
			seeds = append(seeds, Seed{X: id % g.W, Y: id / g.W})
		}

		opts := DefaultSegmentOptions()
		opts.RelaxIterations = 5
		if trial%2 == 0 {
			opts.DisableTopologyCheck = true
		}
		result, err := Segment(g, seeds, &opts)
		if err != nil {
			t.Fatalf("trial %d seeds %v: %v", trial, seeds, err)
		}
		report := CheckConnectivity(g, result.Labels, result.Seeds)
		if !report.Connected {
			t.Fatalf("trial %d seeds %v produced pseudo-connectivity: %v", trial, seeds, report.Problems)
		}
	}
}

func TestDiagonalTouchAcrossBlockedCenterIsRepaired(t *testing.T) {
	g, err := NewGrid(3, 3)
	if err != nil {
		t.Fatal(err)
	}
	g.Blocked[g.ID(1, 1)] = true
	seeds := []Seed{{X: 0, Y: 0}, {X: 2, Y: 0}}
	labels := []int{
		0, 1, 1,
		1, -1, 1,
		1, 1, 0,
	}

	changed, err := EnsureConnected(g, labels, seeds)
	if err != nil {
		t.Fatal(err)
	}
	if changed == 0 {
		t.Fatal("diagonal orphan component should have been recolored")
	}
	report := CheckConnectivity(g, labels, seeds)
	if !report.Connected {
		t.Fatalf("after repair: %v", report.Problems)
	}
	if labels[g.ID(2, 2)] != 1 {
		t.Fatalf("diagonal pseudo-component should merge with adjacent root, got label %d", labels[g.ID(2, 2)])
	}
}

func TestClosedRingHoleCannotBeCrossed(t *testing.T) {
	g, err := NewGrid(11, 11)
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			if math.Max(math.Abs(float64(x-5)), math.Abs(float64(y-5))) == 3 {
				g.Blocked[g.ID(x, y)] = true
			}
		}
	}

	seeds := []Seed{{X: 0, Y: 0}, {X: 10, Y: 10}}
	result, err := Segment(g, seeds, &SegmentOptions{RelaxIterations: 0})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Seeds) != 3 || result.AutoAddedSeeds != 1 {
		t.Fatalf("expected one coverage seed inside ring, got %+v added=%d", result.Seeds, result.AutoAddedSeeds)
	}
	report := CheckConnectivity(g, result.Labels, result.Seeds)
	if !report.Connected {
		t.Fatalf("ring segmentation not connected: %v", report.Problems)
	}

	strict := DefaultSegmentOptions()
	strict.StrictSeeds = true
	strict.RelaxIterations = 0
	if _, err := Segment(g, seeds, &strict); err == nil {
		t.Fatal("strict mode should reject a seedless component inside the ring")
	}
}

func TestGeodesicDistanceDetoursAroundWall(t *testing.T) {
	g, err := NewGrid(9, 9)
	if err != nil {
		t.Fatal(err)
	}
	for y := 2; y <= 6; y++ {
		g.Blocked[g.ID(4, y)] = true
	}
	seed := Seed{X: 1, Y: 4}
	target := g.ID(7, 4)
	field, err := MultiSourceDistance(g, []Seed{seed}, nil, DistanceOptions{EightNeighbors: false})
	if err != nil {
		t.Fatal(err)
	}
	if math.IsInf(field.Dist[target], 1) {
		t.Fatal("target should be reachable by going around wall")
	}
	if field.Dist[target] < 11.9 {
		t.Fatalf("distance %.2f does not account for detour around wall", field.Dist[target])
	}

	for id := target; id != g.ID(seed.X, seed.Y); id = field.Parent[id] {
		if id < 0 || g.Blocked[id] {
			t.Fatalf("parent chain entered invalid cell %d", id)
		}
		parent := field.Parent[id]
		if parent < 0 {
			t.Fatal("parent chain ended before seed")
		}
		dx := absInt(id%g.W - parent%g.W)
		dy := absInt(id/g.W - parent/g.W)
		if dx+dy != 1 {
			t.Fatalf("segmentation parent edge is not 4-connected: %d -> %d", id, parent)
		}
	}
}

func TestWriteOutputsCreatesReadablePNGs(t *testing.T) {
	g := NewSyntheticGrid(48, 36)
	result, err := Segment(g, []Seed{{X: 8, Y: 8}, {X: 38, Y: 28}, {X: 24, Y: 18}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	files, err := WriteOutputs(dir, result)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{files.SeedDistribution, files.RegionColors, files.BoundaryLines, files.DistanceHeatmap} {
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := png.DecodeConfig(f)
		closeErr := f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
		if cfg.Width <= 0 || cfg.Height <= 0 {
			t.Fatalf("invalid PNG config for %s", path)
		}
	}
	if _, err := os.Stat(filepath.Join(dir, "segmentation.json")); err != nil {
		t.Fatal(err)
	}
}
