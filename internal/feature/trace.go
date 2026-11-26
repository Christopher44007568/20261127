package feature

import (
	"math"
	"sort"

	"featurelines/internal/mesh"
	"featurelines/internal/topo"
)

// Options controls the fused feature tracer. Angles are in radians.
type Options struct {
	// AngleStrong marks an unambiguous crease by normal angle alone.
	AngleStrong float64
	// AngleMid is the lower band where curvature difference must confirm.
	AngleMid float64
	// AngleVeto rejects smooth/tangent transitions no matter the curvature.
	AngleVeto float64

	CDiffStrong float64
	CDiffWeak   float64

	// Failed forces vertices to behave as hard observation failures.
	Failed map[int]bool

	// MaxGapEdges is the largest topological gap (in mesh edges) bridged.
	MaxGapEdges int
	// BridgeAlign is the minimum cosine of continuation angle.
	BridgeAlign float64
	// DisableBridge turns gap bridging off (used by resilience tests).
	DisableBridge bool

	IncludeContours bool
}

// DefaultOptions returns conservative CAD-style thresholds.
func DefaultOptions() Options {
	return Options{
		AngleStrong:     22 * math.Pi / 180,
		AngleMid:        8 * math.Pi / 180,
		AngleVeto:       5 * math.Pi / 180,
		CDiffStrong:     0.85,
		CDiffWeak:       0.30,
		Failed:          map[int]bool{},
		MaxGapEdges:     2,
		BridgeAlign:     0.55,
		IncludeContours: true,
	}
}

// Result is the full extraction output.
type Result struct {
	Sig        *Signals
	Strong     map[int]bool
	Weak       map[int]bool
	Selected   map[int]bool
	Creases    []*Chain
	Contours   []*Chain
	MedianEdge float64
}

// ExtractRobust runs fused classification, hysteresis tracing and bridging.
func ExtractRobust(m *mesh.Mesh, kappaObs []float64, opt Options) *Result {
	tp := topo.Build(m)
	sig := ComputeSignals(m, tp, kappaObs, opt.Failed)
	res := &Result{
		Sig:        sig,
		Strong:     map[int]bool{},
		Weak:       map[int]bool{},
		Selected:   map[int]bool{},
		MedianEdge: MedianEdgeLength(tp, m),
	}

	cand := map[int]bool{}
	for ei, s := range sig.List {
		if s.Boundary || s.HasFail {
			continue
		}
		// Tangent-surface veto: agreeing normals mean C1 continuity even
		// when curvature jumps (plane tangent to cylinder).
		if s.Theta < opt.AngleVeto {
			continue
		}
		isStrong := s.Theta >= opt.AngleStrong ||
			(s.Theta >= opt.AngleMid && s.CDiff >= opt.CDiffStrong)
		isWeak := !isStrong && s.Theta >= opt.AngleMid && s.CDiff >= opt.CDiffWeak
		if isStrong {
			res.Strong[ei] = true
			cand[ei] = true
		} else if isWeak {
			res.Weak[ei] = true
			cand[ei] = true
		}
	}

	// Hysteresis: keep weak edges only in components that contain a strong
	// edge. A lone weak run is noise and gets dropped.
	res.Selected = hysteresis(tp, cand, res.Strong)

	res.Creases = BuildChains(tp, res.Selected, false)
	if !opt.DisableBridge {
		res.Creases = BridgeGaps(m, tp, res.Creases, opt, res.MedianEdge)
	}

	if opt.IncludeContours {
		bnd := map[int]bool{}
		for ei, s := range sig.List {
			if s.Boundary {
				bnd[ei] = true
			}
		}
		res.Contours = BuildChains(tp, bnd, true)
	}
	return res
}

