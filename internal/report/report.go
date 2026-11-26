package report

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"skinlab/internal/skin"
)

// Dataset is everything needed to render one reproducible experiment.
type Dataset struct {
	Mesh            skin.Mesh
	Bind            skin.Comparison
	Final           skin.Comparison
	Trajectory      []skin.TrajectorySample
	Markers         []int
	BindExact       bool
	BindMaxDistance float64
}

// Summary is also written to summary.json for downstream checks.
type Summary struct {
	BendDeg             float64 `json:"bend_deg"`
	TwistDeg            float64 `json:"twist_deg"`
	VertexCount         int     `json:"vertex_count"`
	QuadCount           int     `json:"quad_count"`
	RingCount           int     `json:"ring_count"`
	SegmentCount        int     `json:"segment_count"`
	BindPoseExact       bool    `json:"bind_pose_exact"`
	BindMaxError        float64 `json:"bind_max_error"`
	LBSMinAreaRatio     float64 `json:"lbs_min_area_ratio"`
	LBSMaxAreaRatio     float64 `json:"lbs_max_area_ratio"`
	DQMinAreaRatio      float64 `json:"dq_min_area_ratio"`
	DQMaxAreaRatio      float64 `json:"dq_max_area_ratio"`
	LBSMinQuadAreaRatio float64 `json:"lbs_min_quad_area_ratio"`
	LBSMaxQuadAreaRatio float64 `json:"lbs_max_quad_area_ratio"`
	DQMinQuadAreaRatio  float64 `json:"dq_min_quad_area_ratio"`
	DQMaxQuadAreaRatio  float64 `json:"dq_max_quad_area_ratio"`
	LBSMinAreaX         float64 `json:"lbs_min_area_x"`
	DQMaxAreaX          float64 `json:"dq_max_area_x"`
}

// Write creates the static HTML report and all machine-readable companion
// files. It uses only the Go standard library.
func Write(outDir string, dataset Dataset) (Summary, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return Summary{}, fmt.Errorf("create output directory: %w", err)
	}

	summary := summarize(dataset)
	quadExtrema := quadExtrema(dataset.Final)
	summary.LBSMinQuadAreaRatio = quadExtrema.lbsMin
	summary.LBSMaxQuadAreaRatio = quadExtrema.lbsMax
	summary.DQMinQuadAreaRatio = quadExtrema.dqMin
	summary.DQMaxQuadAreaRatio = quadExtrema.dqMax

	sceneSVG := RenderMeshComparisonSVG(dataset.Mesh, bindPoints(dataset.Mesh), dataset.Final)
	areaSVG := RenderAreaChartSVG(dataset.Final.Areas)
	trajectorySVG := RenderTrajectorySVG(dataset.Trajectory, dataset.Markers)
	heatmapSVG := RenderHeatmapSVG(dataset.Mesh, dataset.Final.Quads)

	page := pageData{
		Summary:       summary,
		SceneSVG:      template.HTML(sceneSVG),
		AreaSVG:       template.HTML(areaSVG),
		TrajectorySVG: template.HTML(trajectorySVG),
		HeatmapSVG:    template.HTML(heatmapSVG),
	}

	var pageBuffer bytes.Buffer
	if err := reportTemplate.Execute(&pageBuffer, page); err != nil {
		return Summary{}, fmt.Errorf("render report template: %w", err)
	}
	files := map[string][]byte{
		"index.html":       pageBuffer.Bytes(),
		"area_curve.csv":   areaCSV(dataset.Final.Areas),
		"trajectories.csv": trajectoryCSV(dataset.Trajectory),
		"deformation.csv":  heatmapCSV(dataset.Final.Quads),
		"bind_pose.csv":    bindPoseCSV(dataset),
	}
	summaryBytes, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return Summary{}, fmt.Errorf("encode summary: %w", err)
	}
	files["summary.json"] = append(summaryBytes, '\n')

	for name, contents := range files {
		path := filepath.Join(outDir, name)
		if err := os.WriteFile(path, contents, 0o644); err != nil {
			return Summary{}, fmt.Errorf("write %s: %w", path, err)
		}
	}
	return summary, nil
}

