package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for PNG, CSV, and JSON outputs")
	weight := flag.Int("weight", 900, "variable font weight in the 100..900 axis")
	samples := flag.Int("samples", 4, "supersampling factor per raster axis")
	flag.Parse()

	if *samples < 1 {
		*samples = 1
	}
	report, err := GenerateArtifacts(*outDir, *weight, *samples)
	if err != nil {
		log.Fatalf("generate weight-lab artifacts: %v", err)
	}

	fmt.Printf("weight=%d requested_delta=%.2f applied_delta=%.4f topology_valid=%v\n",
		report.Weight, report.RequestedDelta, report.Expansion.Delta, report.Topology.Valid)
	fmt.Printf("counter_area=%.2f design_units^2 intersections=%d naive_highlight_hits=%d\n",
		report.Expansion.Outline.CounterArea(), report.Topology.IntersectionCount, len(report.NaiveIntersections))
	for _, key := range []string{"original", "expanded", "highlight", "size_csv", "area_svg", "metrics_json"} {
		fmt.Printf("%-12s %s\n", key+":", report.Files[key])
	}
}
