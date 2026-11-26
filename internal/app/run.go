package app

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"meshdiagnostics/internal/diag"
	"meshdiagnostics/internal/mesh"
	"meshdiagnostics/internal/qmesh"
	"meshdiagnostics/internal/remesh"
	"meshdiagnostics/internal/render"
	"meshdiagnostics/internal/sample"
)

// Options configures a complete diagnostic run.
type Options struct {
	InputPath      string
	OutDir         string
	CreaseAngleDeg float64
	CollapseDeg    float64
	Smooth         bool
	Width          int
	Height         int
	SuperSample    int
}

// Result contains the in-memory products and paths for reporting.
type Result struct {
	Source     *mesh.TriangleMesh
	Quad       *qmesh.Mesh
	Analysis   *diag.Analysis
	Remesh     remesh.Stats
	Features   mesh.FeatureInfo
	SourcePath string
	Files      []string
	Duration   time.Duration
}

// Run executes the full mesh diagnostic pipeline.
func Run(opt Options) (*Result, error) {
	started := time.Now()
	if opt.OutDir == "" {
		opt.OutDir = "out"
	}
	if opt.CreaseAngleDeg <= 0 {
		opt.CreaseAngleDeg = 40
	}
	opt.CreaseAngleDeg = remesh.ClampCreaseAngle(opt.CreaseAngleDeg)
	if opt.CollapseDeg <= 0 || opt.CollapseDeg >= 90 {
		opt.CollapseDeg = 18
	}
	if opt.Width <= 0 {
		opt.Width = 1200
	}
	if opt.Height <= 0 {
		opt.Height = 900
	}
	if opt.SuperSample <= 0 {
		opt.SuperSample = 2
	}
	if err := os.MkdirAll(opt.OutDir, 0o755); err != nil {
		return nil, err
	}

	var src *mesh.TriangleMesh
	var err error
	if opt.InputPath != "" {
		src, err = mesh.ReadOBJ(opt.InputPath)
		if err != nil {
			return nil, err
		}
		src.Orient()
	} else {
		src = sample.Organic()
	}
	if len(src.Faces) == 0 || len(src.Vertices) == 0 {
		return nil, fmt.Errorf("input mesh is empty")
	}

	quad, rs, feats := remesh.Convert(src, remesh.Options{
		CreaseAngleDeg: opt.CreaseAngleDeg,
		Smooth:         opt.Smooth,
	})
	if err := quad.Validate(); err != nil {
		return nil, fmt.Errorf("quad remesh validation: %w", err)
	}
	analysis := diag.Analyze(quad, opt.CollapseDeg)

	result := &Result{
		Source: src, Quad: quad, Analysis: analysis, Remesh: rs, Features: feats,
	}
	sourcePath := filepath.Join(opt.OutDir, "source_mesh.obj")
	if err := src.WriteOBJFile(sourcePath); err != nil {
		return nil, err
	}
	result.SourcePath = sourcePath
	result.Files = append(result.Files, sourcePath)

	quadPath := filepath.Join(opt.OutDir, "quad_mesh.obj")
	if err := quad.WriteOBJ(quadPath); err != nil {
		return nil, err
	}
	result.Files = append(result.Files, quadPath)

	rOpt := render.RenderOptions{
		Width: opt.Width, Height: opt.Height, CollapseDeg: opt.CollapseDeg,
		SuperSample: opt.SuperSample,
	}
	renders := []struct {
		name string
		fn   func(string) error
	}{
		{"original.png", func(p string) error { return render.RenderOriginal(p, src, feats, rOpt) }},
		{"quad_mesh.png", func(p string) error { return render.RenderQuad(p, quad, analysis, rs, rOpt) }},
		{"curvature_bands.png", func(p string) error { return render.RenderCurvature(p, quad, analysis, rOpt) }},
		{"local_zoom.png", func(p string) error { return render.RenderZoom(p, quad, analysis, rs, rOpt) }},
	}
	for _, item := range renders {
		path := filepath.Join(opt.OutDir, item.name)
		if err := item.fn(path); err != nil {
			return nil, fmt.Errorf("%s: %w", item.name, err)
		}
		result.Files = append(result.Files, path)
	}

	triEdges := triangleEdgeLengths(src)
	histPath := filepath.Join(opt.OutDir, "edge_length_histogram.png")
	if err := render.RenderHistogram(histPath, triEdges, analysis.EdgeLengths, analysis, rs, opt.CollapseDeg); err != nil {
		return nil, err
	}
	result.Files = append(result.Files, histPath)

	if err := writeEdgeCSV(filepath.Join(opt.OutDir, "edge_lengths.csv"), src, quad, feats); err != nil {
		return nil, err
	}
	result.Files = append(result.Files, filepath.Join(opt.OutDir, "edge_lengths.csv"))
	if err := writeFaceCSV(filepath.Join(opt.OutDir, "quad_face_metrics.csv"), quad, analysis); err != nil {
		return nil, err
	}
	result.Files = append(result.Files, filepath.Join(opt.OutDir, "quad_face_metrics.csv"))
	if err := writeFeatureCSV(filepath.Join(opt.OutDir, "feature_edges.csv"), src, quad, feats); err != nil {
		return nil, err
	}
	result.Files = append(result.Files, filepath.Join(opt.OutDir, "feature_edges.csv"))
	if err := writeHistogramCSV(filepath.Join(opt.OutDir, "edge_length_histogram.csv"), triEdges, analysis.EdgeLengths, 30); err != nil {
		return nil, err
	}
	result.Files = append(result.Files, filepath.Join(opt.OutDir, "edge_length_histogram.csv"))

	reportPath := filepath.Join(opt.OutDir, "report.txt")
	result.Duration = time.Since(started)
	if err := writeReport(reportPath, opt, result, triEdges); err != nil {
		return nil, err
	}
	result.Files = append(result.Files, reportPath)
	result.Duration = time.Since(started)
	return result, nil
}