func summarize(dataset Dataset) Summary {
	summary := Summary{
		BendDeg:         roundDegrees(dataset.Final.Pose.Bend * 180 / math.Pi),
		TwistDeg:        roundDegrees(dataset.Final.Pose.Twist * 180 / math.Pi),
		VertexCount:     len(dataset.Mesh.Vertices),
		QuadCount:       len(dataset.Mesh.Quads),
		RingCount:       dataset.Mesh.RingCount,
		SegmentCount:    dataset.Mesh.Segments,
		BindPoseExact:   dataset.BindExact,
		BindMaxError:    dataset.BindMaxDistance,
		LBSMinAreaRatio: dataset.Final.MinLBSRatio,
		LBSMaxAreaRatio: dataset.Final.MaxLBSRatio,
		DQMinAreaRatio:  dataset.Final.MinDQRatio,
		DQMaxAreaRatio:  dataset.Final.MaxDQRatio,
	}
	for _, area := range dataset.Final.Areas {
		if area.LBSRatio == summary.LBSMinAreaRatio {
			summary.LBSMinAreaX = area.X
		}
		if area.DQRatio == summary.DQMaxAreaRatio {
			summary.DQMaxAreaX = area.X
		}
	}
	return summary
}

func roundDegrees(value float64) float64 {
	return math.Round(value*1e6) / 1e6
}

type extrema struct {
	lbsMin, lbsMax, dqMin, dqMax float64
}

func quadExtrema(comparison skin.Comparison) extrema {
	result := extrema{
		lbsMin: math.Inf(1), lbsMax: math.Inf(-1),
		dqMin: math.Inf(1), dqMax: math.Inf(-1),
	}
	for _, quad := range comparison.Quads {
		result.lbsMin = math.Min(result.lbsMin, quad.LBSRatio)
		result.lbsMax = math.Max(result.lbsMax, quad.LBSRatio)
		result.dqMin = math.Min(result.dqMin, quad.DQRatio)
		result.dqMax = math.Max(result.dqMax, quad.DQRatio)
	}
	return result
}

func bindPoints(mesh skin.Mesh) []skin.Vec3 {
	points := make([]skin.Vec3, len(mesh.Vertices))
	for index, vertex := range mesh.Vertices {
		points[index] = vertex.Bind
	}
	return points
}

func areaCSV(areas []skin.AreaSample) []byte {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write([]string{"ring", "bind_x", "bind_area", "lbs_area", "dq_area", "lbs_ratio", "dq_ratio", "mean_upper_weight", "mean_forearm_weight"})
	for _, area := range areas {
		_ = writer.Write([]string{
			strconv.Itoa(area.Ring),
			number(area.X), number(area.BindArea), number(area.LBSArea), number(area.DQArea),
			number(area.LBSRatio), number(area.DQRatio), number(area.MeanUpper), number(area.MeanFore),
		})
	}
	writer.Flush()
	return buffer.Bytes()
}

func trajectoryCSV(samples []skin.TrajectorySample) []byte {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write([]string{"angle_deg", "method", "vertex", "ring", "segment", "x", "y", "z", "upper_weight", "forearm_weight"})
	for _, sample := range samples {
		_ = writer.Write([]string{
			number(sample.AngleDeg), sample.Method, strconv.Itoa(sample.Vertex),
			strconv.Itoa(sample.Ring), strconv.Itoa(sample.Segment),
			number(sample.Position.X), number(sample.Position.Y), number(sample.Position.Z),
			number(sample.Weights[0]), number(sample.Weights[1]),
		})
	}
	writer.Flush()
	return buffer.Bytes()
}

func heatmapCSV(quads []skin.QuadSample) []byte {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write([]string{"ring", "segment", "lbs_quad_area_ratio", "dq_quad_area_ratio"})
	for _, quad := range quads {
		_ = writer.Write([]string{strconv.Itoa(quad.Ring), strconv.Itoa(quad.Segment), number(quad.LBSRatio), number(quad.DQRatio)})
	}
	writer.Flush()
	return buffer.Bytes()
}

func bindPoseCSV(dataset Dataset) []byte {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.Write([]string{"vertex", "ring", "segment", "lbs_x", "lbs_y", "lbs_z", "dq_x", "dq_y", "dq_z", "bit_exact"})
	for index, vertex := range dataset.Mesh.Vertices {
		lbs := dataset.Bind.LBS[index]
		dq := dataset.Bind.DQ[index]
		_ = writer.Write([]string{
			strconv.Itoa(index), strconv.Itoa(vertex.Ring), strconv.Itoa(vertex.Segment),
			number(lbs.X), number(lbs.Y), number(lbs.Z), number(dq.X), number(dq.Y), number(dq.Z),
			strconv.FormatBool(lbs == dq),
		})
	}
	writer.Flush()
	return buffer.Bytes()
}

func number(value float64) string {
	return strconv.FormatFloat(value, 'f', 8, 64)
}
