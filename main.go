package main

import (
	"flag"
	"fmt"
	"os"
	"time"

	"meshdiagnostics/internal/app"
)

func main() {
	var opt app.Options
	var input string
	flag.StringVar(&input, "in", "", "input triangle OBJ (default: procedural organic model)")
	flag.StringVar(&opt.OutDir, "out", "out", "output directory")
	flag.Float64Var(&opt.CreaseAngleDeg, "crease", 40, "feature-edge dihedral threshold in degrees")
	flag.Float64Var(&opt.CollapseDeg, "collapse", 18, "quad corner-angle collapse threshold in degrees")
	flag.BoolVar(&opt.Smooth, "smooth", true, "apply feature-aware Catmull-Clark smoothing")
	flag.IntVar(&opt.Width, "width", 1200, "image width in pixels")
	flag.IntVar(&opt.Height, "height", 900, "image height in pixels")
	flag.IntVar(&opt.SuperSample, "ss", 2, "supersampling factor")
	flag.Parse()
	opt.InputPath = input

	result, err := app.Run(opt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "meshdiagnostics: %v\n", err)
		os.Exit(1)
	}
	a := result.Analysis
	fmt.Printf("source: %d vertices, %d triangles, %d feature edges\n",
		len(result.Source.Vertices), len(result.Source.Faces), len(result.Features.Edges))
	fmt.Printf("quad:   %d vertices, %d quads, feature length kept %.2f%%\n",
		len(result.Quad.Vertices), len(result.Quad.Faces), result.Remesh.FeatureLengthRatio*100)
	fmt.Printf("edges:  min %.4f / median %.4f / max %.4f, CV %.3f\n",
		a.EdgeStats.Min, a.EdgeStats.Median, a.EdgeStats.Max, a.EdgeStats.CV)
	fmt.Printf("angles: %d/%d collapsed quads (threshold %.1f deg), worst %.2f deg (face %d)\n",
		len(a.Collapsed), a.FaceCount, opt.CollapseDeg, a.WorstMetrics.MinAngleDeg, a.WorstFace)
	fmt.Printf("outputs written to %s (%s)\n", opt.OutDir, result.Duration.Round(time.Millisecond))
}
