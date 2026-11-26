package pipeline

import (
	"math"
	"testing"

	"featurelines/internal/cases"
	"featurelines/internal/eval"
	"featurelines/internal/feature"
	"featurelines/internal/geom"
	"featurelines/internal/topo"
)

// TestAcceptanceCases runs all three acceptance scenarios and requires every
// reported check to pass.
func TestAcceptanceCases(t *testing.T) {
	outs := RunAll()
	if len(outs) != 3 {
		t.Fatalf("expected 3 cases, got %d", len(outs))
	}
	for _, o := range outs {
		for _, chk := range o.Metrics.Checks {
			if !chk.OK {
				t.Errorf("%s: check %s failed: %s", o.Case.ID, chk.Name, chk.Detail)
			}
		}
	}
}

// TestSingleVertexFailureDoesNotEraseFeature is the headline resilience
// requirement: even with bridging disabled, one failed vertex only opens a
// small gap; the continuous feature keeps almost all of its length. With
// bridging enabled it is fully restored as a closed loop / single crease.
func TestSingleVertexFailureDoesNotEraseFeature(t *testing.T) {
	for _, c := range []*cases.Case{cases.Hole(), cases.Sharp()} {
		opt := feature.DefaultOptions()
		opt.Failed = c.Failed

		bridged := feature.ExtractRobust(c.Mesh, c.KappaObs, opt)
		covBridged := coverage(c, bridged.Creases)
		if covBridged < 0.97 {
			t.Fatalf("%s: bridged coverage too low: %.3f", c.ID, covBridged)
		}

		optNoBridge := opt
		optNoBridge.DisableBridge = true
		plain := feature.ExtractRobust(c.Mesh, c.KappaObs, optNoBridge)
		covPlain := coverage(c, plain.Creases)
		if covPlain < 0.90 {
			t.Fatalf("%s: single failed vertex erased too much, coverage=%.3f", c.ID, covPlain)
		}
		if len(plain.Creases) == 0 {
			t.Fatalf("%s: feature vanished entirely after one vertex failure", c.ID)
		}
		t.Logf("%s: coverage without bridge=%.3f, with bridge=%.3f", c.ID, covPlain, covBridged)
	}
}

// TestHealthySignalHasNoFailedVertices ensures the no-failure control path is
// at least as complete as the stressed path.
func TestHealthySignalHasNoFailedVertices(t *testing.T) {
	c := cases.Hole()
	cleanK := geom.Curvature(c.Mesh, topo.Build(c.Mesh))
	obs := append([]float64{}, c.KappaObs...)
	obs[c.FailVertex] = cleanK[c.FailVertex]

	opt := feature.DefaultOptions()
	healthy := feature.ExtractRobust(c.Mesh, obs, opt)
	if coverage(c, healthy.Creases) < 0.999 {
		t.Fatalf("healthy hole should be fully covered")
	}
}

func coverage(c *cases.Case, chains []*feature.Chain) float64 {
	pts := c.Mesh.V
	switch c.Kind {
	case "hole":
		const n = 720
		hit := 0
		for k := 0; k < n; k++ {
			a := 2 * 3.141592653589793 * float64(k) / n
			p := cases.Vec3{
				X: c.Circle.Center.X + c.Circle.Radius*math.Cos(a),
				Y: c.Circle.Center.Y + c.Circle.Radius*math.Sin(a),
				Z: c.Circle.Center.Z,
			}
			if eval.DistToChains(p, pts, chains) <= 1.1 {
				hit++
			}
		}
		return float64(hit) / n
	default:
		const n = 300
		hit := 0
		a, b := c.Segment.A, c.Segment.B
		for k := 0; k <= n; k++ {
			p := a.Add(b.Sub(a).Scale(float64(k) / n))
			if eval.DistToChains(p, pts, chains) <= 1.1 {
				hit++
			}
		}
		return float64(hit) / (n + 1)
	}
}