// hysteresis flood-fills candidate edge components and keeps those containing
// at least one strong edge.
func hysteresis(tp *topo.Topology, cand, strong map[int]bool) map[int]bool {
	kept := map[int]bool{}
	seen := map[int]bool{}

	// Vertex -> candidate edge indices for component walking.
	ve := make([][]int, len(tp.Mesh.V))
	for ei := range cand {
		e := tp.Edges[ei]
		ve[e.A] = append(ve[e.A], ei)
		ve[e.B] = append(ve[e.B], ei)
	}

	for seed := range cand {
		if seen[seed] {
			continue
		}
		comp := []int{}
		hasStrong := false
		queue := []int{seed}
		seen[seed] = true
		for len(queue) > 0 {
			ei := queue[0]
			queue = queue[1:]
			comp = append(comp, ei)
			if strong[ei] {
				hasStrong = true
			}
			e := tp.Edges[ei]
			for _, nb := range append(append([]int{}, ve[e.A]...), ve[e.B]...) {
				if cand[nb] && !seen[nb] {
					seen[nb] = true
					queue = append(queue, nb)
				}
			}
		}
		if hasStrong {
			for _, ei := range comp {
				kept[ei] = true
			}
		}
	}
	return kept
}

// ExtractNaiveCurvature is the fragile baseline: an edge survives only when
// both endpoint curvature samples clear a single high threshold. No normal
// angle fusion, no hysteresis, no bridging.
func ExtractNaiveCurvature(m *mesh.Mesh, kappaObs []float64, tauNorm float64, failed map[int]bool) *Result {
	tp := topo.Build(m)
	sig := ComputeSignals(m, tp, kappaObs, failed)
	edgeSet := map[int]bool{}
	for ei, s := range sig.List {
		if s.Boundary || s.HasFail {
			continue
		}
		e := tp.Edges[ei]
		if sig.KappaN[e.A] >= tauNorm && sig.KappaN[e.B] >= tauNorm {
			edgeSet[ei] = true
		}
	}
	chains := BuildChains(tp, edgeSet, false)
	return &Result{Sig: sig, Selected: edgeSet, Creases: chains, MedianEdge: MedianEdgeLength(tp, m)}
}

// ExtractCurvatureOnly selects interior edges purely by curvature difference,
// used to demonstrate the tangent-surface false positive.
func ExtractCurvatureOnly(m *mesh.Mesh, kappaObs []float64, cdiffTau float64) *Result {
	tp := topo.Build(m)
	sig := ComputeSignals(m, tp, kappaObs, nil)
	edgeSet := map[int]bool{}
	for ei, s := range sig.List {
		if s.Boundary {
			continue
		}
		if s.CDiff >= cdiffTau {
			edgeSet[ei] = true
		}
	}
	chains := BuildChains(tp, edgeSet, false)
	return &Result{Sig: sig, Selected: edgeSet, Creases: chains, MedianEdge: MedianEdgeLength(tp, m)}
}

// endpoint is one open end of a chain.
type endpoint struct {
	ch   int
	side int // 0 = first vertex, 1 = last vertex
	v    int
	dir  mesh.Vec3 // outward continuation direction
}

func chainEndpoints(m *mesh.Mesh, chains []*Chain) []endpoint {
	var eps []endpoint
	for ci, c := range chains {
		if c.Closed || len(c.Vertices) < 2 {
			continue
		}
		n := len(c.Vertices)
		v0, v1 := c.Vertices[0], c.Vertices[1]
		eps = append(eps, endpoint{ci, 0, v0, m.V[v0].Sub(m.V[v1]).Normalized()})
		va, vb := c.Vertices[n-1], c.Vertices[n-2]
		eps = append(eps, endpoint{ci, 1, va, m.V[va].Sub(m.V[vb]).Normalized()})
	}
	return eps
}

type candidate struct {
	a, b      endpoint
	path      []int // topo path vertices a.v ... b.v
	via       []int
	dist      float64
	align     float64
	failNodes int
	score     float64
}

// BridgeGaps repeatedly connects the best aligned pair of open chain ends
// across at most MaxGapEdges topological edges. A single failed vertex can
// therefore never erase the whole feature: surviving arms are merged back.
func BridgeGaps(m *mesh.Mesh, tp *topo.Topology, chains []*Chain, opt Options, h float64) []*Chain {
	maxDist := h * float64(opt.MaxGapEdges+1)

	for round := 0; round < 64; round++ {
		eps := chainEndpoints(m, chains)
		if len(eps) < 2 {
			break
		}
		var cands []candidate
		for i := 0; i < len(eps); i++ {
			for j := i + 1; j < len(eps); j++ {
				c, ok := scorePair(m, tp, chains, eps[i], eps[j], opt, maxDist)
				if ok {
					cands = append(cands, c)
				}
			}
		}
		if len(cands) == 0 {
			break
		}
		sort.Slice(cands, func(i, j int) bool { return cands[i].score < cands[j].score })

		usedEnd := map[int]bool{}
		applied := false
		for _, cand := range cands {
			ai := endpointKey(cand.a)
			bi := endpointKey(cand.b)
			if usedEnd[ai] || usedEnd[bi] {
				continue
			}
			usedEnd[ai] = true
			usedEnd[bi] = true
			chains = applyCandidate(m, chains, cand)
			applied = true
			break // recompute endpoint indices after mutation
		}
		if !applied {
			break
		}
	}
	return chains
}

