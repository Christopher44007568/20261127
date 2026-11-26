package holefill

import (
	"fmt"
	"math"
)

type PatchOptions struct {
	RefineLevels int
	AlphaWrap    bool
	Alpha        float64
	Strength     float64
}

func DefaultPatchOptions() PatchOptions {
	return PatchOptions{RefineLevels: 2, AlphaWrap: true, Alpha: 0.58, Strength: 0.55}
}

type PatchReport struct {
	LoopID    int
	Class     HoleClass
	Triangles int
	Vertices  int
	Wrapped   bool
}

func PatchAllLoops(m *Mesh, loops []*BoundaryLoop, opts PatchOptions) ([]PatchReport, error) {
	if opts.RefineLevels < 0 || opts.RefineLevels > 4 {
		return nil, fmt.Errorf("refine levels must be in [0,4]")
	}
	if opts.Alpha <= 0 {
		opts.Alpha = DefaultPatchOptions().Alpha
	}
	if opts.Strength <= 0 {
		opts.Strength = DefaultPatchOptions().Strength
	}
	surround := m.ComputeNormals()
	oldVertexCount := len(m.V)
	var reports []PatchReport
	type patchedLoop struct {
		loop *BoundaryLoop
		ids  []int
	}
	var patched []patchedLoop

	for _, loop := range loops {
		if loop.Class == ClassOuter || loop.Class == ClassUnknown {
			continue
		}
		triCountBefore := len(m.T)
		newIDs, err := patchOneLoop(m, loop, surround, opts)
		if err != nil {
			return nil, fmt.Errorf("patch loop %d (%s): %w", loop.ID, loop.Class, err)
		}
		reports = append(reports, PatchReport{
			LoopID: loop.ID, Class: loop.Class,
			Triangles: len(m.T) - triCountBefore,
			Vertices:  len(newIDs), Wrapped: opts.AlphaWrap,
		})
		patched = append(patched, patchedLoop{loop: loop, ids: newIDs})
	}

	m.ComputeNormals()
	copy(m.Norm[:oldVertexCount], surround)
	for _, p := range patched {
		smoothVertexNormals(m, p.ids, 10, 0.55)
	}
	return reports, nil
}

func patchOneLoop(m *Mesh, loop *BoundaryLoop, surround []Vec3, opts PatchOptions) ([]int, error) {
	n := len(loop.Verts)
	order := make([]int, n)
	for i := range order {
		order[i] = i
	}
	if loop.Metrics.SignedArea < 0 {
		for i, j := 0, len(order)-1; i < j; i, j = i+1, j-1 {
			order[i], order[j] = order[j], order[i]
		}
	}
	poly := make([]Vec2, n)
	global := make([]int, n)
	for pos, localIdx := range order {
		poly[pos] = loop.Pts[localIdx]
		global[pos] = loop.Verts[localIdx]
	}
	ears, err := earClip(poly)
	if err != nil {
		ears = fanFallback(poly)
	}

	desired := averageBoundaryNormals(surround, loop.Verts)
	tag := FaceTag{Kind: KindPatch, Class: loop.Class, LoopID: loop.ID}
	var newIDs []int
	for _, localTri := range ears {
		tri := Tri{global[localTri[0]], global[localTri[1]], global[localTri[2]]}
		fn := triangleNormal(m, tri)
		if desired.Len() > 0 && fn.Dot(desired) < 0 {
			tri[1], tri[2] = tri[2], tri[1]
		}
		subdivideCentroid(m, tri, opts.RefineLevels, tag, &newIDs)
	}

	if opts.AlphaWrap {
		alphaWrapLoop(m, loop, surround, newIDs, desired, opts.Alpha, opts.Strength)
	}
	smoothInteriorVertices(m, newIDs, 8, 0.45)
	return newIDs, nil
}

