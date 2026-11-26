package bloom

import (
	"fmt"
)

type FormatRender struct {
	Format             FormatID
	Plan               *Plan
	Execution          *Execution
	Reference          *Image
	FinalError         ErrorStats
	ErrorImage         *Image
	ErrorNormalization float64
}

func RenderReference(g *Graph) (*Image, error) {
	if g == nil {
		return nil, fmt.Errorf("nil graph")
	}
	plan, err := g.Negotiate(FormatRGBA16F)
	if err != nil {
		return nil, err
	}
	exec, err := g.ExecuteUnquantized(plan)
	if err != nil {
		return nil, err
	}
	return exec.Final.Linear, nil
}

func RenderFormat(g *Graph, format FormatID, reference *Image) (*FormatRender, error) {
	if g == nil {
		return nil, fmt.Errorf("nil graph")
	}
	if err := reference.Check(); err != nil {
		return nil, fmt.Errorf("reference image: %w", err)
	}
	plan, err := g.Negotiate(format)
	if err != nil {
		return nil, err
	}
	exec, err := g.Execute(plan)
	if err != nil {
		return nil, err
	}
	if exec.Final == nil || exec.Final.Texture == nil {
		return nil, fmt.Errorf("format %s did not produce an encoded final texture", format)
	}
	if exec.Final.Texture.Format != format {
		return nil, fmt.Errorf("final texture is %s, requested %s", exec.Final.Texture.Format, format)
	}
	for id, out := range exec.Nodes {
		if out.Texture == nil {
			return nil, fmt.Errorf("node %q has no encoded texture", id)
		}
		if out.Texture.Format != format {
			return nil, fmt.Errorf("node %q wrote %s into a %s graph", id, out.Texture.Format, format)
		}
	}
	finalError, err := CompareImages(reference, exec.Final.Linear)
	if err != nil {
		return nil, err
	}
	errImage, err := AbsoluteError(reference, exec.Final.Linear)
	if err != nil {
		return nil, err
	}
	normalization := 0.0
	for _, c := range errImage.Pixels {
		normalization = maxFloat(normalization, c.MaxComponent())
	}
	return &FormatRender{
		Format: format, Plan: plan, Execution: exec, Reference: reference,
		FinalError: finalError, ErrorImage: errImage, ErrorNormalization: normalization,
	}, nil
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
