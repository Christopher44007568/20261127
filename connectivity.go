package main

import (
	"fmt"
)

type LabelComponent struct {
	ID    int
	Label int
	Cells []int
}

func BuildLabelComponents(g *Grid, labels []int) ([]int, []LabelComponent) {
	compID := make([]int, g.W*g.H)
	for i := range compID {
		compID[i] = -1
	}
	comps := make([]LabelComponent, 0)
	queue := make([]int, 0, 256)

	for start := 0; start < g.W*g.H; start++ {
		if g.Blocked[start] || compID[start] != -1 {
			continue
		}
		label := labels[start]
		id := len(comps)
		comp := LabelComponent{ID: id, Label: label}
		compID[start] = id
		queue = append(queue[:0], start)

		for len(queue) > 0 {
			p := queue[0]
			queue = queue[1:]
			comp.Cells = append(comp.Cells, p)
			px, py := p%g.W, p/g.W
			for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := px+d[0], py+d[1]
				if !g.Open(nx, ny) {
					continue
				}
				q := g.ID(nx, ny)
				if compID[q] == -1 && labels[q] == label {
					compID[q] = id
					queue = append(queue, q)
				}
			}
		}
		comps = append(comps, comp)
	}
	return compID, comps
}

type ConnectivityReport struct {
	Connected       bool     `json:"connected"`
	OpenCells       int      `json:"open_cells"`
	LabelCellCount  []int    `json:"label_cell_count"`
	ComponentCounts []int    `json:"component_counts"`
	Problems        []string `json:"problems,omitempty"`
}

func CheckConnectivity(g *Grid, labels []int, seeds []Seed) ConnectivityReport {
	report := ConnectivityReport{Connected: true, OpenCells: g.OpenCount()}
	k := len(seeds)
	report.LabelCellCount = make([]int, k)
	report.ComponentCounts = make([]int, k)

	for id := 0; id < g.W*g.H; id++ {
		if g.Blocked[id] {
			continue
		}
		label := labels[id]
		if label < 0 || label >= k {
			report.Connected = false
			report.Problems = append(report.Problems, fmt.Sprintf("cell %d has invalid label %d", id, label))
			continue
		}
		report.LabelCellCount[label]++
	}

	_, comps := BuildLabelComponents(g, labels)
	for _, comp := range comps {
		if comp.Label >= 0 && comp.Label < k {
			report.ComponentCounts[comp.Label]++
		}
	}
	for label, count := range report.ComponentCounts {
		if count != 1 {
			report.Connected = false
			report.Problems = append(report.Problems, fmt.Sprintf("label %d has %d 4-connected components", label, count))
		}
	}
	for label, seed := range seeds {
		if !g.Inside(seed.X, seed.Y) || g.Blocked[g.ID(seed.X, seed.Y)] {
			report.Connected = false
			report.Problems = append(report.Problems, fmt.Sprintf("seed %d is invalid", label))
			continue
		}
		if labels[g.ID(seed.X, seed.Y)] != label {
			report.Connected = false
			report.Problems = append(report.Problems, fmt.Sprintf("seed %d does not own its cell", label))
		}
	}
	return report
}

// EnsureConnected recolors disconnected label-components through the component
// adjacency graph. Every recolored component has a same-label adjacent parent,
// so the final label is 4-connected and never crosses a blocked hole.
func EnsureConnected(g *Grid, labels []int, seeds []Seed) (int, error) {
	if len(seeds) == 0 {
		return 0, fmt.Errorf("no seeds")
	}
	for label, seed := range seeds {
		if !g.Open(seed.X, seed.Y) {
			return 0, fmt.Errorf("seed %d (%d,%d) is not traversable", label, seed.X, seed.Y)
		}
		labels[g.ID(seed.X, seed.Y)] = label
	}

	compID, comps := BuildLabelComponents(g, labels)
	rootComp := make([]int, len(seeds))
	for i := range rootComp {
		rootComp[i] = -1
	}
	for label, seed := range seeds {
		id := g.ID(seed.X, seed.Y)
		c := compID[id]
		if c < 0 {
			return 0, fmt.Errorf("seed %d has no component", label)
		}
		rootComp[label] = c
	}

	adj := make([][]int, len(comps))
	addEdge := func(a, b int) {
		if a == b || a < 0 || b < 0 {
			return
		}
		adj[a] = append(adj[a], b)
		adj[b] = append(adj[b], a)
	}
	for y := 0; y < g.H; y++ {
		for x := 0; x < g.W; x++ {
			id := g.ID(x, y)
			if g.Blocked[id] {
				continue
			}
			if x+1 < g.W && !g.Blocked[id+1] {
				addEdge(compID[id], compID[id+1])
			}
			if y+1 < g.H && !g.Blocked[id+g.W] {
				addEdge(compID[id], compID[id+g.W])
			}
		}
	}

	owner := make([]int, len(comps))
	queue := make([]int, 0, len(comps))
	for i := range owner {
		owner[i] = -1
	}
	for label, c := range rootComp {
		if c < 0 || owner[c] != -1 {
			continue
		}
		owner[c] = label
		queue = append(queue, c)
	}
	for head := 0; head < len(queue); head++ {
		c := queue[head]
		for _, next := range adj[c] {
			if owner[next] == -1 {
				owner[next] = owner[c]
				queue = append(queue, next)
			}
		}
	}

	changed := 0
	for c, comp := range comps {
		if owner[c] < 0 {
			return changed, fmt.Errorf("traversable component has no seed and cannot be connected to one")
		}
		if comp.Label == owner[c] {
			continue
		}
		for _, id := range comp.Cells {
			labels[id] = owner[c]
			changed++
		}
	}
	return changed, nil
}

// flipIsTopologySafe checks that giving p newLabel joins an already connected
// new-label neighbor and removing p does not split the old label.
func flipIsTopologySafe(g *Grid, labels []int, p, newLabel int, marks []int, queue []int, token int) bool {
	oldLabel := labels[p]
	if oldLabel == newLabel {
		return false
	}
	px, py := p%g.W, p/g.W
	oldNeighbors := make([]int, 0, 4)
	hasNew := false
	for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		nx, ny := px+d[0], py+d[1]
		if !g.Open(nx, ny) {
			continue
		}
		q := g.ID(nx, ny)
		switch labels[q] {
		case oldLabel:
			oldNeighbors = append(oldNeighbors, q)
		case newLabel:
			hasNew = true
		}
	}
	if !hasNew {
		return false
	}
	if len(oldNeighbors) <= 1 {
		return true
	}

	queue = append(queue[:0], oldNeighbors[0])
	marks[oldNeighbors[0]] = token
	reached := 1
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		ix, iy := id%g.W, id/g.W
		for _, d := range [][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := ix+d[0], iy+d[1]
			if !g.Open(nx, ny) {
				continue
			}
			q := g.ID(nx, ny)
			if q == p || marks[q] == token || labels[q] != oldLabel {
				continue
			}
			marks[q] = token
			reached++
			queue = append(queue, q)
		}
	}
	for _, q := range oldNeighbors[1:] {
		if marks[q] != token {
			return false
		}
	}
	return true
}
