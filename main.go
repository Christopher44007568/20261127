package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
	"strconv"
	"strings"
)

func main() {
	outDir := flag.String("out", "outputs", "directory for PNG, SVG, and JSON outputs")
	width := flag.Int("width", 192, "grid width")
	height := flag.Int("height", 144, "grid height")
	seedCount := flag.Int("seeds-count", 8, "number of automatic farthest-point seeds")
	seedText := flag.String("seeds", "", `optional seeds as "x1,y1;x2,y2" in grid coordinates`)
	iterations := flag.Int("relax-iterations", 30, "boundary relaxation iterations")
	geoWeight := flag.Float64("geo-weight", 0.22, "geodesic fidelity weight")
	smoothWeight := flag.Float64("smooth-weight", 0.46, "boundary perimeter smoothing weight")
	featureWeight := flag.Float64("feature-weight", 1.05, "curvature/depression attraction weight")
	featureDiscount := flag.Float64("feature-discount", 0.16, "geodesic cost discount on features")
	noChanged := flag.Bool("no-changed-demo", false, "do not render a second changed-seed scenario")
	flag.Parse()

	g := NewSyntheticGrid(*width, *height)
	opts := DefaultSegmentOptions()
	opts.SeedCount = *seedCount
	opts.RelaxIterations = *iterations
	opts.GeoWeight = *geoWeight
	opts.SmoothWeight = *smoothWeight
	opts.FeatureWeight = *featureWeight
	opts.FeatureDiscount = *featureDiscount

	requested, err := parseSeeds(*seedText)
	if err != nil {
		log.Fatal(err)
	}

	result, err := Segment(g, requested, &opts)
	if err != nil {
		log.Fatalf("segmentation failed: %v", err)
	}
	files, err := WriteOutputs(*outDir, result)
	if err != nil {
		log.Fatalf("write outputs failed: %v", err)
	}
	printSummary("default", result, files)

	if *seedText == "" && !*noChanged {
		first := nearestOpen(g, int(float64(g.W)*0.13), int(float64(g.H)*0.84))
		changedSeeds, err := FarthestSeedsStartingAt(g, *seedCount, first)
		if err != nil {
			log.Fatal(err)
		}
		changed, err := Segment(g, changedSeeds, &opts)
		if err != nil {
			log.Fatalf("changed-seed segmentation failed: %v", err)
		}
		changedDir := filepath.Join(*outDir, "changed_seeds")
		changedFiles, err := WriteOutputs(changedDir, changed)
		if err != nil {
			log.Fatal(err)
		}
		printSummary("changed_seeds", changed, changedFiles)
	}
}

func parseSeeds(text string) ([]Seed, error) {
	if strings.TrimSpace(text) == "" {
		return nil, nil
	}
	parts := strings.Split(text, ";")
	seeds := make([]Seed, 0, len(parts))
	for i, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		xy := strings.Split(part, ",")
		if len(xy) != 2 {
			return nil, fmt.Errorf("seed %d must be x,y", i)
		}
		x, err := strconv.Atoi(strings.TrimSpace(xy[0]))
		if err != nil {
			return nil, fmt.Errorf("seed %d x: %w", i, err)
		}
		y, err := strconv.Atoi(strings.TrimSpace(xy[1]))
		if err != nil {
			return nil, fmt.Errorf("seed %d y: %w", i, err)
		}
		seeds = append(seeds, Seed{X: x, Y: y})
	}
	return seeds, nil
}

func nearestOpen(g *Grid, x, y int) Seed {
	if g.Open(x, y) {
		return Seed{X: x, Y: y}
	}
	best := Seed{}
	bestD := int(^uint(0) >> 1)
	for yy := 0; yy < g.H; yy++ {
		for xx := 0; xx < g.W; xx++ {
			if !g.Open(xx, yy) {
				continue
			}
			dx := xx - x
			dy := yy - y
			d := dx*dx + dy*dy
			if d < bestD {
				bestD = d
				best = Seed{X: xx, Y: yy}
			}
		}
	}
	return best
}

func printSummary(name string, r *Result, files OutputFiles) {
	fmt.Printf("[%s] seeds=%d flips=%d repaired_cells=%d boundary_edges=%d feature_score=%.3f connected=%v energy=%.3f\n",
		name, len(r.Seeds), r.Flips, r.RepairedCells, r.BoundaryEdgeCount, r.BoundaryFeature, r.Connectivity.Connected, r.Energy)
	fmt.Printf("[%s] %s, %s, %s, %s\n", name, files.SeedDistribution, files.RegionColors, files.BoundaryLines, files.DistanceHeatmap)
}
