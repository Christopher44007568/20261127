package bloom

import (
	"fmt"
	"sort"
)

// Node is one processing stage. Every node boundary is an encoded texture
// when the graph runs in quantized mode.
type Node interface {
	Kind() string
	InputCount() int
	AcceptsInput(FormatID) bool
	SupportsOutput(FormatID) bool
	Evaluate(inputs []*Image, outputFormat FormatID) (*Image, error)
}

type nodeSpec struct {
	node Node
	deps []string
}

type Graph struct {
	sink  string
	nodes map[string]nodeSpec
}

func NewGraph(sink string) *Graph {
	return &Graph{sink: sink, nodes: make(map[string]nodeSpec)}
}

func (g *Graph) AddNode(id string, node Node, inputs ...string) error {
	if id == "" {
		return fmt.Errorf("node id must not be empty")
	}
	if node == nil {
		return fmt.Errorf("node %q is nil", id)
	}
	if _, exists := g.nodes[id]; exists {
		return fmt.Errorf("node %q already exists", id)
	}
	if node.InputCount() != len(inputs) {
		return fmt.Errorf("node %q expects %d inputs, got %d", id, node.InputCount(), len(inputs))
	}
	seen := make(map[string]struct{}, len(inputs))
	for _, dep := range inputs {
		if dep == id {
			return fmt.Errorf("node %q cannot depend on itself", id)
		}
		if _, exists := g.nodes[dep]; !exists {
			return fmt.Errorf("node %q depends on unknown node %q", id, dep)
		}
		if _, duplicate := seen[dep]; duplicate {
			return fmt.Errorf("node %q lists input %q more than once", id, dep)
		}
		seen[dep] = struct{}{}
	}
	g.nodes[id] = nodeSpec{node: node, deps: append([]string(nil), inputs...)}
	return nil
}

type Plan struct {
	Format        FormatID            `json:"format"`
	Order         []string            `json:"order"`
	OutputFormats map[string]FormatID `json:"node_output_formats"`
}

// Negotiate propagates the sink's requested format backwards across every
// edge. A node may only emit a format it advertises, and each downstream node
// must be able to consume the format selected for its upstream edge.
func (g *Graph) Negotiate(requested FormatID) (*Plan, error) {
	if _, err := FormatInfoFor(requested); err != nil {
		return nil, err
	}
	if _, ok := g.nodes[g.sink]; !ok {
		return nil, fmt.Errorf("sink node %q is missing", g.sink)
	}
	order, err := g.topologicalOrder()
	if err != nil {
		return nil, err
	}
	formats := make(map[string]FormatID, len(g.nodes))
	for i := len(order) - 1; i >= 0; i-- {
		id := order[i]
		spec := g.nodes[id]
		if !spec.node.SupportsOutput(requested) {
			return nil, fmt.Errorf("node %q (%s) cannot output %s", id, spec.node.Kind(), requested)
		}
		formats[id] = requested
		for _, dep := range spec.deps {
			if !spec.node.AcceptsInput(requested) {
				return nil, fmt.Errorf("node %q (%s) cannot consume %s", id, spec.node.Kind(), requested)
			}
			if _, assigned := formats[dep]; assigned && formats[dep] != requested {
				return nil, fmt.Errorf("edge %s -> %s has conflicting formats %s and %s",
					dep, id, formats[dep], requested)
			}
		}
	}
	return &Plan{
		Format:        requested,
		Order:         order,
		OutputFormats: formats,
	}, nil
}

func (g *Graph) topologicalOrder() ([]string, error) {
	state := make(map[string]uint8, len(g.nodes))
	order := make([]string, 0, len(g.nodes))
	var visit func(string) error
	visit = func(id string) error {
		switch state[id] {
		case 1:
			return fmt.Errorf("node graph contains a cycle at %q", id)
		case 2:
			return nil
		}
		state[id] = 1
		deps := append([]string(nil), g.nodes[id].deps...)
		sort.Strings(deps)
		for _, dep := range deps {
			if err := visit(dep); err != nil {
				return err
			}
		}
		state[id] = 2
		order = append(order, id)
		return nil
	}
	if err := visit(g.sink); err != nil {
		return nil, err
	}
	if len(order) != len(g.nodes) {
		missing := make([]string, 0)
		for id := range g.nodes {
			if state[id] == 0 {
				missing = append(missing, id)
			}
		}
		sort.Strings(missing)
		return nil, fmt.Errorf("graph has nodes unreachable from sink %q: %v", g.sink, missing)
	}
	return order, nil
}

