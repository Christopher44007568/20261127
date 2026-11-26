package eval

import (
	"fmt"
	"math"

	"featurelines/internal/cases"
	"featurelines/internal/feature"
	"featurelines/internal/topo"
)

func circleSampler(c *cases.Case) (func(Vec3) float64, func(int) Vec3) {
	ci := c.Circle
	dist := func(p Vec3) float64 {
		dz := p.Z - ci.Center.Z
		rad := math.Hypot(p.X-ci.Center.X, p.Y-ci.Center.Y)
		return math.Hypot(rad-ci.Radius, dz)
	}
	at := func(k int) Vec3 {
		a := 2 * math.Pi * float64(k) / 720
		return Vec3{X: ci.Center.X + ci.Radius*math.Cos(a), Y: ci.Center.Y + ci.Radius*math.Sin(a), Z: ci.Center.Z}
	}
	return dist, at
}

func evalHole(c *cases.Case, robust *feature.Result, naive *feature.Result, m *Metrics) {
	pts := c.Mesh.V
	distCircle, at := circleSampler(c)

	const samples = 720
	hit := 0
	const covTol = 1.1
	for k := 0; k < samples; k++ {
		p := at(k)
		if DistToChains(p, pts, robust.Creases) <= covTol {
			hit++
		}
	}
	m.Coverage = float64(hit) / samples
	m.RobustChains = len(robust.Creases)

	var best *feature.Chain
	bestMean := math.MaxFloat64
	nearChains := []*feature.Chain{}
	for _, ch := range robust.Creases {
		mean, mx := chainMeanDist(ch, pts, distCircle)
		if mx < 3.0 {
			nearChains = append(nearChains, ch)
		}
		if mean < bestMean {
			bestMean = mean
			best = ch
		}
	}
	if best != nil && bestMean > 3.0 {
		best = nil
	}

	if best != nil {
		m.ClosedLoop = best.Closed
		m.BridgeCount = len(best.Gaps)
		m.RobustLength = best.Length(c.Mesh)
		if best.Closed {
			m.ClosureResidual = 0
		} else if len(best.Vertices) >= 2 {
			m.ClosureResidual = pts[best.Vertices[0]].Dist(pts[best.Vertices[len(best.Vertices)-1]])
		}
	}

	// False endpoints: any open end of a circle-near robust chain.
	for ci, ch := range nearChains {
		if ch.Closed || len(ch.Vertices) < 2 {
			continue
		}
		for _, vi := range []int{ch.Vertices[0], ch.Vertices[len(ch.Vertices)-1]} {
			p := pts[vi]
			m.Endpoints = append(m.Endpoints, EndpointMark{
				X: p.X, Y: p.Y, Z: p.Z, Error: distCircle(p),
				Kind: "false-break", Chain: ci,
			})
		}
	}
	m.EndpointCount = len(m.Endpoints)

	// Naive baseline fragments and paired break gaps.
	type ne struct {
		mark EndpointMark
	}
	var naiveEnds []EndpointMark
	segments := 0
	for ci, ch := range naive.Creases {
		_, mx := chainMeanDist(ch, pts, distCircle)
		if mx >= 2.5 || len(ch.Vertices) < 2 {
			continue
		}
		segments++
		for _, vi := range []int{ch.Vertices[0], ch.Vertices[len(ch.Vertices)-1]} {
			p := pts[vi]
			naiveEnds = append(naiveEnds, EndpointMark{
				X: p.X, Y: p.Y, Z: p.Z, Error: distCircle(p), Kind: "naive", Chain: ci,
			})
		}
	}
	// Greedy closest pairing of break endpoints.
	used := make([]bool, len(naiveEnds))
	for i := range naiveEnds {
		if used[i] {
			continue
		}
		bestJ, bestD := -1, math.MaxFloat64
		for j := i + 1; j < len(naiveEnds); j++ {
			if used[j] {
				continue
			}
			if naiveEnds[i].Chain == naiveEnds[j].Chain {
				continue // a chain's own ends are already connected
			}
			pi := Vec3{X: naiveEnds[i].X, Y: naiveEnds[i].Y, Z: naiveEnds[i].Z}
			pj := Vec3{X: naiveEnds[j].X, Y: naiveEnds[j].Y, Z: naiveEnds[j].Z}
			if d := pi.Dist(pj); d < bestD {
				bestD, bestJ = d, j
			}
		}
		if bestJ >= 0 && bestD < 4.0 {
			used[i], used[bestJ] = true, true
			naiveEnds[i].Gap = bestD
			naiveEnds[bestJ].Gap = bestD
			naiveEnds[i].HasPair = true
			naiveEnds[bestJ].HasPair = true
			naiveEnds[i].PairX, naiveEnds[i].PairY, naiveEnds[i].PairZ =
				naiveEnds[bestJ].X, naiveEnds[bestJ].Y, naiveEnds[bestJ].Z
			naiveEnds[bestJ].PairX, naiveEnds[bestJ].PairY, naiveEnds[bestJ].PairZ =
				naiveEnds[i].X, naiveEnds[i].Y, naiveEnds[i].Z
			m.NaiveGapMax = math.Max(m.NaiveGapMax, bestD)
		}
	}
	m.NaiveMarks = naiveEnds
	m.NaiveSegments = segments
	m.NaiveEndpoints = len(naiveEnds)

	m.Checks = []Check{
		{Name: "hole/closed_loop", OK: m.ClosedLoop, Detail: "robust tracer yields one closed rim loop"},
		{Name: "hole/coverage_ge_97pct", OK: m.Coverage >= 0.97, Detail: fmt.Sprintf("coverage=%.3f", m.Coverage)},
		{Name: "hole/no_false_endpoints", OK: m.EndpointCount == 0, Detail: fmt.Sprintf("false endpoints=%d", m.EndpointCount)},
		{Name: "hole/failed_vertex_bridged", OK: m.BridgeCount >= 1, Detail: fmt.Sprintf("bridges=%d", m.BridgeCount)},
		{Name: "baseline/high_threshold_fragments", OK: m.NaiveSegments >= 3, Detail: fmt.Sprintf("naive segments=%d endpoints=%d", m.NaiveSegments, m.NaiveEndpoints)},
	}
}

