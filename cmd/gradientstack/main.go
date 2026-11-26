package main

import (
	"flag"
	"fmt"
	"os"

	"gradientstack/internal/studio"
)

func main() {
	out := flag.String("out", "out", "directory for generated PNG artifacts")
	flag.Parse()

	report, err := studio.Generate(*out)
	if err != nil {
		fmt.Fprintln(os.Stderr, "gradientstack:", err)
		os.Exit(1)
	}
	fmt.Printf("wrote %d files to %s\n", len(report.Files)+1, *out)
	fmt.Printf("banding: raw=%g dithered=%g; tile delta=%.3g; seam=%g LSB\n",
		report.Checks.RawBandRuns,
		report.Checks.DitheredBandRuns,
		report.Checks.TileBoundaryDelta,
		report.Checks.SeamResidualLSB,
	)
}
