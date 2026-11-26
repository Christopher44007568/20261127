package main

import (
	"flag"
	"fmt"
	"log"
)

func main() {
	outputDir := flag.String("out", "outputs", "directory for mask PNGs and final composite")
	flag.Parse()

	if err := RenderDemo(*outputDir); err != nil {
		log.Fatalf("render clip demo: %v", err)
	}

	fmt.Printf("Wrote masks and composite to %s\n", *outputDir)
}
