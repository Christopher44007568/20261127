package postfx

import (
	"fmt"
	"sort"
)

type ExecutionMode int

const (
	// ExecuteReachable removes every node that cannot contribute to a requested
	// root. It is the normal optimized mode.
	ExecuteReachable ExecutionMode = iota
	// ExecuteAllActive runs every enabled node, including nodes whose outputs
	// are not requested. It is useful as an unoptimized reference run.
	ExecuteAllActive
)

type ExecuteOptions struct {
	Mode            ExecutionMode
	Roots           []ResourceID
	DisabledOutputs map[ResourceID]bool
	External        map[ResourceID]*Image
}

type SkippedPass struct {
	ID     PassID
	Reason string
}

// Plan is the result of dead-pass elimination before any pixels are written.
type Plan struct {
	Mode    ExecutionMode
	Passes  []PassID
	Order   []PassID
	Skipped []SkippedPass
	Roots   []ResourceID

	live             map[PassID]bool
	enabledOutputs   map[ResourceID]bool
	requiredOutputs  map[ResourceID]bool
	readers          map[ResourceID]int
	producerOfOutput map[ResourceID]PassID
}

func (g *Graph) Plan(options ExecuteOptions) (*Plan, error) {
	if err := g.Validate(); err != nil {
		return nil, err
	}
	if options.Mode != ExecuteReachable && options.Mode != ExecuteAllActive {
		return nil, fmt.Errorf("postfx: invalid execution mode %d", options.Mode)
	}

	disabled := options.DisabledOutputs
	if disabled == nil {
		disabled = map[ResourceID]bool{}
	}
	roots := make(map[ResourceID]bool)
	for _, root := range options.Roots {
		if disabled[root] {
			continue
		}
		if _, ok := g.textures[root]; !ok {
			return nil, fmt.Errorf("postfx: requested unknown root %q", root)
		}
		roots[root] = true
	}

	enabledOutputs := make(map[ResourceID]bool)
	for _, pass := range g.passes {
		for _, output := range pass.Outputs {
			if !disabled[output] {
				enabledOutputs[output] = true
			}
		}
	}

	var live map[PassID]bool
	skipped := make(map[PassID]string)
	var err error
	switch options.Mode {
	case ExecuteReachable:
		live, err = g.reachablePasses(roots, enabledOutputs, options.External, disabled)
		if err != nil {
			return nil, err
		}
	case ExecuteAllActive:
		live = g.allActivePasses(enabledOutputs, options.External)
	}

	order, err := g.orderPasses(live)
	if err != nil {
		return nil, err
	}

	for _, pass := range g.passes {
		if live[pass.ID] {
			continue
		}
		if _, ok := skipped[pass.ID]; ok {
			continue
		}
		skipped[pass.ID] = reasonForSkipped(pass, roots, enabledOutputs)
	}

	plan := &Plan{
		Mode:             options.Mode,
		Passes:           make([]PassID, 0, len(live)),
		Skipped:          make([]SkippedPass, 0, len(skipped)),
		Roots:            make([]ResourceID, 0, len(roots)),
		live:             live,
		enabledOutputs:   enabledOutputs,
		requiredOutputs:  make(map[ResourceID]bool),
		readers:          make(map[ResourceID]int),
		producerOfOutput: make(map[ResourceID]PassID),
	}
	for id := range live {
		plan.Passes = append(plan.Passes, id)
	}
	for id, reason := range skipped {
		plan.Skipped = append(plan.Skipped, SkippedPass{ID: id, Reason: reason})
	}
	for root := range roots {
		plan.Roots = append(plan.Roots, root)
	}
	sort.Slice(plan.Passes, func(i, j int) bool { return plan.Passes[i] < plan.Passes[j] })
	sort.Slice(plan.Skipped, func(i, j int) bool { return plan.Skipped[i].ID < plan.Skipped[j].ID })
	sort.Slice(plan.Roots, func(i, j int) bool { return plan.Roots[i] < plan.Roots[j] })
	plan.Order = order
	plan.countReaders(g, roots)
	for root := range roots {
		plan.requiredOutputs[root] = true
	}
	for _, id := range order {
		for _, input := range g.byID[id].Inputs {
			if _, internal := g.producer[input]; internal {
				plan.requiredOutputs[input] = true
			}
		}
	}
	if options.Mode == ExecuteAllActive {
		for output := range enabledOutputs {
			plan.requiredOutputs[output] = true
		}
	}
	return plan, nil
}

func reasonForSkipped(pass *Pass, roots, enabled map[ResourceID]bool) string {
	hasEnabledOutput := false
	for _, output := range pass.Outputs {
		if enabled[output] {
			hasEnabledOutput = true
			break
		}
	}
	if !hasEnabledOutput {
		return "all outputs disabled"
	}
	for _, output := range pass.Outputs {
		if roots[output] {
			return "requested output unavailable"
		}
	}
	return "not reachable from requested outputs"
}