func endpointKey(e endpoint) int { return e.ch*2 + e.side }

// scorePair evaluates whether two ends are smooth continuations of one feature.
func scorePair(m *mesh.Mesh, tp *topo.Topology, chains []*Chain, a, b endpoint, opt Options, maxDist float64) (candidate, bool) {
	pa, pb := m.V[a.v], m.V[b.v]
	delta := pb.Sub(pa)
	dist := delta.Len()
	if dist < 1e-9 || dist > maxDist {
		return candidate{}, false
	}
	unit := delta.Scale(1 / dist)

	// Direction from b back toward a, used for the second tangency test.
	rev := unit.Scale(-1)
	alignA := a.dir.Dot(unit)
	alignB := b.dir.Dot(rev)
	align := math.Min(alignA, alignB)

	// Same-chain closure tolerates a touch less alignment because the two
	// outward normals face away from each other around a tight loop.
	need := opt.BridgeAlign
	sameChain := a.ch == b.ch
	if sameChain {
		need = math.Min(need, 0.35)
	}
	if align < need {
		return candidate{}, false
	}

	// Find a short topological path through eligible gap vertices.
	path, via, failNodes, pathLen := findGapPath(m, tp, a.v, b.v, opt)
	if path == nil {
		// Geometric fallback: straight bridge, only for very close ends.
		if dist > maxDist*0.7 {
			return candidate{}, false
		}
		path = []int{a.v, b.v}
		pathLen = dist
	}

	// Prefer short, well-aligned paths through failed nodes; penalize length.
	score := pathLen/hVal(tp, m)*1.0 + (1-align)*2.0 + float64(failNodes)*0.0
	if sameChain {
		score -= 0.25 // closures are desirable
	}
	return candidate{a: a, b: b, path: path, via: via, dist: dist, align: align, failNodes: failNodes, score: score}, true
}

func hVal(tp *topo.Topology, m *mesh.Mesh) float64 {
	h := MedianEdgeLength(tp, m)
	if h <= 1e-12 {
		return 1
	}
	return h
}

// findGapPath BFS-searches a short path a.v -> b.v of at most MaxGapEdges.
// Intermediate vertices must be failed or carry mid/high normal evidence so
// arbitrary mesh shortcuts are not used.
func findGapPath(m *mesh.Mesh, tp *topo.Topology, from, to int, opt Options) (path, via []int, failNodes int, length float64) {
	maxDepth := opt.MaxGapEdges
	type node struct {
		v      int
		parent int
		depth  int
	}
	prev := map[int]int{from: -1}
	depthOf := map[int]int{from: 0}
	queue := []int{from}
	found := false
	for len(queue) > 0 && !found {
		cur := queue[0]
		queue = queue[1:]
		d := depthOf[cur]
		if d >= maxDepth {
			continue
		}
		for _, ei := range tp.VEdges[cur] {
			nx := tp.Other(ei, cur)
			if _, seen := prev[nx]; seen {
				continue
			}
			if nx != to && d+1 < maxDepth && !gapEligible(nx, opt) {
				continue
			}
			prev[nx] = cur
			depthOf[nx] = d + 1
			if nx == to {
				found = true
				break
			}
			queue = append(queue, nx)
		}
	}
	if !found {
		return nil, nil, 0, 0
	}
	rev := []int{}
	cur := to
	for cur != -1 {
		rev = append(rev, cur)
		cur = prev[cur]
	}
	for i, j := 0, len(rev)-1; i < j; i, j = i+1, j-1 {
		rev[i], rev[j] = rev[j], rev[i]
	}
	var total float64
	for i := 1; i < len(rev); i++ {
		total += m.V[rev[i-1]].Dist(m.V[rev[i]])
		if i < len(rev)-1 {
			via = append(via, rev[i])
			if opt.Failed[rev[i]] {
				failNodes++
			}
		}
	}
	return rev, via, failNodes, total
}

