package main

import (
	"flag"
	"fmt"
	"os"

	"materiallod/internal/pipeline"
)

func main() {
	defaults := pipeline.DefaultOptions()
	out := flag.String("out", defaults.Out, "directory for PNG, CSV, JSON, and Markdown artifacts")
	flag.Parse()

	art, err := pipeline.Generate(pipeline.Options{Out: *out})
	if err != nil {
		fmt.Fprintf(os.Stderr, "material-lod: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("material-lod: wrote %d materials, %d view samples, %d boundary rows to %s\n",
		len(art.Materials), len(art.Sweeps), len(art.Continuity.Rows), *out)
	for _, boundary := range art.Continuity.Boundaries {
		status := "PASS"
		if !boundary.Passed {
			status = "FAIL"
		}
		fmt.Printf("boundary %.1f %s max_step=%.5f %s\n",
			boundary.BoundaryLOD, boundary.Transition, boundary.ContinuousMaxStep, status)
	}
	for _, boundary := range art.Continuity.Boundaries {
		if !boundary.Passed {
			os.Exit(1)
		}
	}
}
