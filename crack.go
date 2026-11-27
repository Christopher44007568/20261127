package main

import "sort"

func (t *Topology) RepairCracks(o *Octree) int {
	added := FillBoundaryCycles(o, &t.Tris)
	t.RepairTriangles += added
	return added
}

func FillBoundaryCycles(o *Octree, tris *[]Triangle) int {
	added := 0
	for pass := 0; pass < 3; pass++ {
		boundary := BoundaryEdges(*tris)
		directed := map[[2]int]bool{}
		out := map[int][]int{}
		for _, tri := range *tris {
			ids := tri.V
			for i := 0; i < 3; i++ {
				a := ids[i]
				b := ids[(i+1)%3]
				key := [2]int{a, b}
				und := [2]int{a, b}
				if a > b {
					und = [2]int{b, a}
				}
				if boundary[und] == 1 {
					directed[key] = true
					out[a] = append(out[a], b)
				}
			}
		}
		if len(directed) == 0 {
			break
		}
		for k := range out {
			vs := out[k]
			sort.Ints(vs)
			out[k] = vs
		}

		used := map[[2]int]bool{}
		var repair []Triangle
		edges := make([][2]int, 0, len(directed))
		for e := range directed {
			edges = append(edges, e)
		}
		sort.Slice(edges, func(i, j int) bool {
			if edges[i][0] != edges[j][0] {
				return edges[i][0] < edges[j][0]
			}
			return edges[i][1] < edges[j][1]
		})

		for _, start := range edges {
			if used[start] {
				continue
			}
			cycle := traceBoundaryCycle(start, directed, out, used, o)
			if len(cycle) < 3 {
				continue
			}
			for i := 1; i < len(cycle)-1; i++ {
				repair = append(repair, Triangle{
					V:      [3]int{cycle[0], cycle[i], cycle[i+1]},
					Repair: true,
				})
				added++
			}
		}
		if len(repair) == 0 {
			break
		}
		*tris = append(*tris, repair...)
	}
	return added
}

func traceBoundaryCycle(
	start [2]int,
	directed map[[2]int]bool,
	out map[int][]int,
	used map[[2]int]bool,
	o *Octree,
) []int {
	used[start] = true
	cycle := []int{start[0], start[1]}
	startVertex := start[0]
	cur := start[1]

	for steps := 0; steps < len(directed)+4; steps++ {
		if cur == startVertex {
			return cycle
		}
		var next int
		bestScore := 1e100
		found := false
		for _, cand := range out[cur] {
			e := [2]int{cur, cand}
			if !directed[e] || used[e] {
				continue
			}
			score := boundaryTurnScore(o, cycle[len(cycle)-2], cur, cand)
			if score < bestScore {
				bestScore = score
				next = cand
				found = true
			}
		}
		if !found {
			return nil
		}
		used[[2]int{cur, next}] = true
		if next == startVertex {
			return cycle
		}
		cycle = append(cycle, next)
		cur = next
	}
	return nil
}

func boundaryTurnScore(o *Octree, prev, cur, next int) float64 {
	a := o.Leaves[prev].Sharp
	b := o.Leaves[cur].Sharp
	c := o.Leaves[next].Sharp
	ab := b.Sub(a).Normalize()
	bc := c.Sub(b).Normalize()
	// Prefer continuing straight/smoothly, then shorter edges.
	turn := 1 - ab.Dot(bc)
	length := b.Sub(c).Norm()
	return turn*10 + length
}