func triangleEdgeLengths(m *mesh.TriangleMesh) []float64 {
	edges := m.EdgeMap()
	out := make([]float64, 0, len(edges))
	for key := range edges {
		out = append(out, m.EdgeLength(key))
	}
	sort.Float64s(out)
	return out
}

func writeEdgeCSV(path string, tri *mesh.TriangleMesh, quad *qmesh.Mesh, features mesh.FeatureInfo) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.Write([]string{"mesh", "a", "b", "length", "feature"}); err != nil {
		return err
	}
	triKeys := sortedEdgeKeys(tri.EdgeMap())
	for _, key := range triKeys {
		length := tri.EdgeLength(key)
		feat := "0"
		if features.Edges[key] {
			feat = "1"
		}
		_ = w.Write([]string{"triangle", strconv.Itoa(key.A), strconv.Itoa(key.B),
			fmt.Sprintf("%.9g", length), feat})
	}
	quadEdges := quad.Edges()
	quadKeys := make([]mesh.EdgeKey, 0, len(quadEdges))
	for key := range quadEdges {
		quadKeys = append(quadKeys, key)
	}
	sortEdgeKeys(quadKeys)
	for _, key := range quadKeys {
		length := quad.Vertices[key.A].Sub(quad.Vertices[key.B]).Len()
		feat := "0"
		if quad.Features[key] {
			feat = "1"
		}
		_ = w.Write([]string{"quad", strconv.Itoa(key.A), strconv.Itoa(key.B),
			fmt.Sprintf("%.9g", length), feat})
	}
	w.Flush()
	return w.Error()
}

func sortedEdgeKeys(edges map[mesh.EdgeKey]*mesh.EdgeFace) []mesh.EdgeKey {
	keys := make([]mesh.EdgeKey, 0, len(edges))
	for key := range edges {
		keys = append(keys, key)
	}
	sortEdgeKeys(keys)
	return keys
}

func sortEdgeKeys(keys []mesh.EdgeKey) {
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].A != keys[j].A {
			return keys[i].A < keys[j].A
		}
		return keys[i].B < keys[j].B
	})
}

