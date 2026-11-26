package main

import (
	"math"
)

type JoinStyle string

const (
	JoinMiter JoinStyle = "miter"
	JoinBevel JoinStyle = "bevel"
	JoinRound JoinStyle = "round"
)

// ExpandOptions controls distance expansion and local corner correction.
type ExpandOptions struct {
	MiterLimit   float64
	Join         JoinStyle
	ArcSteps     int
	MinArea      float64
	SearchPasses int
	ValidateNest bool
}

func DefaultExpandOptions() ExpandOptions {
	return ExpandOptions{
		MiterLimit:   3.2,
		Join:         JoinMiter,
		ArcSteps:     6,
		MinArea:      1e-6,
		SearchPasses: 42,
		ValidateNest: true,
	}
}

// ExpansionResult records both the requested and topology-safe distance.
type ExpansionResult struct {
	Outline          Outline        `json:"outline"`
	RequestedDelta   float64        `json:"requested_delta"`
	Delta            float64        `json:"delta"`
	Clearance        float64        `json:"clearance"`
	DistanceBudget   float64        `json:"distance_budget"`
	Intersections    []Intersection `json:"intersections"`
	Valid            bool           `json:"valid"`
	Reduced          bool           `json:"reduced"`
	SearchPasses     int            `json:"search_passes"`
	CorrectedCorners int            `json:"corrected_corners"`
	CornerActions    []string       `json:"corner_actions,omitempty"`
}

func lineIntersection(p, dirP, q, dirQ Vec2) (point Vec2, t, u float64, ok bool) {
	den := dirP.Cross(dirQ)
	if math.Abs(den) < 1e-12 {
		return Vec2{}, 0, 0, false
	}
	qp := q.Sub(p)
	t = qp.Cross(dirQ) / den
	u = qp.Cross(dirP) / den
	return p.Add(dirP.Scale(t)), t, u, true
}

func miterPoint(c Contour, i int, delta float64) (point Vec2, ratio, tPrev, tNext, prevLen, nextLen float64, ok bool) {
	n := len(c)
	prev := c[(i-1+n)%n]
	cur := c[i]
	next := c[(i+1)%n]
	uPrev := cur.Sub(prev).Normalized()
	uNext := next.Sub(cur).Normalized()
	if uPrev.Len2() == 0 || uNext.Len2() == 0 {
		return Vec2{}, 0, 0, 0, 0, 0, false
	}
	nPrev := uPrev.RightNormal()
	nNext := uNext.RightNormal()
	lineA := cur.Add(nPrev.Scale(delta))
	lineB := cur.Add(nNext.Scale(delta))
	p, tPrev, tNext, ok := lineIntersection(lineA, uPrev, lineB, uNext)
	if !ok {
		return Vec2{}, 0, 0, 0, 0, 0, false
	}
	ratio = p.Sub(cur).Len() / math.Max(delta, 1e-12)
	return p, ratio, tPrev, tNext, cur.Sub(prev).Len(), next.Sub(cur).Len(), true
}

// OffsetContour applies a Euclidean boundary distance using right normals.
// Canonical outlines have material on the left, so right is expansion.
func OffsetContour(c Contour, delta float64, opts ExpandOptions) (Contour, []string, bool) {
	if delta <= 0 {
		return c.Clone(), nil, true
	}
	limit := opts.MiterLimit
	if limit <= 0 {
		limit = 3.2
	}
	actions := make([]string, 0, len(c))
	out := make(Contour, 0, len(c)*2)
	n := len(c)
	localSafe := true

	for i := 0; i < n; i++ {
		cur := c[i]
		prev := c[(i-1+n)%n]
		next := c[(i+1)%n]
		uPrev := cur.Sub(prev).Normalized()
		uNext := next.Sub(cur).Normalized()
		nPrev := uPrev.RightNormal()
		nNext := uNext.RightNormal()
		inEnd := cur.Add(nPrev.Scale(delta))
		outStart := cur.Add(nNext.Scale(delta))

		if opts.Join == JoinBevel {
			out = append(out, inEnd, outStart)
			actions = append(actions, "bevel")
			continue
		}

		mp, ratio, tPrev, tNext, prevLen, nextLen, ok := miterPoint(c, i, delta)
		innerGeometry := ok && tPrev <= 1e-7 && tNext >= -1e-7
		insideAdjacentEdges := tPrev >= -prevLen-1e-6 && tNext <= nextLen+1e-6
		innerClip := innerGeometry && insideAdjacentEdges
		outerMiter := ok && tPrev >= -1e-7 && tNext <= 1e-7
		if innerGeometry && !insideAdjacentEdges {
			localSafe = false
		}

		if opts.Join == JoinMiter && ok && innerClip {
			out = append(out, mp)
			actions = append(actions, "clip")
			continue
		}

		useMiter := opts.Join == JoinMiter && ok && outerMiter && ratio <= limit && ratio > 0
		if useMiter {
			out = append(out, mp)
			actions = append(actions, "miter")
			continue
		}

		if opts.Join == JoinRound && ok && !innerClip {
			arc := rightNormalArc(cur, delta, nPrev, nNext, opts.ArcSteps)
			out = append(out, arc...)
			actions = append(actions, "round")
			continue
		}

		if innerClip {
			out = append(out, mp)
			actions = append(actions, "clip")
		} else {
			out = append(out, inEnd, outStart)
			actions = append(actions, "bevel")
		}
	}
	return CleanContour(out, 1e-8), actions, localSafe
}

