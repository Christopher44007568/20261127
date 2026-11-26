package main

import (
	"flag"
	"fmt"
	"os"

	"bloomcompare/bloom"
)

func main() {
	width := flag.Int("width", 480, "output width in pixels")
	height := flag.Int("height", 180, "output height in pixels")
	out := flag.String("out", "out", "artifact output directory")
	flag.Parse()

	summary, err := bloom.Generate(*out, *width, *height)
	if err != nil {
		fmt.Fprintf(os.Stderr, "bloomdiff: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("generated %dx%d format comparison in %s\n", summary.Width, summary.Height, *out)
	for _, f := range summary.Formats {
		fmt.Printf("  %-10s max_error=%.8g clamped_values=%d ppm_mismatches=%d\n",
			f.Format, f.FinalError.MaxAbs, f.ClampedChannelValues, f.PPMReload.MismatchCount)
	}
}