func writeFaceCSV(path string, q *qmesh.Mesh, a *diag.Analysis) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"face", "min_angle_deg", "max_angle_deg", "angle0", "angle1",
		"angle2", "angle3", "convex", "curvature_band", "area", "planarity"})
	for i, m := range a.FaceMetrics {
		_ = w.Write([]string{
			strconv.Itoa(i), ftoa(m.MinAngleDeg), ftoa(m.MaxAngleDeg),
			ftoa(m.AnglesDeg[0]), ftoa(m.AnglesDeg[1]), ftoa(m.AnglesDeg[2]), ftoa(m.AnglesDeg[3]),
			boolInt(m.Convex), strconv.Itoa(a.CurvBands[i]), ftoa(m.Area), ftoa(m.Planarity),
		})
	}
	w.Flush()
	return w.Error()
}

func writeFeatureCSV(path string, tri *mesh.TriangleMesh, quad *qmesh.Mesh, features mesh.FeatureInfo) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.Write([]string{"mesh", "a", "b", "length"}); err != nil {
		return err
	}
	keys := make([]mesh.EdgeKey, 0, len(features.Edges))
	for key := range features.Edges {
		keys = append(keys, key)
	}
	sortEdgeKeys(keys)
	for _, key := range keys {
		_ = w.Write([]string{"triangle", strconv.Itoa(key.A), strconv.Itoa(key.B),
			ftoa(tri.EdgeLength(key))})
	}
	keys = keys[:0]
	for key := range quad.Features {
		keys = append(keys, key)
	}
	sortEdgeKeys(keys)
	for _, key := range keys {
		_ = w.Write([]string{"quad", strconv.Itoa(key.A), strconv.Itoa(key.B),
			ftoa(quad.Vertices[key.A].Sub(quad.Vertices[key.B]).Len())})
	}
	w.Flush()
	return w.Error()
}

func writeHistogramCSV(path string, tri, quad []float64, bins int) error {
	lo, hi := 0.0, 0.0
	if len(tri) > 0 {
		lo, hi = tri[0], tri[len(tri)-1]
	}
	if len(quad) > 0 {
		if len(tri) == 0 || quad[0] < lo {
			lo = quad[0]
		}
		if len(tri) == 0 || quad[len(quad)-1] > hi {
			hi = quad[len(quad)-1]
		}
	}
	if hi-lo < 1e-14 {
		hi = lo + 1
	}
	width := (hi - lo) / float64(bins)
	triCounts := binCounts(tri, lo, width, bins)
	quadCounts := binCounts(quad, lo, width, bins)
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	_ = w.Write([]string{"bin_lo", "bin_hi", "tri_count", "quad_count", "tri_density", "quad_density"})
	for i := 0; i < bins; i++ {
		triD := 0.0
		if len(tri) > 0 {
			triD = float64(triCounts[i]) / (float64(len(tri)) * width)
		}
		quadD := 0.0
		if len(quad) > 0 {
			quadD = float64(quadCounts[i]) / (float64(len(quad)) * width)
		}
		_ = w.Write([]string{
			ftoa(lo + float64(i)*width), ftoa(lo + float64(i+1)*width),
			strconv.Itoa(triCounts[i]), strconv.Itoa(quadCounts[i]), ftoa(triD), ftoa(quadD),
		})
	}
	w.Flush()
	return w.Error()
}

func binCounts(values []float64, lo, width float64, bins int) []int {
	out := make([]int, bins)
	for _, v := range values {
		i := int((v - lo) / width)
		if i < 0 {
			i = 0
		}
		if i >= bins {
			i = bins - 1
		}
		out[i]++
	}
	return out
}

func ftoa(v float64) string {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return "nan"
	}
	return strconv.FormatFloat(v, 'f', 6, 64)
}

func boolInt(v bool) string {
	if v {
		return "1"
	}
	return "0"
}

