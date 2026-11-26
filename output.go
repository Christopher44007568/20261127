package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
)

type OutputFiles struct {
	SeedDistribution string `json:"seed_distribution"`
	RegionColors     string `json:"region_colors"`
	BoundaryLines    string `json:"boundary_lines"`
	DistanceHeatmap  string `json:"distance_heatmap"`
	BoundarySVG      string `json:"boundary_svg"`
	Manifest         string `json:"manifest"`
}

func WriteOutputs(dir string, result *Result) (OutputFiles, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return OutputFiles{}, err
	}
	files := OutputFiles{
		SeedDistribution: filepath.Join(dir, "seed_distribution.png"),
		RegionColors:     filepath.Join(dir, "region_colors.png"),
		BoundaryLines:    filepath.Join(dir, "boundary_lines.png"),
		DistanceHeatmap:  filepath.Join(dir, "distance_heatmap.png"),
		BoundarySVG:      filepath.Join(dir, "boundary_lines.svg"),
		Manifest:         filepath.Join(dir, "segmentation.json"),
	}

	images := map[string]image.Image{
		files.SeedDistribution: RenderSeedMap(result),
		files.RegionColors:     RenderRegionMap(result),
		files.BoundaryLines:    RenderBoundaryMap(result),
		files.DistanceHeatmap:  RenderHeatmap(result),
	}
	for path, img := range images {
		if err := savePNG(path, img); err != nil {
			return OutputFiles{}, err
		}
	}
	if err := os.WriteFile(files.BoundarySVG, []byte(RenderBoundarySVG(result)), 0o644); err != nil {
		return OutputFiles{}, err
	}
	manifest, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return OutputFiles{}, err
	}
	if err := os.WriteFile(files.Manifest, manifest, 0o644); err != nil {
		return OutputFiles{}, err
	}
	return files, nil
}

func savePNG(path string, img image.Image) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	encodeErr := png.Encode(file, img)
	closeErr := file.Close()
	if encodeErr != nil {
		return encodeErr
	}
	return closeErr
}

func RenderBoundarySVG(r *Result) string {
	g := r.Grid
	var b strings.Builder
	fmt.Fprintf(&b, `<svg xmlns="http://www.w3.org/2000/svg" width="%d" height="%d" viewBox="-1.5 -1.5 %d %d">`,
		(g.W+3)*4, (g.H+3)*4, g.W+3, g.H+3)
	b.WriteString(`<rect width="100%" height="100%" fill="#111827"/>`)

	for id, blocked := range g.Blocked {
		if !blocked {
			continue
		}
		x := id % g.W
		y := id / g.W
		fmt.Fprintf(&b, `<rect x="%d" y="%d" width="1" height="1" fill="#05070a"/>`, x, y)
	}

	segments := ExtractBoundarySegments(g, r.Labels)
	b.WriteString(`<g stroke="#000000" stroke-width="0.42" stroke-linecap="square">`)
	for _, s := range segments {
		fmt.Fprintf(&b, `<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f"/>`, s.X1, s.Y1, s.X2, s.Y2)
	}
	b.WriteString(`</g>`)
	b.WriteString(`<g stroke="#eef6ff" stroke-width="0.16" stroke-linecap="square">`)
	for _, s := range segments {
		fmt.Fprintf(&b, `<line x1="%.2f" y1="%.2f" x2="%.2f" y2="%.2f"/>`, s.X1, s.Y1, s.X2, s.Y2)
	}
	b.WriteString(`</g>`)

	for i, seed := range r.Seeds {
		fill := labelColor(i)
		fmt.Fprintf(&b, `<circle cx="%.2f" cy="%.2f" r="0.48" fill="#ffffff" stroke="#000000" stroke-width="0.12"/>`,
			float64(seed.X)+0.5, float64(seed.Y)+0.5)
		fmt.Fprintf(&b, `<circle cx="%.2f" cy="%.2f" r="0.28" fill="rgb(%d,%d,%d)"/>`,
			float64(seed.X)+0.5, float64(seed.Y)+0.5, fill.R, fill.G, fill.B)
	}
	b.WriteString(`</svg>`)
	return b.String()
}
