package holefill

import "math"

func ClassifyLoops(m *Mesh, loops []*BoundaryLoop) {
	if len(loops) == 0 {
		return
	}
	outer := 0
	for i, l := range loops {
		l.Class = ClassUnknown
		if l.Metrics.Area > loops[outer].Metrics.Area {
			outer = i
		}
	}
	loops[outer].Class = ClassOuter
	loops[outer].Metrics.Reason = "largest projected boundary loop; retained as open scan contour"

	for i, l := range loops {
		if i == outer {
			continue
		}
		l.Metrics.MinOuterGap = loopDistanceWorldXY(m, l, loops[outer])
		l.Metrics.MinPairGap = math.Inf(1)
	}

	type pairRelation struct {
		a, b int
		gap  float64
	}
	var pairs []pairRelation
	for i := 0; i < len(loops); i++ {
		if i == outer {
			continue
		}
		for j := i + 1; j < len(loops); j++ {
			if j == outer {
				continue
			}
			gap := loopDistanceWorldXY(m, loops[i], loops[j])
			pairs = append(pairs, pairRelation{i, j, gap})
			if gap < loops[i].Metrics.MinPairGap {
				loops[i].Metrics.MinPairGap = gap
			}
			if gap < loops[j].Metrics.MinPairGap {
				loops[j].Metrics.MinPairGap = gap
			}
		}
	}

	parent := make([]int, len(loops))
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(x int) int {
		if parent[x] != x {
			parent[x] = find(parent[x])
		}
		return parent[x]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}

	closePair := make(map[int]bool)
	for _, p := range pairs {
		ra := loops[p.a].Radius()
		rb := loops[p.b].Radius()
		scale := math.Min(ra, rb)
		if scale < 0.05 {
			scale = 0.05
		}
		if p.gap <= 0.78*scale {
			closePair[p.a] = true
			closePair[p.b] = true
			union(p.a, p.b)
		}
	}
	for i, l := range loops {
		if i == outer {
			continue
		}
		outerClose := l.Metrics.MinOuterGap <= 0.70*math.Max(l.Radius(), 0.2)
		switch {
		case l.Metrics.Aspect >= 3.1 && l.Metrics.MinSelfGap/math.Max(l.Metrics.Length, 1e-6) <= 0.30:
			l.Class = ClassSlit
			l.Metrics.Reason = "high PCA aspect ratio with close non-adjacent sides"
		case l.Metrics.Convexity <= 0.84 && l.Metrics.ReflexCorners >= 1 && l.Metrics.RightCorners >= 3:
			l.Class = ClassLNotch
			l.Metrics.Reason = "low hull convexity with reflex orthogonal corner"
		case closePair[i] || outerClose:
			l.Class = ClassClosePair
			if outerClose && !closePair[i] {
				l.Metrics.Reason = "hole boundary lies close to outer contour"
			} else {
				l.Metrics.Reason = "distinct hole loops separated by a thin bridge"
			}
		default:
			l.Class = ClassOrdinaryHole
			l.Metrics.Reason = "compact convex interior boundary"
		}
	}

	rootToPair := make(map[int]int)
	nextPair := 1
	for i, l := range loops {
		if l.Class != ClassClosePair || (!closePair[i] && l.Metrics.MinOuterGap > 0.70*math.Max(l.Radius(), 0.2)) {
			continue
		}
		root := find(i)
		if _, ok := rootToPair[root]; !ok {
			rootToPair[root] = nextPair
			nextPair++
		}
		l.Metrics.PairID = rootToPair[root]
	}
}

func loopDistanceWorldXY(m *Mesh, a, b *BoundaryLoop) float64 {
	best := math.Inf(1)
	aw := make([]Vec2, len(a.Verts))
	bw := make([]Vec2, len(b.Verts))
	for i, idx := range a.Verts {
		p := m.V[idx]
		aw[i] = Vec2{p.X, p.Y}
	}
	for i, idx := range b.Verts {
		p := m.V[idx]
		bw[i] = Vec2{p.X, p.Y}
	}
	for i := 0; i < len(aw); i++ {
		a0, a1 := aw[i], aw[(i+1)%len(aw)]
		for j := 0; j < len(bw); j++ {
			b0, b1 := bw[j], bw[(j+1)%len(bw)]
			gap := segmentDistance2D(a0, a1, b0, b1)
			if gap < best {
				best = gap
			}
		}
	}
	return best
}
