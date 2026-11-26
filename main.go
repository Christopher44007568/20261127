package main

import (
	"flag"
	"fmt"
	"os"

	"shadowcache/internal/demo"
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for PPM files and report.json")
	flag.Parse()

	report, err := demo.Run(*outDir)
	if err != nil {
		fmt.Fprintf(os.Stderr, "shadow cache demo failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println(demo.Summary(report))
	for _, c := range report.Checks {
		status := "PASS"
		if !c.OK {
			status = "FAIL"
		}
		fmt.Printf("  [%s] %s: %s\n", status, c.Name, c.Detail)
	}
	if !report.AllChecksPass {
		os.Exit(1)
	}
}
