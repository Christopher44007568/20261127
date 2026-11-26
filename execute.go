package postfx

import (
	"fmt"
	"sort"
)

type PassContext struct {
	PassID   PassID
	PassName string
	Inputs   map[ResourceID]*Image
	Outputs  map[ResourceID]*Image
}

func (c *PassContext) Input(id ResourceID) *Image {
	return c.Inputs[id]
}

func (c *PassContext) Output(id ResourceID) *Image {
	return c.Outputs[id]
}

type ExecutedNode struct {
	ID      PassID
	Name    string
	Path    string
	Outputs []ResourceID
}

type ExecutionReport struct {
	Mode              ExecutionMode
	Executed          []ExecutedNode
	PlannedPasses     []PassID
	Skipped           []SkippedPass
	AllocatedOutputs  []ResourceID
	ReleasedOutputs   []ResourceID
	Roots             []ResourceID
	DisabledOutputs   []ResourceID
	PeakLiveTextures  int
	PeakTextureBytes  int64
	TempTexturesAtEnd int
}

type Frame struct {
	Images map[ResourceID]*Image
	Report ExecutionReport
}

func (g *Graph) Execute(options ExecuteOptions) (*Frame, error) {
	plan, err := g.Plan(options)
	if err != nil {
		return nil, err
	}
	if options.External == nil {
		options.External = map[ResourceID]*Image{}
	}
	disabled := options.DisabledOutputs
	if disabled == nil {
		disabled = map[ResourceID]bool{}
	}
	for id, img := range options.External {
		spec, ok := g.textures[id]
		if !ok {
			return nil, fmt.Errorf("postfx: external image %q is not declared", id)
		}
		if !spec.External {
			return nil, fmt.Errorf("postfx: image %q is not an external resource", id)
		}
		if img.Width != spec.Width || img.Height != spec.Height {
			return nil, fmt.Errorf("postfx: external image %q is %dx%d, expected %dx%d",
				id, img.Width, img.Height, spec.Width, spec.Height)
		}
	}
	for _, root := range plan.Roots {
		if spec, ok := g.textures[root]; ok && spec.External {
			if _, supplied := options.External[root]; !supplied {
				return nil, fmt.Errorf("postfx: missing external root %q", root)
			}
			continue
		}
		if !plan.live[plan.producerOfOutput[root]] {
			return nil, fmt.Errorf("postfx: requested root %q is not produced by a selected pass", root)
		}
	}

	report := ExecutionReport{
		Mode:          plan.Mode,
		PlannedPasses: append([]PassID(nil), plan.Order...),
		Skipped:       append([]SkippedPass(nil), plan.Skipped...),
		Roots:         append([]ResourceID(nil), plan.Roots...),
	}
	for id := range disabled {
		report.DisabledOutputs = append(report.DisabledOutputs, id)
	}
	sort.Slice(report.DisabledOutputs, func(i, j int) bool {
		return report.DisabledOutputs[i] < report.DisabledOutputs[j]
	})

	refs := make(map[ResourceID]int, len(plan.readers))
	for id, count := range plan.readers {
		refs[id] = count
	}
	pool := newTexturePool()
	liveImages := make(map[ResourceID]*Image)
	for id, img := range options.External {
		liveImages[id] = img
	}
	released := make(map[ResourceID]bool)
	retained := make(map[ResourceID]bool)
	for _, root := range plan.Roots {
		retained[root] = true
	}

	release := func(id ResourceID) {
		if released[id] {
			return
		}
		img := liveImages[id]
		producer, internal := g.producer[id]
		if img == nil || !internal || !plan.live[producer] {
			return
		}
		released[id] = true
		report.ReleasedOutputs = append(report.ReleasedOutputs, id)
		pool.release(img)
		delete(liveImages, id)
	}
	updatePeak := func() {
		if len(liveImages) > report.PeakLiveTextures {
			report.PeakLiveTextures = len(liveImages)
		}
		var bytes int64
		for id := range liveImages {
			spec := g.textures[id]
			bytes += int64(spec.Width * spec.Height * 3 * 8)
		}
		if bytes > report.PeakTextureBytes {
			report.PeakTextureBytes = bytes
		}
	}

	for _, id := range plan.Order {
		pass := g.byID[id]
		ctx := &PassContext{
			PassID:   pass.ID,
			PassName: displayName(pass),
			Inputs:   make(map[ResourceID]*Image, len(pass.Inputs)),
			Outputs:  make(map[ResourceID]*Image, len(pass.Outputs)),
		}
		for _, input := range pass.Inputs {
			img := liveImages[input]
			if img == nil {
				return nil, fmt.Errorf("postfx: pass %q has no live input %q", pass.ID, input)
			}
			ctx.Inputs[input] = img
		}
		executedOutputs := make([]ResourceID, 0, len(pass.Outputs))
		for _, output := range pass.Outputs {
			if disabled[output] || !plan.requiredOutputs[output] {
				continue
			}
			spec := g.textures[output]
			img := pool.acquire(spec.Width, spec.Height)
			liveImages[output] = img
			ctx.Outputs[output] = img
			executedOutputs = append(executedOutputs, output)
			report.AllocatedOutputs = append(report.AllocatedOutputs, output)
		}
		updatePeak()
		if pass.Run != nil {
			if err := pass.Run(ctx); err != nil {
				return nil, fmt.Errorf("postfx: pass %q: %w", pass.ID, err)
			}
		}
		report.Executed = append(report.Executed, ExecutedNode{
			ID:      pass.ID,
			Name:    displayName(pass),
			Path:    pass.Path,
			Outputs: executedOutputs,
		})

		for _, input := range pass.Inputs {
			if refs[input] > 0 {
				refs[input]--
			}
			if refs[input] == 0 {
				release(input)
			}
		}
		for _, output := range executedOutputs {
			if refs[output] == 0 {
				release(output)
			}
		}
	}

	images := make(map[ResourceID]*Image, len(plan.Roots))
	for _, root := range plan.Roots {
		img := liveImages[root]
		if img == nil {
			return nil, fmt.Errorf("postfx: selected root %q was not produced", root)
		}
		images[root] = img
	}
	for id := range liveImages {
		if _, internal := g.producer[id]; !internal || retained[id] {
			continue
		}
		report.TempTexturesAtEnd++
	}
	return &Frame{Images: images, Report: report}, nil
}

func displayName(pass *Pass) string {
	if pass.Name != "" {
		return pass.Name
	}
	return string(pass.ID)
}

type textureKey struct {
	width  int
	height int
}

type texturePool struct {
	free map[textureKey][]*Image
}

func newTexturePool() *texturePool {
	return &texturePool{free: make(map[textureKey][]*Image)}
}

func (p *texturePool) acquire(width, height int) *Image {
	key := textureKey{width: width, height: height}
	images := p.free[key]
	if len(images) > 0 {
		img := images[len(images)-1]
		p.free[key] = images[:len(images)-1]
		return img
	}
	return NewImage(width, height)
}

func (p *texturePool) release(img *Image) {
	if img == nil {
		return
	}
	for i := range img.Pix {
		img.Pix[i] = 0
	}
	key := textureKey{width: img.Width, height: img.Height}
	p.free[key] = append(p.free[key], img)
}