func earClip(poly []Vec2) ([]Tri, error) {
	n := len(poly)
	if n < 3 {
		return nil, fmt.Errorf("polygon has %d vertices", n)
	}
	ids := make([]int, n)
	for i := range ids {
		ids[i] = i
	}
	var tris []Tri
	guard := 0
	for len(ids) > 3 {
		guard++
		if guard > 10*n*n+100 {
			return nil, fmt.Errorf("ear clipping did not converge")
		}
		k := len(ids)
		bestAt := -1
		bestScore := -1.0
		for at := 0; at < k; at++ {
			prev := ids[(at-1+k)%k]
			cur := ids[at]
			next := ids[(at+1)%k]
			a, b, c := poly[prev], poly[cur], poly[next]
			cross := orient2(a, b, c)
			if cross <= 1e-8 {
				continue
			}
			quality := earQuality(a, b, c)
			if quality < 0.055 {
				continue
			}
			blocked := false
			for _, candidate := range ids {
				if candidate == prev || candidate == cur || candidate == next {
					continue
				}
				if pointInTriangle(poly[candidate], a, b, c) {
					blocked = true
					break
				}
			}
			if blocked {
				continue
			}
			areaScore := cross / (1 + math.Hypot(b.X-a.X, b.Y-a.Y)*math.Hypot(c.X-b.X, c.Y-b.Y))
			score := quality + 0.15*areaScore
			if score > bestScore {
				bestScore = score
				bestAt = at
			}
		}
		if bestAt < 0 {
			return nil, fmt.Errorf("no ear found for %d-vertex polygon", n)
		}
		at := bestAt
		prev := ids[(at-1+k)%k]
		cur := ids[at]
		next := ids[(at+1)%k]
		tris = append(tris, Tri{prev, cur, next})
		ids = append(ids[:at], ids[at+1:]...)
	}
	tris = append(tris, Tri{ids[0], ids[1], ids[2]})
	return tris, nil
}

func earQuality(a, b, c Vec2) float64 {
	ab := b.Sub(a)
	bc := c.Sub(b)
	ca := a.Sub(c)
	lab := ab.Len()
	lbc := bc.Len()
	lca := ca.Len()
	if lab < 1e-9 || lbc < 1e-9 || lca < 1e-9 {
		return 0
	}
	cross := math.Abs(ab.Cross(bc))
	return math.Min(cross/(lab*lbc), math.Min(
		2*cross/(lab*lca),
		2*cross/(lbc*lca),
	))
}

func fanFallback(poly []Vec2) []Tri {
	var out []Tri
	for i := 1; i+1 < len(poly); i++ {
		out = append(out, Tri{0, i, i + 1})
	}
	return out
}

func subdivideCentroid(m *Mesh, tri Tri, levels int, tag FaceTag, newIDs *[]int) {
	if levels == 0 {
		m.AppendTri(tri, tag)
		return
	}
	a := m.V[tri[0]]
	b := m.V[tri[1]]
	c := m.V[tri[2]]
	g := m.AppendVertex(a.Add(b).Add(c).Scale(1.0 / 3.0))
	*newIDs = append(*newIDs, g)
	subdivideCentroid(m, Tri{tri[0], tri[1], g}, levels-1, tag, newIDs)
	subdivideCentroid(m, Tri{tri[1], tri[2], g}, levels-1, tag, newIDs)
	subdivideCentroid(m, Tri{tri[2], tri[0], g}, levels-1, tag, newIDs)
}

func triangleNormal(m *Mesh, t Tri) Vec3 {
	a := m.V[t[0]]
	b := m.V[t[1]]
	c := m.V[t[2]]
	return b.Sub(a).Cross(c.Sub(a)).Normalize()
}

func averageBoundaryNormals(normals []Vec3, ids []int) Vec3 {
	var n Vec3
	for _, id := range ids {
		if id >= 0 && id < len(normals) {
			n = n.Add(normals[id])
		}
	}
	return n.Scale(1 / float64(len(ids))).Normalize()
}

func alphaWrapLoop(m *Mesh, loop *BoundaryLoop, normals []Vec3, ids []int, desired Vec3, alpha, strength float64) {
	if desired.Len() < 0.5 {
		desired = loop.Normal
		if desired.Len() < 0.5 {
			desired = Vec3{0, 0, 1}
		}
	}
	for _, id := range ids {
		p := m.V[id]
		var weightSum, heightSum float64
		for i, aID := range loop.Verts {
			bID := loop.Verts[(i+1)%len(loop.Verts)]
			a, b := m.V[aID], m.V[bID]
			q, t := closestPointOnSegment3(p, a, b)
			d := p.Distance(q)
			na := safeNormal(normals, aID, desired)
			nb := safeNormal(normals, bID, desired)
			nq := na.Scale(1 - t).Add(nb.Scale(t)).Normalize()
			if nq.Len() < 0.5 {
				nq = desired
			}
			denom := desired.Dot(nq)
			if math.Abs(denom) < 0.25 {
				denom = sign(denom) * 0.25
			}
			height := q.Sub(p).Dot(nq) / denom
			w := wrapWeight(d, alpha)
			heightSum += w * height
			weightSum += w
		}
		if weightSum > 1e-12 {
			delta := heightSum / weightSum
			delta = math.Max(-alpha*0.20, math.Min(alpha*0.20, delta))
			m.V[id] = p.Add(desired.Scale(delta * strength))
		}
	}
}

