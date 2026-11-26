// Command coverlab is the alternate entry point: `go run ./cmd/coverlab`.
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