type NodeOutput struct {
	ID           string              `json:"id"`
	Kind         string              `json:"kind"`
	Format       FormatID            `json:"format"`
	Texture      *Texture            `json:"-"`
	Linear       *Image              `json:"-"`
	Quantization *QuantizationReport `json:"quantization,omitempty"`
}

type Execution struct {
	Format    FormatID               `json:"format"`
	Quantized bool                   `json:"quantized"`
	Order     []string               `json:"order"`
	Nodes     map[string]*NodeOutput `json:"-"`
	Final     *NodeOutput            `json:"-"`
	Range     RangeCache             `json:"range_cache"`
}

func (g *Graph) Execute(plan *Plan) (*Execution, error) {
	return g.execute(plan, true)
}

// ExecuteUnquantized evaluates the same nodes without texture round trips. It
// is the linear reference used to measure cumulative format error.
func (g *Graph) ExecuteUnquantized(plan *Plan) (*Execution, error) {
	return g.execute(plan, false)
}

func (g *Graph) execute(plan *Plan, quantize bool) (*Execution, error) {
	if plan == nil {
		return nil, fmt.Errorf("nil plan")
	}
	if len(plan.Order) != len(g.nodes) {
		return nil, fmt.Errorf("plan has %d stages, graph has %d", len(plan.Order), len(g.nodes))
	}
	exec := &Execution{
		Format: plan.Format, Quantized: quantize,
		Order: append([]string(nil), plan.Order...),
		Nodes: make(map[string]*NodeOutput, len(g.nodes)),
	}
	for _, id := range plan.Order {
		spec, ok := g.nodes[id]
		if !ok {
			return nil, fmt.Errorf("plan references unknown node %q", id)
		}
		format, ok := plan.OutputFormats[id]
		if !ok {
			return nil, fmt.Errorf("plan has no negotiated format for node %q", id)
		}
		if format != plan.Format {
			return nil, fmt.Errorf("node %q selected %s, graph requested %s", id, format, plan.Format)
		}
		inputs := make([]*Image, 0, len(spec.deps))
		for _, dep := range spec.deps {
			out, ok := exec.Nodes[dep]
			if !ok {
				return nil, fmt.Errorf("node %q executed before dependency %q", id, dep)
			}
			inputs = append(inputs, out.Linear)
		}
		linear, err := spec.node.Evaluate(inputs, format)
		if err != nil {
			return nil, fmt.Errorf("node %q (%s): %w", id, spec.node.Kind(), err)
		}
		if err := linear.Check(); err != nil {
			return nil, fmt.Errorf("node %q (%s) produced an invalid image: %w", id, spec.node.Kind(), err)
		}
		if len(inputs) > 0 {
			if err := sameDimensions(inputs[0], linear); err != nil {
				return nil, fmt.Errorf("node %q: %w", id, err)
			}
		}
		out := &NodeOutput{ID: id, Kind: spec.node.Kind(), Format: format, Linear: linear}
		if quantize {
			tex, report, err := EncodeImage(linear, format)
			if err != nil {
				return nil, fmt.Errorf("node %q encode %s: %w", id, format, err)
			}
			decoded, err := DecodeImage(tex)
			if err != nil {
				return nil, fmt.Errorf("node %q decode %s: %w", id, format, err)
			}
			out.Texture = tex
			out.Linear = decoded
			out.Quantization = report
			exec.Range = exec.Range.Merge(report.Range)
		}
		exec.Nodes[id] = out
	}
	final, ok := exec.Nodes[g.sink]
	if !ok {
		return nil, fmt.Errorf("sink %q was not executed", g.sink)
	}
	exec.Final = final
	return exec, nil
}
