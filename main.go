// Command featurelines extracts robust feature lines from meshes using
// curvature difference and normal-angle tracing, and renders the acceptance
// figures (original model, feature overlay, endpoint-error map).
//
// It uses only the Go standard library. Run:  go run .
package main

import (
	"fmt"
	"os"

	"featurelines/internal/pipeline"
	"featurelines/internal/report"
)

func main() {
	outs := pipeline.RunAll()

	const renderDir = "artifacts/render"
	sets := report.RenderAll(outs)
	must(report.WritePanels(renderDir, outs, sets))
	must(report.WriteCanonicalThree("artifacts", sets))
	must(report.WriteJSON("artifacts/metrics.json", outs))
	must(report.WriteFeatureOBJ("artifacts/features.obj", outs))
	must(report.WriteMarkdown("artifacts/验收报告.md", "render", outs))

	allPass := true
	for _, o := range outs {
		m := o.Metrics
		status := "PASS"
		if !m.Pass {
			status = "FAIL"
			allPass = false
		}
		fmt.Printf("[%s] %-8s coverage=%.3f closed=%v endpoints=%d maxErr=%.3f bridges=%d naiveSeg=%d\n",
			status, o.Case.ID, m.Coverage, m.ClosedLoop, m.EndpointCount,
			m.MaxEndpointError, m.BridgeCount, m.NaiveSegments)
		for _, chk := range m.Checks {
			mark := "ok"
			if !chk.OK {
				mark = "XX"
			}
			fmt.Printf("    (%s) %-42s %s\n", mark, chk.Name, chk.Detail)
		}
	}

	fmt.Println("\nwrote artifacts/render/*.png, artifacts/metrics.json, artifacts/features.obj, artifacts/验收报告.md")
	if !allPass {
		os.Exit(1)
	}
}

func must(err error) {
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(2)
	}
}