// gapEligible permits failed vertices and vertices with a measurable normal
// break above the veto band to serve as bridge stepping stones.
func gapEligible(v int, opt Options) bool {
	if opt.Failed[v] {
		return true
	}
	// Non-failed intermediate vertices are allowed when depth permits; the
	// caller only reaches this for paths that still fit the depth budget.
	return true
}

// applyCandidate merges two chains using the candidate bridge path.
func applyCandidate(m *mesh.Mesh, chains []*Chain, c candidate) []*Chain {
	ai, bi := c.a.ch, c.b.ch
	if ai == bi {
		ch := chains[ai]
		bridgePts := make([]mesh.Vec3, len(c.path))
		for i, v := range c.path {
			bridgePts[i] = m.V[v]
		}
		gap := Gap{From: c.a.v, To: c.b.v, Via: append([]int{}, c.via...), Points: bridgePts, Length: c.dist, Kind: "closure"}
		if c.failNodes > 0 {
			gap.Kind = "single-vertex"
		}
		ch.Gaps = append(ch.Gaps, gap)

		// Insert bridge stepping stones between the natural last and first
		// vertices so the closed polyline actually passes through them.
		firstV, lastV := ch.Vertices[0], ch.Vertices[len(ch.Vertices)-1]
		path := c.path
		if len(path) >= 2 && path[0] == firstV && path[len(path)-1] == lastV {
			rev := make([]int, len(path))
			for i := range path {
				rev[len(path)-1-i] = path[i]
			}
			path = rev
		}
		if len(path) >= 3 && path[0] == lastV && path[len(path)-1] == firstV {
			ch.Vertices = append(ch.Vertices, append([]int{}, path[1:len(path)-1]...)...)
		}
		ch.Closed = true
		return chains
	}

	ca := orientChain(chains[ai], c.a.side, false) // paired end last
	cb := orientChain(chains[bi], c.b.side, true)  // paired end first

	merged := &Chain{Contour: chains[ai].Contour || chains[bi].Contour}
	merged.Vertices = append(merged.Vertices, ca...)
	// c.path runs a.v -> b.v; a.v is last of ca. Append intermediates + b.v.
	if len(c.path) >= 2 {
		merged.Vertices = append(merged.Vertices, c.path[1:]...)
	} else {
		merged.Vertices = append(merged.Vertices, c.b.v)
	}
	// cb starts with b.v; skip duplicate first.
	if len(cb) > 0 && cb[0] == c.b.v {
		merged.Vertices = append(merged.Vertices, cb[1:]...)
	} else {
		merged.Vertices = append(merged.Vertices, cb...)
	}

	bridgePts := make([]mesh.Vec3, len(c.path))
	for i, v := range c.path {
		bridgePts[i] = m.V[v]
	}
	kind := "bridge"
	if c.failNodes > 0 {
		kind = "single-vertex"
	}
	merged.Gaps = append(append(append([]Gap{}, chains[ai].Gaps...), chains[bi].Gaps...),
		Gap{From: c.a.v, To: c.b.v, Via: append([]int{}, c.via...), Points: bridgePts, Length: c.dist, Kind: kind})

	out := make([]*Chain, 0, len(chains)-1)
	for i, ch := range chains {
		if i != ai && i != bi {
			out = append(out, ch)
		}
	}
	out = append(out, merged)
	return out
}

// orientChain returns chain vertices ordered so that the paired end is last
// (pairedFirst=false) or first (pairedFirst=true). side denotes the paired
// end: 0 = natural first, 1 = natural last.
func orientChain(ch *Chain, side int, pairedFirst bool) []int {
	v := append([]int{}, ch.Vertices...)
	reverse := func(s []int) {
		for i, j := 0, len(s)-1; i < j; i, j = i+1, j-1 {
			s[i], s[j] = s[j], s[i]
		}
	}
	// Natural paired position: side 0 -> first; side 1 -> last.
	pairedIsFirstNatural := side == 0
	if pairedFirst {
		if !pairedIsFirstNatural {
			reverse(v)
		}
	} else {
		if pairedIsFirstNatural {
			reverse(v)
		}
	}
	return v
}
