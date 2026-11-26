package postfx

import "fmt"

const (
	ResHDR          ResourceID = "hdr"
	ResDepth        ResourceID = "depth"
	ResBloomBright  ResourceID = "bloom-bright"
	ResBloomH       ResourceID = "bloom-h"
	ResBloom        ResourceID = "bloom"
	ResBloomPreview ResourceID = "bloom-preview-image"
	ResFogDensity   ResourceID = "fog-density"
	ResFog          ResourceID = "fog"
	ResFogDebug     ResourceID = "fog-debug"
	ResFogPreview   ResourceID = "fog-preview-image"
	ResUI           ResourceID = "ui"
	ResUIDebug      ResourceID = "ui-debug"
	ResUIPreview    ResourceID = "ui-preview-image"
	ResBeauty       ResourceID = "beauty"
	ResFinal        ResourceID = "final"
)

const (
	PathCore    = "core"
	PathBloom   = "bloom"
	PathFog     = "volumetric-fog"
	PathUI      = "ui"
	PathPreview = "preview"
)

type Comparison struct {
	Full            *Image
	Culled          *Image
	Difference      *Image
	FullReport      ExecutionReport
	CulledReport    ExecutionReport
	PreviewImages   map[ResourceID]*Image
	PreviewReport   ExecutionReport
	MaxDifference   float64
	PixelIdentical  bool
	DisabledReturns []ResourceID
}

func BuildDemoGraph(width, height int) (*Graph, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("postfx: invalid demo size %dx%d", width, height)
	}
	g := NewGraph()
	for _, id := range []ResourceID{
		ResHDR, ResDepth, ResBloomBright, ResBloomH, ResBloom,
		ResBloomPreview, ResFogDensity, ResFog, ResFogDebug,
		ResFogPreview, ResUI, ResUIDebug, ResUIPreview, ResBeauty, ResFinal,
	} {
		if err := g.AddTexture(id, TextureSpec{Width: width, Height: height}); err != nil {
			return nil, err
		}
	}

	passes := []*Pass{
		{
			ID:      "scene",
			Name:    "Scene raster",
			Path:    PathCore,
			Outputs: []ResourceID{ResHDR, ResDepth},
			Run:     runScene,
		},
		{
			ID:      "bloom-threshold",
			Name:    "Bloom threshold",
			Path:    PathBloom,
			Inputs:  []ResourceID{ResHDR},
			Outputs: []ResourceID{ResBloomBright},
			Run:     runBloomThreshold,
		},
		{
			ID:      "bloom-blur-h",
			Name:    "Bloom blur horizontal",
			Path:    PathBloom,
			Inputs:  []ResourceID{ResBloomBright},
			Outputs: []ResourceID{ResBloomH},
			Run:     runBloomBlurH,
		},
		{
			ID:      "bloom-blur-v",
			Name:    "Bloom blur vertical",
			Path:    PathBloom,
			Inputs:  []ResourceID{ResBloomH},
			Outputs: []ResourceID{ResBloom},
			Run:     runBloomBlurV,
		},
		{
			ID:      "bloom-preview",
			Name:    "Bloom path preview",
			Path:    PathPreview,
			Inputs:  []ResourceID{ResBloom},
			Outputs: []ResourceID{ResBloomPreview},
			Run:     runBloomPreview,
		},
		{
			ID:      "fog-density",
			Name:    "Fog density",
			Path:    PathFog,
			Inputs:  []ResourceID{ResDepth},
			Outputs: []ResourceID{ResFogDensity},
			Run:     runFogDensity,
		},
		{
			ID:      "fog-scatter",
			Name:    "Fog light scattering",
			Path:    PathFog,
			Inputs:  []ResourceID{ResFogDensity, ResDepth},
			Outputs: []ResourceID{ResFog, ResFogDebug},
			Run:     runFogScatter,
		},
		{
			ID:      "fog-preview",
			Name:    "Fog path preview",
			Path:    PathPreview,
			Inputs:  []ResourceID{ResFogDensity, ResFog},
			Outputs: []ResourceID{ResFogPreview},
			Run:     runFogPreview,
		},
		{
			ID:        "ui-layer",
			Name:      "UI layer",
			Path:      PathUI,
			WeakAfter: []PassID{"bloom-blur-v"},
			Outputs:   []ResourceID{ResUI, ResUIDebug},
			Run:       runUI,
		},
		{
			ID:      "ui-preview",
			Name:    "UI path preview",
			Path:    PathPreview,
			Inputs:  []ResourceID{ResUI},
			Outputs: []ResourceID{ResUIPreview},
			Run:     runUIPreview,
		},
		{
			ID:      "beauty",
			Name:    "Composite post effects",
			Path:    PathCore,
			Inputs:  []ResourceID{ResHDR, ResBloom, ResFogDensity, ResFog, ResUI},
			Outputs: []ResourceID{ResBeauty},
			Run:     runBeauty,
		},
		{
			ID:      "tonemap",
			Name:    "Tonemap",
			Path:    PathCore,
			Inputs:  []ResourceID{ResBeauty},
			Outputs: []ResourceID{ResFinal},
			Run:     runTonemap,
		},
	}
	for _, pass := range passes {
		if err := g.AddPass(pass); err != nil {
			return nil, err
		}
	}
	if err := g.Validate(); err != nil {
		return nil, err
	}
	return g, nil
}

func RenderComparison(width, height int) (*Comparison, error) {
	graph, err := BuildDemoGraph(width, height)
	if err != nil {
		return nil, err
	}
	disabled := map[ResourceID]bool{
		ResUIDebug:  true,
		ResFogDebug: true,
	}
	base := ExecuteOptions{
		Roots:           []ResourceID{ResFinal},
		DisabledOutputs: disabled,
	}

	fullOptions := base
	fullOptions.Mode = ExecuteAllActive
	fullFrame, err := graph.Execute(fullOptions)
	if err != nil {
		return nil, fmt.Errorf("postfx: full execution: %w", err)
	}

	culledOptions := base
	culledOptions.Mode = ExecuteReachable
	culledFrame, err := graph.Execute(culledOptions)
	if err != nil {
		return nil, fmt.Errorf("postfx: culled execution: %w", err)
	}

	full := fullFrame.Images[ResFinal]
	culled := culledFrame.Images[ResFinal]
	if full == nil || culled == nil {
		return nil, fmt.Errorf("postfx: final image missing")
	}
	diff, err := AbsDiff(full, culled)
	if err != nil {
		return nil, err
	}
	maxDiff, err := MaxDiff(full, culled)
	if err != nil {
		return nil, err
	}

	previewOptions := ExecuteOptions{
		Mode:            ExecuteReachable,
		Roots:           []ResourceID{ResBloomPreview, ResFogPreview, ResUIPreview},
		DisabledOutputs: disabled,
	}
	previewFrame, err := graph.Execute(previewOptions)
	if err != nil {
		return nil, fmt.Errorf("postfx: preview execution: %w", err)
	}

	return &Comparison{
		Full:            full,
		Culled:          culled,
		Difference:      diff,
		FullReport:      fullFrame.Report,
		CulledReport:    culledFrame.Report,
		PreviewImages:   previewFrame.Images,
		PreviewReport:   previewFrame.Report,
		MaxDifference:   maxDiff,
		PixelIdentical:  maxDiff == 0,
		DisabledReturns: []ResourceID{ResFogDebug, ResUIDebug},
	}, nil
}
