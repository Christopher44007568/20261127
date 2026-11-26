package main

import (
	"math"
	"sort"
)

type OutlierReport struct {
	Keep      []int
	Scores    []float64
	Threshold float64
	Spacing   float64
	Removed   int
	Total     int
	K         int
}

func FilterOutliers(points []V3, k int, robustZ float64) OutlierReport {
	if k <= 0 {
		k = 12
	}
	if k >= len(points) {
		k = maxInt(3, len(points)/2)
	}
	rep := OutlierReport{Total: len(points), K: k}
	if len(points) <= k+1 {
		rep.Keep = make([]int, len(points))
		for i := range points {
			rep.Keep[i] = i
		}
		return rep
	}

	tree := NewKDTree(points)
	scores := make([]float64, len(points))
	first := make([]float64, len(points))
	for i, p := range points {
		ns := tree.KNN(p, k+1)
		sum := 0.0
		count := 0
		for _, n := range ns {
			if n.ID == i {
				continue
			}
			d := sqrtFloat(n.Dist2)
			sum += d
			count++
			if count == 1 {
				first[i] = d
			}
		}
		if count > 0 {
			scores[i] = sum / float64(count)
		}
	}

	med := medianFloat(scores)
	deviations := make([]float64, len(scores))
	for i, s := range scores {
		deviations[i] = absFloat(s - med)
	}
	mad := medianFloat(deviations)
	sigma := 1.4826 * mad
	spacing := medianFloat(first)
	threshold := med + robustZ*sigma
	if floor := med * 1.9; threshold < floor {
		threshold = floor
	}
	if floor := spacing * 4.0; threshold < floor {
		threshold = floor
	}

	keep := make([]int, 0, len(points))
	for i, s := range scores {
		if s <= threshold {
			keep = append(keep, i)
		}
	}
	rep.Keep = keep
	rep.Scores = scores
	rep.Threshold = threshold
	rep.Spacing = spacing
	rep.Removed = len(points) - len(keep)
	return rep
}

func EstimateNormals(points []V3, k int) ([]V3, []float64) {
	if k <= 0 {
		k = 24
	}
	normals := make([]V3, len(points))
	quality := make([]float64, len(points))
	if len(points) == 0 {
		return normals, quality
	}
	tree := NewKDTree(points)

	for i, p := range points {
		ns := tree.KNN(p, minInt(k+1, len(points)))
		centroid := V3{}
		count := 0
		for _, n := range ns {
			centroid = centroid.Add(points[n.ID])
			count++
		}
		if count == 0 {
			normals[i] = V3{0, 0, 1}
			continue
		}
		centroid = centroid.Div(float64(count))
		var cov [3][3]float64
		for _, n := range ns {
			r := points[n.ID].Sub(centroid)
			cov[0][0] += r.X * r.X
			cov[0][1] += r.X * r.Y
			cov[0][2] += r.X * r.Z
			cov[1][1] += r.Y * r.Y
			cov[1][2] += r.Y * r.Z
			cov[2][2] += r.Z * r.Z
		}
		cov[1][0] = cov[0][1]
		cov[2][0] = cov[0][2]
		cov[2][1] = cov[1][2]
		vals, vecs := eigenSym3(cov)
		n := Normalize(V3{vecs[0][0], vecs[1][0], vecs[2][0]})
		if Norm(n) < 0.5 {
			n = V3{0, 0, 1}
		}
		normals[i] = n
		sum := vals[0] + vals[1] + vals[2]
		if sum > 1e-18 {
			quality[i] = vals[0] / sum
		}
	}
	return normals, quality
}

type graphEdge struct {
	a, b   int
	weight float64
}

type unionFind struct {
	parent []int
	size   []int
}

func newUnionFind(n int) *unionFind {
	u := &unionFind{parent: make([]int, n), size: make([]int, n)}
	for i := range u.parent {
		u.parent[i] = i
		u.size[i] = 1
	}
	return u
}

func (u *unionFind) find(x int) int {
	for u.parent[x] != x {
		u.parent[x] = u.parent[u.parent[x]]
		x = u.parent[x]
	}
	return x
}

func (u *unionFind) union(a, b int) bool {
	ra, rb := u.find(a), u.find(b)
	if ra == rb {
		return false
	}
	if u.size[ra] < u.size[rb] {
		ra, rb = rb, ra
	}
	u.parent[rb] = ra
	u.size[ra] += u.size[rb]
	return true
}

type OrientationReport struct {
	Flips      int
	Components int
	Edges      int
}

func OrientNormalsMST(points []V3, normals []V3, k int, preferred V3) OrientationReport {
	rep := OrientationReport{}
	if len(points) == 0 || len(normals) != len(points) {
		return rep
	}
	if k <= 0 {
		k = 10
	}
	preferred = Normalize(preferred)
	tree := NewKDTree(points)
	edges := make([]graphEdge, 0, len(points)*k)
	seen := map[uint64]bool{}
	addEdge := func(a, b int) {
		if a == b {
			return
		}
		key := edgeKey(a, b)
		if seen[key] {
			return
		}
		seen[key] = true
		d := Dot(normals[a], normals[b])
		w := 1 - absFloat(d)
		if w < 0 {
			w = 0
		}
		edges = append(edges, graphEdge{a, b, w})
	}

	for i, p := range points {
		ns := tree.KNN(p, minInt(k+1, len(points)))
		for _, n := range ns {
			if n.ID != i {
				addEdge(i, n.ID)
			}
		}
	}
	sort.Slice(edges, func(i, j int) bool { return edges[i].weight < edges[j].weight })

	uf := newUnionFind(len(points))
	adj := make([][]int, len(points))
	mstCount := 0
	for _, e := range edges {
		if uf.union(e.a, e.b) {
			adj[e.a] = append(adj[e.a], e.b)
			adj[e.b] = append(adj[e.b], e.a)
			mstCount++
		}
	}
	rep.Edges = len(edges)

	visited := make([]bool, len(points))
	for root := 0; root < len(points); root++ {
		if visited[root] {
			continue
		}
		rep.Components++
		component := []int{}
		queue := []int{root}
		visited[root] = true
		for head := 0; head < len(queue); head++ {
			cur := queue[head]
			component = append(component, cur)
			for _, nb := range adj[cur] {
				if visited[nb] {
					continue
				}
				visited[nb] = true
				if Dot(normals[cur], normals[nb]) < 0 {
					normals[nb] = normals[nb].Mul(-1)
					rep.Flips++
				}
				queue = append(queue, nb)
			}
		}
		if Norm(preferred) > 0.1 {
			s := 0.0
			for _, id := range component {
				s += Dot(normals[id], preferred)
			}
			if s < 0 {
				for _, id := range component {
					normals[id] = normals[id].Mul(-1)
				}
			}
		}
	}
	return rep
}

func edgeKey(a, b int) uint64 {
	if a > b {
		a, b = b, a
	}
	return uint64(uint32(a))<<32 | uint64(uint32(b))
}

func sqrtFloat(x float64) float64 {
	return math.Sqrt(math.Max(0, x))
}

func absFloat(x float64) float64 {
	return math.Abs(x)
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