func evalSharp(c *cases.Case, robust *feature.Result, m *Metrics) {
	pts := c.Mesh.V
	a, b := c.Segment.A, c.Segment.B
	distSeg := func(p Vec3) float64 { return PointSegDist(p, a, b) }

	const samples = 300
	hit := 0
	const covTol = 1.1
	for k := 0; k <= samples; k++ {
		t := float64(k) / samples
		p := a.Add(b.Sub(a).Scale(t))
		if DistToChains(p, pts, robust.Creases) <= covTol {
			hit++
		}
	}
	m.Coverage = float64(hit) / (samples + 1)

	var best *feature.Chain
	bestMean := math.MaxFloat64
	for _, ch := range robust.Creases {
		mean, _ := chainMeanDist(ch, pts, distSeg)
		if mean < bestMean {
			bestMean, best = mean, ch
		}
	}

	if best != nil && bestMean < 3.0 {
		m.RobustChains = 1
		m.RobustLength = best.Length(c.Mesh)
		m.BridgeCount = len(best.Gaps)
		p0 := pts[best.Vertices[0]]
		p1 := pts[best.Vertices[len(best.Vertices)-1]]
		if best.Closed {
			m.ClosedLoop = true
		} else {
			direct := p0.Dist(a) + p1.Dist(b)
			swapped := p0.Dist(b) + p1.Dist(a)
			var ea, eb float64
			var ta, tb Vec3
			if direct <= swapped {
				ea, eb = p0.Dist(a), p1.Dist(b)
				ta, tb = a, b
			} else {
				ea, eb = p0.Dist(b), p1.Dist(a)
				ta, tb = b, a
			}
			m.MaxEndpointError = math.Max(ea, eb)
			m.EndpointCount = 2
			m.Endpoints = []EndpointMark{
				{X: p0.X, Y: p0.Y, Z: p0.Z, TargetX: ta.X, TargetY: ta.Y, TargetZ: ta.Z, Error: ea, Kind: "true"},
				{X: p1.X, Y: p1.Y, Z: p1.Z, TargetX: tb.X, TargetY: tb.Y, TargetZ: tb.Z, Error: eb, Kind: "true"},
			}
		}
	}

	m.Checks = []Check{
		{Name: "sharp/coverage_ge_95pct", OK: m.Coverage >= 0.95, Detail: fmt.Sprintf("coverage=%.3f", m.Coverage)},
		{Name: "sharp/two_endpoints", OK: m.EndpointCount == 2, Detail: fmt.Sprintf("endpoints=%d", m.EndpointCount)},
		{Name: "sharp/endpoint_error_le_1p2mm", OK: m.MaxEndpointError <= 1.2, Detail: fmt.Sprintf("max endpoint error=%.3f mm", m.MaxEndpointError)},
		{Name: "sharp/failed_vertex_bridged", OK: m.BridgeCount >= 1, Detail: fmt.Sprintf("bridges=%d", m.BridgeCount)},
	}
}

func evalTangent(c *cases.Case, robust *feature.Result, curvOnly *feature.Result, m *Metrics) {
	pts := c.Mesh.V
	a, b := c.Seam.A, c.Seam.B
	distSeam := func(p Vec3) float64 { return PointSegDist(p, a, b) }

	m.RobustChains = len(robust.Creases)
	falseLen := 0.0
	for _, ch := range robust.Creases {
		mean, _ := chainMeanDist(ch, pts, distSeam)
		if mean < 3.0 {
			falseLen += ch.Length(c.Mesh)
		}
	}
	m.FalseFeatureLength = falseLen

	curvLen := 0.0
	fired := 0
	if curvOnly != nil {
		tp := curvOnly.Sig.Topo
		seamVerts := c.SeamVerts
		for j := 0; j+1 < len(seamVerts); j++ {
			va, vb := seamVerts[j], seamVerts[j+1]
			ei, ok := tp.EdgeIndex[topo.Key(va, vb)]
			if !ok || !curvOnly.Selected[ei] {
				continue
			}
			curvLen += c.Mesh.V[va].Dist(c.Mesh.V[vb])
			fired++
		}
	}
	m.CurvOnlySeamLength = curvLen

	m.Checks = []Check{
		{Name: "tangent/no_false_crease", OK: falseLen <= 1.0 && m.RobustChains == 0,
			Detail: fmt.Sprintf("robust crease chains=%d false length=%.2f mm", m.RobustChains, falseLen)},
		{Name: "tangent/curvature_only_flags_seam", OK: curvLen >= 15.0,
			Detail: fmt.Sprintf("curvature-only fires on %d/%d seam edges, %.1f mm", fired, len(c.SeamVerts)-1, curvLen)},
	}
}