func wrapWeight(d, alpha float64) float64 {
	g := math.Exp(-(d * d) / (2 * alpha * alpha))
	inv := 1.0 / (d*d + 0.025)
	return g * (0.35 + 0.65*inv)
}

func safeNormal(normals []Vec3, id int, fallback Vec3) Vec3 {
	if id >= 0 && id < len(normals) && normals[id].Len() > 0.5 {
		return normals[id]
	}
	return fallback
}

func sign(x float64) float64 {
	if x < 0 {
		return -1
	}
	return 1
}

func closestPointOnSegment3(p, a, b Vec3) (Vec3, float64) {
	ab := b.Sub(a)
	l2 := ab.Dot(ab)
	if l2 < 1e-12 {
		return a, 0
	}
	t := p.Sub(a).Dot(ab) / l2
	t = math.Max(0, math.Min(1, t))
	return a.Add(ab.Scale(t)), t
}

func smoothInteriorVertices(m *Mesh, ids []int, iterations int, factor float64) {
	if len(ids) == 0 || iterations <= 0 {
		return
	}
	movable := make(map[int]bool, len(ids))
	for _, id := range ids {
		movable[id] = true
	}
	neighbors := make(map[int]map[int]struct{}, len(ids))
	addNeighbor := func(a, b int) {
		if !movable[a] {
			return
		}
		if neighbors[a] == nil {
			neighbors[a] = make(map[int]struct{})
		}
		neighbors[a][b] = struct{}{}
	}
	for _, t := range m.T {
		addNeighbor(t[0], t[1])
		addNeighbor(t[0], t[2])
		addNeighbor(t[1], t[0])
		addNeighbor(t[1], t[2])
		addNeighbor(t[2], t[0])
		addNeighbor(t[2], t[1])
	}

	for iter := 0; iter < iterations; iter++ {
		updates := make(map[int]Vec3, len(ids))
		for id := range movable {
			set := neighbors[id]
			if len(set) == 0 {
				continue
			}
			var avg Vec3
			for nid := range set {
				avg = avg.Add(m.V[nid])
			}
			avg = avg.Scale(1 / float64(len(set)))
			updates[id] = m.V[id].Add(avg.Sub(m.V[id]).Scale(factor))
		}
		for id, p := range updates {
			m.V[id] = p
		}
	}
}

func smoothVertexNormals(m *Mesh, ids []int, iterations int, factor float64) {
	if len(ids) == 0 || iterations <= 0 || len(m.Norm) != len(m.V) {
		return
	}
	movable := make(map[int]bool, len(ids))
	for _, id := range ids {
		movable[id] = true
	}
	neighbors := make(map[int]map[int]struct{}, len(ids))
	add := func(a, b int) {
		if !movable[a] {
			return
		}
		if neighbors[a] == nil {
			neighbors[a] = make(map[int]struct{})
		}
		neighbors[a][b] = struct{}{}
	}
	for _, t := range m.T {
		add(t[0], t[1])
		add(t[0], t[2])
		add(t[1], t[0])
		add(t[1], t[2])
		add(t[2], t[0])
		add(t[2], t[1])
	}
	for iter := 0; iter < iterations; iter++ {
		updates := make(map[int]Vec3, len(ids))
		for id := range movable {
			set := neighbors[id]
			if len(set) == 0 {
				continue
			}
			var avg Vec3
			for nid := range set {
				n := m.Norm[nid]
				if n.Len() < 0.5 {
					n = Vec3{0, 0, 1}
				}
				avg = avg.Add(n)
			}
			avg = avg.Scale(1 / float64(len(set))).Normalize()
			cur := m.Norm[id]
			if cur.Len() < 0.5 {
				cur = Vec3{0, 0, 1}
			}
			updates[id] = slerpNormal(cur, avg, factor)
		}
		for id, n := range updates {
			m.Norm[id] = n
		}
	}
}