func writeReport(path string, opt Options, r *Result, triEdges []float64) error {
	triStats := diag.Summarize(triEdges)
	var b strings.Builder
	fmt.Fprintf(&b, "MESH DIAGNOSTICS REPORT\n")
	fmt.Fprintf(&b, "generated: %s\n", time.Now().Format(time.RFC3339))
	if opt.InputPath == "" {
		fmt.Fprintf(&b, "input: procedural organic sample\n")
	} else {
		fmt.Fprintf(&b, "input: %s\n", opt.InputPath)
	}
	fmt.Fprintf(&b, "crease_threshold_deg: %.2f\n", opt.CreaseAngleDeg)
	fmt.Fprintf(&b, "collapse_threshold_deg: %.2f\n", opt.CollapseDeg)
	fmt.Fprintf(&b, "smoothing: %t\n\n", opt.Smooth)

	fmt.Fprintf(&b, "[source triangle mesh]\n")
	fmt.Fprintf(&b, "vertices: %d\ntriangles: %d\n", len(r.Source.Vertices), len(r.Source.Faces))
	fmt.Fprintf(&b, "feature_edges: %d (boundary=%d crease=%d nonmanifold=%d)\n",
		len(r.Features.Edges), r.Features.Boundary, r.Features.Crease, r.Features.NonManifold)
	fmt.Fprintf(&b, "\n[quad mesh]\n")
	fmt.Fprintf(&b, "vertices: %d\nquads: %d\nedges: %d\n", len(r.Quad.Vertices),
		len(r.Quad.Faces), r.Remesh.QuadEdges)
	fmt.Fprintf(&b, "feature_segments: %d\n", r.Remesh.QuadFeatures)
	fmt.Fprintf(&b, "feature_original_length: %.9f\n", r.Remesh.OriginalFeatureLen)
	fmt.Fprintf(&b, "feature_quad_length: %.9f\n", r.Remesh.QuadFeatureLen)
	fmt.Fprintf(&b, "feature_length_ratio: %.6f\n\n", r.Remesh.FeatureLengthRatio)

	fmt.Fprintf(&b, "[quad curvature bands]\n")
	fmt.Fprintf(&b, "mean_abs_curvature=%.6f p95=%.6f max=%.6f\n", r.Analysis.CurvStats.Mean,
		r.Analysis.CurvStats.P95, r.Analysis.CurvStats.Max)
	for i, t := range r.Analysis.CurvThresholds {
		fmt.Fprintf(&b, "quantile_band_%d_threshold: %.6f\n", i, t)
	}
	fmt.Fprintf(&b, "\n")

	fmt.Fprintf(&b, "[edge length diagnostic]\n")
	fmt.Fprintf(&b, "quad edges: min=%.6f p5=%.6f median=%.6f p95=%.6f max=%.6f mean=%.6f cv=%.4f\n",
		r.Analysis.EdgeStats.Min, r.Analysis.EdgeStats.P5, r.Analysis.EdgeStats.Median,
		r.Analysis.EdgeStats.P95, r.Analysis.EdgeStats.Max, r.Analysis.EdgeStats.Mean, r.Analysis.EdgeStats.CV)
	fmt.Fprintf(&b, "triangle edges: min=%.6f p5=%.6f median=%.6f p95=%.6f max=%.6f mean=%.6f cv=%.4f\n\n",
		triStats.Min, triStats.P5, triStats.Median, triStats.P95, triStats.Max, triStats.Mean, triStats.CV)

	fmt.Fprintf(&b, "[quad angle collapse]\n")
	fmt.Fprintf(&b, "collapsed_quads: %d / %d (%.3f%%)\n", len(r.Analysis.Collapsed),
		r.Analysis.FaceCount, float64(len(r.Analysis.Collapsed))*100/float64(r.Analysis.FaceCount))
	fmt.Fprintf(&b, "worst_face: %d\n", r.Analysis.WorstFace)
	fmt.Fprintf(&b, "worst_min_angle_deg: %.4f\n", r.Analysis.WorstMetrics.MinAngleDeg)
	fmt.Fprintf(&b, "worst_max_angle_deg: %.4f\n", r.Analysis.WorstMetrics.MaxAngleDeg)
	fmt.Fprintf(&b, "worst_convex: %t\n", r.Analysis.WorstMetrics.Convex)
	fmt.Fprintf(&b, "corner_angle_min=%.4f p5=%.4f median=%.4f p95=%.4f max=%.4f\n\n",
		r.Analysis.AngleStats.Min, r.Analysis.AngleStats.P5, r.Analysis.AngleStats.Median,
		r.Analysis.AngleStats.P95, r.Analysis.AngleStats.Max)

	fmt.Fprintf(&b, "[outputs]\n")
	for _, p := range r.Files {
		fmt.Fprintf(&b, "%s\n", p)
	}
	fmt.Fprintf(&b, "elapsed: %s\n", r.Duration.Round(time.Millisecond))
	return os.WriteFile(path, []byte(b.String()), 0o644)
}
