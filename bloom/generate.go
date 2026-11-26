package bloom

import (
	"fmt"
	"os"
	"path/filepath"
)

type Summary struct {
	Scene        string                  `json:"scene"`
	Width        int                     `json:"width"`
	Height       int                     `json:"height"`
	BloomChain   []string                `json:"bloom_chain"`
	OutputPolicy string                  `json:"output_policy"`
	ToneMapping  string                  `json:"ppm_tone_mapping"`
	HDRReference string                  `json:"hdr_reference"`
	Formats      []*FormatArtifactReport `json:"formats"`
}

// Generate renders the same graph once per format. The linear reference is
// evaluated once without any intermediate texture writes.
func Generate(root string, width, height int) (*Summary, error) {
	if root == "" {
		return nil, fmt.Errorf("output directory must not be empty")
	}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	graph, err := BuildBloomGraph(width, height)
	if err != nil {
		return nil, err
	}
	reference, err := RenderReference(graph)
	if err != nil {
		return nil, err
	}
	summary := &Summary{
		Scene: "procedural HDR exposure ramp, low-light gradient, color patches, and 150000-nit emitter",
		Width: width, Height: height,
		BloomChain:   []string{"source", "threshold", "blur.horizontal", "blur.vertical", "scale", "composite"},
		OutputPolicy: "clamp to the destination format and cache excess per channel; no cross-format buffers",
		ToneMapping:  "linear Reinhard then sRGB transfer for PPM/PNG display output",
		HDRReference: "float64 linear evaluation of the identical node graph",
	}
	for _, format := range AllFormats() {
		render, err := RenderFormat(graph, format, reference)
		if err != nil {
			return nil, fmt.Errorf("render %s: %w", format, err)
		}
		report, err := WriteFormatArtifacts(root, render)
		if err != nil {
			return nil, fmt.Errorf("write %s artifacts: %w", format, err)
		}
		if report.ClampedChannelValues == 0 {
			return nil, fmt.Errorf("format %s unexpectedly did not exercise range clamping", format)
		}
		summary.Formats = append(summary.Formats, report)
	}
	if err := writeJSON(filepath.Join(root, "summary.json"), summary); err != nil {
		return nil, err
	}
	return summary, nil
}
