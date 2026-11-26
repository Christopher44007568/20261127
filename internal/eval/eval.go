// Package eval compares extracted chains against procedural ground truth and
// quantifies coverage, closure and endpoint errors.
package eval

import (
	"math"

	"featurelines/internal/cases"
	"featurelines/internal/feature"
	"featurelines/internal/linalg"
)

// Vec3 alias.
type Vec3 = linalg.Vec3

// EndpointMark is one detected line endpoint and its measured error.
type EndpointMark struct {
	X, Y, Z float64
	// TargetX.. is the matched ideal endpoint (sharp case).
	TargetX, TargetY, TargetZ float64
	Error                     float64
	// Gap is the distance to the facing endpoint of a broken loop (hole).
	Gap     float64
	PairX   float64
	PairY   float64
	PairZ   float64
	HasPair bool
	Kind    string // "true", "false-break", "robust", "naive"
	Chain   int
}

// Check is one acceptance assertion.
type Check struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail"`
}

// Metrics is the serializable evaluation result for one case.
type Metrics struct {
	CaseID string `json:"case_id"`
	Kind   string `json:"kind"`

	RobustChains int     `json:"robust_chains"`
	RobustLength float64 `json:"robust_length_mm"`
	Coverage     float64 `json:"coverage"`
	ClosedLoop   bool    `json:"closed_loop"`
	BridgeCount  int     `json:"bridge_count"`

	EndpointCount    int            `json:"endpoint_count"`
	MaxEndpointError float64        `json:"max_endpoint_error_mm"`
	ClosureResidual  float64        `json:"closure_residual_mm"`
	Endpoints        []EndpointMark `json:"endpoints"`

	NaiveSegments  int            `json:"naive_segments"`
	NaiveEndpoints int            `json:"naive_endpoints"`
	NaiveGapMax    float64        `json:"naive_gap_max_mm"`
	NaiveMarks     []EndpointMark `json:"naive_marks"`

	CurvOnlySeamLength float64 `json:"curvonly_seam_length_mm"`
	FalseFeatureLength float64 `json:"false_feature_length_mm"`

	Checks []Check `json:"checks"`
	Pass   bool    `json:"pass"`
}

// PointSegDist returns distance from p to segment a-b.
func PointSegDist(p, a, b Vec3) float64 {
	ab := b.Sub(a)
	l2 := ab.Len2()
	if l2 < 1e-18 {
		return p.Dist(a)
	}
	t := p.Sub(a).Dot(ab) / l2
	t = linalg.Clamp(t, 0, 1)
	q := a.Add(ab.Scale(t))
	return p.Dist(q)
}

// ChainPoints returns the polyline segments of a chain as point pairs.
func chainSegs(ch *feature.Chain, pts []Vec3) [][2]Vec3 {
	var segs [][2]Vec3
	vi := ch.Vertices
	for i := 1; i < len(vi); i++ {
		segs = append(segs, [2]Vec3{pts[vi[i-1]], pts[vi[i]]})
	}
	if ch.Closed && len(vi) > 2 {
		segs = append(segs, [2]Vec3{pts[vi[len(vi)-1]], pts[vi[0]]})
	}
	return segs
}

// DistToChains is the minimum distance from p to any chain polyline.
func DistToChains(p Vec3, pts []Vec3, chains []*feature.Chain) float64 {
	best := math.MaxFloat64
	for _, ch := range chains {
		for _, s := range chainSegs(ch, pts) {
			if d := PointSegDist(p, s[0], s[1]); d < best {
				best = d
			}
		}
	}
	return best
}

// chainNear reports whether most of the chain lies within tol of a sampler.
func chainMeanDist(ch *feature.Chain, pts []Vec3, distFn func(Vec3) float64) (mean, max float64) {
	sum := 0.0
	for _, v := range ch.Vertices {
		d := distFn(pts[v])
		sum += d
		max = math.Max(max, d)
	}
	if len(ch.Vertices) > 0 {
		mean = sum / float64(len(ch.Vertices))
	}
	return mean, max
}

// Evaluate computes all metrics and acceptance checks for one case.
func Evaluate(c *cases.Case, robust *feature.Result, naive *feature.Result, curvOnly *feature.Result) *Metrics {
	m := &Metrics{CaseID: c.ID, Kind: c.Kind}

	switch c.Kind {
	case "hole":
		evalHole(c, robust, naive, m)
	case "sharp":
		evalSharp(c, robust, m)
	case "tangent":
		evalTangent(c, robust, curvOnly, m)
	}

	m.Pass = true
	for _, chk := range m.Checks {
		if !chk.OK {
			m.Pass = false
		}
	}
	return m
}