func (g *Graph) reachablePasses(roots, enabled map[ResourceID]bool, external map[ResourceID]*Image, disabled map[ResourceID]bool) (map[PassID]bool, error) {
	live := make(map[PassID]bool)
	visiting := make(map[PassID]bool)
	var visit func(PassID) error
	visit = func(id PassID) error {
		if live[id] {
			return nil
		}
		pass := g.byID[id]
		for _, input := range pass.Inputs {
			if disabled[input] {
				return fmt.Errorf("postfx: pass %q reads user-disabled output %q", pass.ID, input)
			}
			if spec, ok := g.textures[input]; ok && spec.External {
				if _, supplied := external[input]; !supplied {
					return fmt.Errorf("postfx: missing external input %q for pass %q", input, pass.ID)
				}
				continue
			}
			producer, ok := g.producer[input]
			if !ok {
				return fmt.Errorf("postfx: pass %q reads unproduced resource %q", pass.ID, input)
			}
			if !enabled[input] {
				return fmt.Errorf("postfx: pass %q needs unavailable output %q", pass.ID, input)
			}
			if visiting[producer] {
				return fmt.Errorf("postfx: dependency cycle at pass %q", producer)
			}
			visiting[producer] = true
			if err := visit(producer); err != nil {
				return err
			}
			delete(visiting, producer)
		}
		live[id] = true
		live[pass.ID] = true
		return nil
	}
	for _, pass := range g.passes {
		for _, output := range pass.Outputs {
			if !roots[output] || !enabled[output] {
				continue
			}
			visiting[pass.ID] = true
			if err := visit(pass.ID); err != nil {
				return nil, err
			}
			delete(visiting, pass.ID)
		}
	}
	return live, nil
}

func (g *Graph) allActivePasses(enabled map[ResourceID]bool, external map[ResourceID]*Image) map[PassID]bool {
	live := make(map[PassID]bool)
	for _, pass := range g.passes {
		for _, output := range pass.Outputs {
			if enabled[output] {
				live[pass.ID] = true
				break
			}
		}
	}
	for changed := true; changed; {
		changed = false
		for id := range live {
			pass := g.byID[id]
			for _, input := range pass.Inputs {
				if spec, ok := g.textures[input]; ok && spec.External {
					if _, supplied := external[input]; !supplied {
						delete(live, id)
						changed = true
						break
					}
					continue
				}
				producer, ok := g.producer[input]
				if !ok || !live[producer] || !enabled[input] {
					delete(live, id)
					changed = true
					break
				}
			}
		}
	}
	return live
}

func (g *Graph) orderPasses(live map[PassID]bool) ([]PassID, error) {
	indegree := make(map[PassID]int, len(live))
	next := make(map[PassID][]PassID, len(live))
	for id := range live {
		pass := g.byID[id]
		seen := make(map[PassID]bool)
		edges := make([]PassID, 0, len(pass.Inputs)+len(pass.WeakAfter))
		for _, input := range pass.Inputs {
			if producer, ok := g.producer[input]; ok && live[producer] && !seen[producer] {
				seen[producer] = true
				edges = append(edges, producer)
			}
		}
		for _, weak := range pass.WeakAfter {
			if live[weak] && !seen[weak] {
				seen[weak] = true
				edges = append(edges, weak)
			}
		}
		for _, dependency := range edges {
			next[dependency] = append(next[dependency], id)
			indegree[id]++
		}
	}
	ready := make([]PassID, 0, len(live))
	for id := range live {
		if indegree[id] == 0 {
			ready = append(ready, id)
		}
	}
	sort.Slice(ready, func(i, j int) bool { return ready[i] < ready[j] })
	order := make([]PassID, 0, len(live))
	for len(ready) > 0 {
		id := ready[0]
		ready = ready[1:]
		order = append(order, id)
		dependents := next[id]
		sort.Slice(dependents, func(i, j int) bool { return dependents[i] < dependents[j] })
		for _, dependent := range dependents {
			indegree[dependent]--
			if indegree[dependent] == 0 {
				ready = append(ready, dependent)
				sort.Slice(ready, func(i, j int) bool { return ready[i] < ready[j] })
			}
		}
	}
	if len(order) != len(live) {
		return nil, fmt.Errorf("postfx: dependency cycle among %d selected passes", len(live)-len(order))
	}
	return order, nil
}

func (p *Plan) countReaders(g *Graph, roots map[ResourceID]bool) {
	for id := range p.live {
		pass := g.byID[id]
		seen := make(map[ResourceID]bool)
		for _, input := range pass.Inputs {
			if seen[input] {
				continue
			}
			seen[input] = true
			p.readers[input]++
		}
	}
	for root := range roots {
		p.readers[root]++
	}
	for id := range p.live {
		for _, output := range g.byID[id].Outputs {
			if p.enabledOutputs[output] {
				p.producerOfOutput[output] = id
			}
		}
	}
}