func rightNormalArc(center Vec2, radius float64, from, to Vec2, steps int) []Vec2 {
	if steps < 2 {
		steps = 6
	}
	a0 := math.Atan2(from.Y, from.X)
	a1 := math.Atan2(to.Y, to.X)
	diff := a1 - a0
	for diff > math.Pi {
		diff -= 2 * math.Pi
	}
	for diff < -math.Pi {
		diff += 2 * math.Pi
	}
	points := make([]Vec2, 0, steps+1)
	for k := 0; k <= steps; k++ {
		a := a0 + diff*float64(k)/float64(steps)
		points = append(points, center.Add(Vec2{X: math.Cos(a) * radius, Y: math.Sin(a) * radius}))
	}
	return points
}

// NaiveMiterContour deliberately uses unlimited miter intersections. It is
// used for diagnostics and self-intersection highlighting.
func NaiveMiterContour(c Contour, delta float64) Contour {
	if delta <= 0 {
		return c.Clone()
	}
	out := make(Contour, 0, len(c))
	n := len(c)
	for i := 0; i < n; i++ {
		cur := c[i]
		prev := c[(i-1+n)%n]
		next := c[(i+1)%n]
		uPrev := cur.Sub(prev).Normalized()
		uNext := next.Sub(cur).Normalized()
		nPrev := uPrev.RightNormal()
		nNext := uNext.RightNormal()
		lineA := cur.Add(nPrev.Scale(delta))
		lineB := cur.Add(nNext.Scale(delta))
		if p, _, _, ok := lineIntersection(lineA, uPrev, lineB, uNext); ok {
			out = append(out, p)
		} else {
			out = append(out, cur.Add(nNext.Scale(delta)))
		}
	}
	return CleanContour(out, 1e-8)
}

func offsetOutlineAt(o Outline, delta float64, opts ExpandOptions) (Outline, int, []string, bool) {
	result := Outline{Outer: make(Contour, 0), Holes: make([]Contour, len(o.Holes))}
	allActions := make([]string, 0)
	corrected := 0
	safe := true

	outer, actions, outerSafe := OffsetContour(o.Outer, delta, opts)
	result.Outer = outer
	safe = safe && outerSafe
	for _, a := range actions {
		allActions = append(allActions, "outer:"+a)
		if a != "miter" {
			corrected++
		}
	}

	for i, h := range o.Holes {
		hc, actions, holeSafe := OffsetContour(h, delta, opts)
		result.Holes[i] = hc
		safe = safe && holeSafe
		for _, a := range actions {
			allActions = append(allActions, "hole:"+a)
			if a != "miter" {
				corrected++
			}
		}
	}
	return result, corrected, allActions, safe
}

func expansionValid(base, candidate Outline, localSafe bool, opts ExpandOptions) ([]Intersection, bool) {
	intersections := FindOutlineIntersections(candidate)
	if len(intersections) != 0 {
		return intersections, false
	}
	if !localSafe {
		return intersections, false
	}
	if candidate.Outer.SignedArea() <= opts.MinArea {
		return intersections, false
	}
	for _, h := range candidate.Holes {
		if h.SignedArea() >= -opts.MinArea {
			return intersections, false
		}
	}
	if opts.ValidateNest {
		want := base.NestingSignature()
		got := candidate.NestingSignature()
		if len(want) != len(got) {
			return intersections, false
		}
		for i := range want {
			if want[i] != got[i] {
				return intersections, false
			}
		}
	}
	return intersections, true
}

// ExpandOutline expands filled strokes by delta design units. A distance
// clearance estimate and exact topology validation keep counters open. If
// the requested delta is unsafe, binary search finds the largest safe delta.
func ExpandOutline(input Outline, delta float64, opts ...ExpandOptions) ExpansionResult {
	cfg := DefaultExpandOptions()
	if len(opts) > 0 {
		cfg = opts[0]
	}
	base := input.Canonical()
	if cfg.SearchPasses <= 0 {
		cfg.SearchPasses = DefaultExpandOptions().SearchPasses
	}

	clearance := math.Inf(1)
	for _, c := range base.Contours() {
		if d := MinimumNonAdjacentDistance(c); !math.IsInf(d, 1) {
			clearance = math.Min(clearance, d)
		}
	}
	clearanceValue := -1.0
	budget := -1.0
	if !math.IsInf(clearance, 1) {
		clearanceValue = clearance
		budget = clearance * 0.45
	}

	makeResult := func(d float64, passes int) ExpansionResult {
		expanded, corrected, actions, localSafe := offsetOutlineAt(base, d, cfg)
		intersections, valid := expansionValid(base, expanded, localSafe, cfg)
		return ExpansionResult{
			Outline:          expanded,
			RequestedDelta:   delta,
			Delta:            d,
			Clearance:        clearanceValue,
			DistanceBudget:   budget,
			Intersections:    intersections,
			Valid:            valid,
			Reduced:          d+1e-9 < delta,
			SearchPasses:     passes,
			CorrectedCorners: corrected,
			CornerActions:    actions,
		}
	}

	if delta <= 0 {
		r := makeResult(0, 0)
		r.Valid = true
		return r
	}

	requested := makeResult(delta, 0)
	if requested.Valid {
		return requested
	}

	lo, hi := 0.0, delta
	best := makeResult(0, 0)
	for pass := 0; pass < cfg.SearchPasses; pass++ {
		mid := (lo + hi) * 0.5
		candidate := makeResult(mid, pass+1)
		if candidate.Valid {
			lo = mid
			best = candidate
		} else {
			hi = mid
		}
	}
	final := makeResult(lo, cfg.SearchPasses)
	if final.Valid {
		final.Reduced = lo+1e-7 < delta
		return final
	}
	best.SearchPasses = cfg.SearchPasses
	best.Reduced = true
	return best
}
