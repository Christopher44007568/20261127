// Command coverlab generates coverage comparisons, edge close-ups, coverage
// maps, error heatmaps and continuity/stain evidence using only the Go
// standard library. Run `go run . -out out`.
package main

import (
	"flag"
	"log"

	"coverlab/app"
)

func main() {
	out := flag.String("out", "out", "directory to write artifacts into")
	flag.Parse()
	if err := app.Run(*out); err != nil {
		log.Fatalf("coverlab: %v", err)
	}
}
