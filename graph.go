package postfx

import "fmt"

type ResourceID string
type PassID string

// TextureSpec describes a graph resource. External resources are supplied by
// the caller; every other resource must be produced by exactly one pass.
type TextureSpec struct {
	Width    int
	Height   int
	External bool
}

// Pass is one render-graph node.
//
// Inputs are strong data dependencies. A pass cannot produce a meaningful
// result unless every strong input is available. WeakAfter is an ordering-only
// dependency: it never keeps a producer alive, but it is honored whenever both
// passes are selected.
type Pass struct {
	ID        PassID
	Name      string
	Path      string
	Inputs    []ResourceID
	WeakAfter []PassID
	Outputs   []ResourceID
	Run       func(*PassContext) error
}

type Graph struct {
	passes   []*Pass
	byID     map[PassID]*Pass
	producer map[ResourceID]PassID
	textures map[ResourceID]TextureSpec
}

func NewGraph() *Graph {
	return &Graph{
		byID:     make(map[PassID]*Pass),
		producer: make(map[ResourceID]PassID),
		textures: make(map[ResourceID]TextureSpec),
	}
}

func (g *Graph) AddTexture(id ResourceID, spec TextureSpec) error {
	if _, exists := g.textures[id]; exists {
		return fmt.Errorf("postfx: resource %q declared twice", id)
	}
	if spec.Width < 0 || spec.Height < 0 {
		return fmt.Errorf("postfx: resource %q has negative size", id)
	}
	g.textures[id] = spec
	return nil
}

func (g *Graph) AddPass(pass *Pass) error {
	if pass == nil {
		return fmt.Errorf("postfx: nil pass")
	}
	if pass.ID == "" {
		return fmt.Errorf("postfx: pass has empty ID")
	}
	if _, exists := g.byID[pass.ID]; exists {
		return fmt.Errorf("postfx: pass %q declared twice", pass.ID)
	}
	if len(pass.Outputs) == 0 {
		return fmt.Errorf("postfx: pass %q has no output", pass.ID)
	}
	outputSeen := make(map[ResourceID]bool)
	for _, output := range pass.Outputs {
		if outputSeen[output] {
			return fmt.Errorf("postfx: pass %q repeats output %q", pass.ID, output)
		}
		outputSeen[output] = true
		if _, exists := g.producer[output]; exists {
			return fmt.Errorf("postfx: resource %q has multiple producers", output)
		}
		spec, exists := g.textures[output]
		if !exists {
			return fmt.Errorf("postfx: pass %q outputs undeclared resource %q", pass.ID, output)
		}
		if spec.External {
			return fmt.Errorf("postfx: pass %q cannot produce external resource %q", pass.ID, output)
		}
	}
	inputSeen := make(map[ResourceID]bool)
	for _, input := range pass.Inputs {
		if inputSeen[input] {
			return fmt.Errorf("postfx: pass %q repeats input %q", pass.ID, input)
		}
		inputSeen[input] = true
		if _, exists := g.textures[input]; !exists {
			return fmt.Errorf("postfx: pass %q reads undeclared resource %q", pass.ID, input)
		}
	}
	weakSeen := make(map[PassID]bool)
	for _, dependency := range pass.WeakAfter {
		if dependency == pass.ID {
			return fmt.Errorf("postfx: pass %q weakly depends on itself", pass.ID)
		}
		if weakSeen[dependency] {
			return fmt.Errorf("postfx: pass %q repeats weak dependency %q", pass.ID, dependency)
		}
		weakSeen[dependency] = true
	}
	g.passes = append(g.passes, pass)
	g.byID[pass.ID] = pass
	for _, output := range pass.Outputs {
		g.producer[output] = pass.ID
	}
	return nil
}

func (g *Graph) Pass(id PassID) (*Pass, bool) {
	pass, ok := g.byID[id]
	return pass, ok
}

func (g *Graph) Texture(id ResourceID) (TextureSpec, bool) {
	spec, ok := g.textures[id]
	return spec, ok
}

func (g *Graph) Passes() []*Pass {
	out := make([]*Pass, len(g.passes))
	copy(out, g.passes)
	return out
}

func (g *Graph) Validate() error {
	external := make(map[ResourceID]bool)
	for id, spec := range g.textures {
		if spec.External {
			external[id] = true
		}
	}
	for _, pass := range g.passes {
		for _, dependency := range pass.WeakAfter {
			if _, ok := g.byID[dependency]; !ok {
				return fmt.Errorf("postfx: pass %q weakly depends on unknown pass %q", pass.ID, dependency)
			}
		}
		for _, input := range pass.Inputs {
			if external[input] {
				continue
			}
			if _, ok := g.producer[input]; !ok {
				return fmt.Errorf("postfx: pass %q reads unproduced resource %q", pass.ID, input)
			}
		}
	}
	return g.detectCycles(g.passes)
}

func (g *Graph) detectCycles(passes []*Pass) error {
	state := make(map[PassID]int, len(passes))
	selected := make(map[PassID]bool, len(passes))
	for _, pass := range passes {
		selected[pass.ID] = true
	}
	var visit func(*Pass, []PassID) error
	visit = func(pass *Pass, path []PassID) error {
		if state[pass.ID] == 1 {
			return fmt.Errorf("postfx: dependency cycle: %v -> %s", path, pass.ID)
		}
		if state[pass.ID] == 2 {
			return nil
		}
		state[pass.ID] = 1
		nextPath := append(append([]PassID(nil), path...), pass.ID)
		for _, input := range pass.Inputs {
			producerID, ok := g.producer[input]
			if !ok || !selected[producerID] {
				continue
			}
			if err := visit(g.byID[producerID], nextPath); err != nil {
				return err
			}
		}
		for _, weakID := range pass.WeakAfter {
			if selected[weakID] {
				if err := visit(g.byID[weakID], nextPath); err != nil {
					return err
				}
			}
		}
		state[pass.ID] = 2
		return nil
	}
	for _, pass := range passes {
		if state[pass.ID] == 0 {
			if err := visit(pass, nil); err != nil {
				return err
			}
		}
	}
	return nil
}
