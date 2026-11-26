// Command check runs the pipeline and prints metrics to the console.
package main

import (
	"fmt"

	"featurelines/internal/pipeline"
)

func main() {
	for _, o := range pipeline.RunAll() {
		m := o.Metrics
		fmt.Printf("=== %s ===\n", o.Case.ID)
		fmt.Printf("  robust chains=%d contours=%d coverage=%.3f closed=%v bridges=%d endpoints=%d maxErr=%.3f closureRes=%.3f falseLen=%.2f curvOnlyLen=%.1f\n",
			m.RobustChains, len(o.Robust.Contours), m.Coverage, m.ClosedLoop, m.BridgeCount,
			m.EndpointCount, m.MaxEndpointError, m.ClosureResidual, m.FalseFeatureLength, m.CurvOnlySeamLength)
		fmt.Printf("  naive segments=%d endpoints=%d gapMax=%.3f\n", m.NaiveSegments, m.NaiveEndpoints, m.NaiveGapMax)
		for _, chk := range m.Checks {
			mark := "PASS"
			if !chk.OK {
				mark = "FAIL"
			}
			fmt.Printf("  [%s] %-40s %s\n", mark, chk.Name, chk.Detail)
		}
		if o.Case.ID == "hole" {
			for i, ch := range o.Robust.Creases {
				fmt.Printf("    rchain %2d closed=%v verts=%3d len=%7.2f gaps=%d\n", i, ch.Closed, len(ch.Vertices), ch.Length(o.Case.Mesh), len(ch.Gaps))
			}
			for i, ch := range o.Naive.Creases {
				fmt.Printf("    nchain %2d closed=%v verts=%3d len=%7.2f\n", i, ch.Closed, len(ch.Vertices), ch.Length(o.Case.Mesh))
			}
		}
	}
}
