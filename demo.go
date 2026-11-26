package main

import (
	"encoding/json"
	"image/color"
	"os"
	"path/filepath"
)

type ArtifactReport struct {
	Weight             int               `json:"weight"`
	RequestedDelta     float64           `json:"requested_delta"`
	Expansion          ExpansionResult   `json:"expansion"`
	Topology           TopologyReport    `json:"topology"`
	StressDelta        float64           `json:"stress_delta"`
	NaiveIntersections []Intersection    `json:"naive_intersections"`
	SizeMetrics        []SizeMetric      `json:"size_metrics"`
	Files              map[string]string `json:"files"`
}

var (
	colorBackground = color.NRGBA{R: 248, G: 245, B: 239, A: 255}
	colorInk        = color.NRGBA{R: 23, G: 42, B: 58, A: 255}
	colorOriginal   = color.NRGBA{R: 37, G: 99, B: 235, A: 255}
	colorExpanded   = color.NRGBA{R: 15, G: 118, B: 110, A: 255}
	colorSafe       = color.NRGBA{R: 21, G: 128, B: 61, A: 255}
	colorWarning    = color.NRGBA{R: 220, G: 38, B: 38, A: 255}
	colorAmber      = color.NRGBA{R: 245, G: 158, B: 11, A: 230}
)

func unionBounds(a, b Bounds) Bounds {
	r := a
	r.Min.X = minFloat(r.Min.X, b.Min.X)
	r.Min.Y = minFloat(r.Min.Y, b.Min.Y)
	r.Max.X = maxFloat(r.Max.X, b.Max.X)
	r.Max.Y = maxFloat(r.Max.Y, b.Max.Y)
	return r
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func drawVertices(cv *Canvas, c Contour, tr Transform, col color.NRGBA, radius float64) {
	for _, p := range c {
		cv.FillCircle(tr.Map(p), radius, col)
	}
}

func generateOriginal(path string, glyph Outline, samples int) error {
	const w, h = 980, 860
	cv := NewCanvas(w, h, colorBackground)
	tr := FitTransform(glyph.Bounds().Expanded(24), w, h, 54)
	cv.FillOutline(glyph, tr, colorInk, samples)
	cv.StrokeOutline(glyph, tr, colorOriginal, 2.2)
	drawVertices(cv, glyph.Outer, tr, color.NRGBA{R: 191, G: 219, B: 254, A: 255}, 3.2)
	for _, hole := range glyph.Holes {
		drawVertices(cv, hole, tr, color.NRGBA{R: 191, G: 219, B: 254, A: 255}, 3.2)
	}
	return SavePNG(cv.Image, path)
}

func generateExpanded(path string, glyph Outline, expansion ExpansionResult, samples int) error {
	const w, h = 980, 860
	cv := NewCanvas(w, h, colorBackground)
	bounds := unionBounds(glyph.Bounds(), expansion.Outline.Bounds()).Expanded(28)
	tr := FitTransform(bounds, w, h, 54)
	cv.FillOutline(expansion.Outline, tr, colorExpanded, samples)
	cv.StrokeOutline(expansion.Outline, tr, color.NRGBA{R: 204, G: 251, B: 241, A: 220}, 2.0)
	cv.StrokeOutline(glyph, tr, colorAmber, 3.0)
	return SavePNG(cv.Image, path)
}

func generateHighlight(path string, glyph Outline, safe ExpansionResult, stressDelta float64, samples int) ([]Intersection, error) {
	const w, h = 1040, 880
	cv := NewCanvas(w, h, color.NRGBA{R: 255, G: 247, B: 237, A: 255})
	bad := Outline{
		Outer: NaiveMiterContour(glyph.Outer, stressDelta),
		Holes: make([]Contour, len(glyph.Holes)),
	}
	for i, hole := range glyph.Holes {
		bad.Holes[i] = NaiveMiterContour(hole, stressDelta)
	}
	hits := FindOutlineIntersections(bad)
	limit := ExpandOutline(glyph, stressDelta, DefaultExpandOptions())

	bounds := unionBounds(glyph.Bounds(), bad.Bounds())
	bounds = unionBounds(bounds, limit.Outline.Bounds()).Expanded(34)
	tr := FitTransform(bounds, w, h, 58)

	cv.FillOutline(bad, tr, color.NRGBA{R: 220, G: 38, B: 38, A: 58}, samples)
	cv.StrokeOutline(bad, tr, colorWarning, 3.2)
	cv.StrokeOutline(limit.Outline, tr, colorSafe, 4.2)
	cv.StrokeOutline(safe.Outline, tr, color.NRGBA{R: 14, G: 116, B: 144, A: 180}, 2.2)
	for _, hit := range hits {
		cv.DrawMarker(hit.Point, tr)
	}
	return hits, SavePNG(cv.Image, path)
}

// GenerateArtifacts runs the demo and writes PNG/CSV/JSON outputs.
func GenerateArtifacts(outDir string, weight int, samples int) (ArtifactReport, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return ArtifactReport{}, err
	}
	glyph := SampleGlyph()
	requested := WeightToDelta(weight)
	expansion := ExpandOutline(glyph, requested, DefaultExpandOptions())
	topology := AnalyzeTopology(glyph, expansion.Outline)
	stressDelta := 47.5

	files := map[string]string{
		"original":     filepath.Join(outDir, "original_outline.png"),
		"expanded":     filepath.Join(outDir, "expanded_outline.png"),
		"highlight":    filepath.Join(outDir, "self_intersection_highlight.png"),
		"size_csv":     filepath.Join(outDir, "font_size_continuity.csv"),
		"area_svg":     filepath.Join(outDir, "font_size_area.svg"),
		"metrics_json": filepath.Join(outDir, "weight_metrics.json"),
	}

	if err := generateOriginal(files["original"], glyph, samples); err != nil {
		return ArtifactReport{}, err
	}
	if err := generateExpanded(files["expanded"], glyph, expansion, samples); err != nil {
		return ArtifactReport{}, err
	}
	hits, err := generateHighlight(files["highlight"], glyph, expansion, stressDelta, samples)
	if err != nil {
		return ArtifactReport{}, err
	}

	sizes := make([]float64, 0, 117)
	for size := 12.0; size <= 128.0+1e-9; size += 1.0 {
		sizes = append(sizes, size)
	}
	metrics := MeasureSizeSeries(glyph, weight, sizes, DefaultExpandOptions())
	csvFile, err := os.Create(files["size_csv"])
	if err != nil {
		return ArtifactReport{}, err
	}
	if err := WriteSizeCSV(csvFile, metrics); err != nil {
		csvFile.Close()
		return ArtifactReport{}, err
	}
	if err := csvFile.Close(); err != nil {
		return ArtifactReport{}, err
	}
	if err := WriteAreaChartSVG(files["area_svg"], metrics); err != nil {
		return ArtifactReport{}, err
	}

	report := ArtifactReport{
		Weight:             weight,
		RequestedDelta:     requested,
		Expansion:          expansion,
		Topology:           topology,
		StressDelta:        stressDelta,
		NaiveIntersections: hits,
		SizeMetrics:        metrics,
		Files:              files,
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return ArtifactReport{}, err
	}
	if err := os.WriteFile(files["metrics_json"], data, 0o644); err != nil {
		return ArtifactReport{}, err
	}
	return report, nil
}
