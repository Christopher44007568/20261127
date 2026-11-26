// Command fillmask demonstrates EvenOdd and NonZero fill rules on the same
// self-intersecting path, together with integer-aligned clipping regions and
// nested subpaths. It writes binary fill masks, nearest-neighbour boundary
// enlargements, a labelled composite PNG and a pixel-level JSON report.
//
// Only the Go standard library is used:
//
//	go run . -out artifacts
//	go run main.go -out artifacts   # equivalent
//	go test ./...
package main

import (
	"flag"
	"fmt"
	"os"

	"fillmask/demo"
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for masks, zooms, composite and report")
	flag.Parse()

	rep, err := demo.Run(*outDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "render failed: %v\n", err)
		os.Exit(2)
	}

	fmt.Println("fillmask artifacts written to", *outDir)
	for _, c := range rep.Checks {
		mark := "PASS"
		if !c.OK {
			mark = "FAIL"
		}
		fmt.Printf("  [%s] %s -- %s\n", mark, c.Name, c.Detail)
	}
	fmt.Printf("rule-different pixels: %d; protected hole pixels: %d\n",
		rep.RuleDiffPixels, rep.ProtectedHolePx)

	if !rep.OK {
		fmt.Fprintln(os.Stderr, "verification failed; see report.json")
		os.Exit(1)
	}
}
