// Package pipeline wires cases to extraction and evaluation.
package pipeline

import (
	"featurelines/internal/cases"
	"featurelines/internal/eval"
	"featurelines/internal/feature"
)

// Output bundles all extraction results and metrics for one case.
type Output struct {
	Case     *cases.Case
	Opt      feature.Options
	Robust   *feature.Result
	Naive    *feature.Result
	CurvOnly *feature.Result
	Metrics  *eval.Metrics
}

// Run executes robust extraction, the naive curvature baseline and the
// curvature-only tangent probe, then scores everything.
func Run(c *cases.Case) *Output {
	opt := feature.DefaultOptions()
	opt.Failed = c.Failed

	robust := feature.ExtractRobust(c.Mesh, c.KappaObs, opt)
	naive := feature.ExtractNaiveCurvature(c.Mesh, c.KappaObs, c.NaiveTau, nil)
	curvOnly := feature.ExtractCurvatureOnly(c.Mesh, c.KappaObs, 0.8)
	m := eval.Evaluate(c, robust, naive, curvOnly)

	return &Output{Case: c, Opt: opt, Robust: robust, Naive: naive, CurvOnly: curvOnly, Metrics: m}
}

// RunAll runs every acceptance case.
func RunAll() []*Output {
	var out []*Output
	for _, c := range cases.All() {
		out = append(out, Run(c))
	}
	return out
}
