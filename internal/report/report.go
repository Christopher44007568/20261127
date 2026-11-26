package report

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"characterbounds/hierbox"
)

type CullArtifact struct {
	Pose         string               `json:"pose"`
	Camera       hierbox.Camera       `json:"camera"`
	Stats        hierbox.CullStats    `json:"stats"`
	Tested       []hierbox.TestedNode `json:"tested"`
	AllStatuses  []hierbox.TestedNode `json:"allStatuses"`
	Visible      []hierbox.BoneID     `json:"visibleBones"`
	Culled       []hierbox.BoneID     `json:"culledBones"`
	VisibleNames []string             `json:"visibleNames"`
	CulledNames  []string             `json:"culledNames"`
}

type IncrementalDemo struct {
	Description       string                `json:"description"`
	Frames            int                   `json:"frames"`
	NodeCount         int                   `json:"nodeCount"`
	MinBoundsUpdates  int                   `json:"minBoundsUpdates"`
	MaxBoundsUpdates  int                   `json:"maxBoundsUpdates"`
	MatrixUpdates     []int                 `json:"matrixUpdates"`
	BoundsUpdates     []int                 `json:"boundsUpdates"`
	StructuralRebuild int                   `json:"structuralRebuilds"`
	Stats             []hierbox.UpdateStats `json:"stats"`
}

type Bundle struct {
	GeneratedAt         string                `json:"generatedAt"`
	GoVersion           string                `json:"goVersion"`
	StandardLibraryOnly bool                  `json:"standardLibraryOnly"`
	PoseOrder           []string              `json:"poseOrder"`
	Poses               []*hierbox.PoseResult `json:"poses"`
	Culling             []CullArtifact        `json:"culling"`
	IncrementalTail     IncrementalDemo       `json:"incrementalTailDemo"`
	Files               map[string]string     `json:"files"`
}

func Generate(r *hierbox.Runner, outDir string) (*Bundle, error) {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return nil, err
	}
	bundle := &Bundle{
		GeneratedAt:         time.Now().Format(time.RFC3339),
		GoVersion:           runtime.Version(),
		StandardLibraryOnly: true,
		PoseOrder:           append([]string(nil), r.PoseOrder...),
		Files:               make(map[string]string),
	}
	camera := hierbox.RecommendedCullCamera()

	for _, name := range r.PoseOrder {
		res, err := r.EvaluatePose(name)
		if err != nil {
			return nil, err
		}
		bundle.Poses = append(bundle.Poses, res)

		overlay := OverlaySVG(r, res)
		overlayName := "overlay_" + name + ".svg"
		if err := writeFile(outDir, overlayName, []byte(overlay)); err != nil {
			return nil, err
		}
		bundle.Files["overlay_"+name] = overlayName

		cull := r.Hierarchy.Cull(camera)
		all := r.Hierarchy.ClassifyAll(camera)
		visibleNames := make([]string, 0, len(cull.VisibleBones))
		for _, id := range cull.VisibleBones {
			visibleNames = append(visibleNames, r.Hierarchy.Nodes[id].Name)
		}
		culledNames := make([]string, 0, len(cull.CulledBones))
		for _, id := range cull.CulledBones {
			culledNames = append(culledNames, r.Hierarchy.Nodes[id].Name)
		}
		bundle.Culling = append(bundle.Culling, CullArtifact{
			Pose: name, Camera: camera, Stats: cull.Stats, Tested: cull.Tested,
			AllStatuses: all, Visible: cull.VisibleBones, Culled: cull.CulledBones,
			VisibleNames: visibleNames, CulledNames: culledNames,
		})
	}

	treeItems := make([]NamedCull, 0, len(bundle.Culling))
	for _, c := range bundle.Culling {
		// Reconstruct the core result shape consumed by the SVG helper.
		core := hierbox.CullResult{Camera: c.Camera, Stats: c.Stats, Tested: c.Tested, VisibleBones: c.Visible, CulledBones: c.Culled}
		treeItems = append(treeItems, NamedCull{Pose: c.Pose, Result: core, All: c.AllStatuses})
	}
	if err := writeFile(outDir, "culling.svg", []byte(CullingTreeSVG(treeItems))); err != nil {
		return nil, err
	}
	bundle.Files["culling_svg"] = "culling.svg"

	chart := VolumeChartSVG(bundle.Poses)
	if err := writeFile(outDir, "volume_layers.svg", []byte(chart)); err != nil {
		return nil, err
	}
	if err := writeFile(outDir, "volume_layers.csv", []byte(VolumeCSV(bundle.Poses))); err != nil {
		return nil, err
	}
	bundle.Files["volume_svg"] = "volume_layers.svg"
	bundle.Files["volume_csv"] = "volume_layers.csv"

	demo, err := runIncrementalTailDemo(r)
	if err != nil {
		return nil, err
	}
	bundle.IncrementalTail = demo
	if err := writeFile(outDir, "update_trace.csv", []byte(UpdateTraceCSV(demo))); err != nil {
		return nil, err
	}
	bundle.Files["update_trace_csv"] = "update_trace.csv"

	if err := writeJSON(outDir, "culling.json", bundle.Culling); err != nil {
		return nil, err
	}
	if err := writeJSON(outDir, "result.json", bundle); err != nil {
		return nil, err
	}
	summary := buildSummary(bundle)
	if err := writeJSON(outDir, "summary.json", summary); err != nil {
		return nil, err
	}
	if err := writeFile(outDir, "index.html", []byte(IndexHTML(bundle))); err != nil {
		return nil, err
	}
	bundle.Files["index_html"] = "index.html"

	// Rewrite result/summary after adding the HTML entry to the file map.
	if err := writeJSON(outDir, "result.json", bundle); err != nil {
		return nil, err
	}
	if err := writeJSON(outDir, "summary.json", buildSummary(bundle)); err != nil {
		return nil, err
	}
	return bundle, nil
}

func writeFile(dir, name string, data []byte) error {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("report: write %s: %w", path, err)
	}
	return nil
}

func writeJSON(dir, name string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	return writeFile(dir, name, data)
}

func runIncrementalTailDemo(r *hierbox.Runner) (IncrementalDemo, error) {
	base := r.Poses[hierbox.PoseTailSwing]
	if err := r.Hierarchy.SetLocals(base); err != nil {
		return IncrementalDemo{}, err
	}
	if _, err := r.Hierarchy.Update(); err != nil {
		return IncrementalDemo{}, err
	}
	tail := []hierbox.BoneID{hierbox.BoneTail0, hierbox.BoneTail1, hierbox.BoneTail2, hierbox.BoneTailTip}
	demo := IncrementalDemo{
		Description: "After the base pose, oscillate only the four tail bones; matrix work follows the tail subtree and bounds work follows the tail plus root ancestor chain.",
		Frames:      12, NodeCount: len(r.Hierarchy.Nodes), MinBoundsUpdates: 1 << 30,
	}
	for frame := 0; frame < demo.Frames; frame++ {
		wave := -14 + 28*float64(frame)/float64(demo.Frames-1)
		for _, id := range tail {
			tr := base[id]
			tr.Rotation = base[id].Rotation.Mul(hierbox.AxisAngle(hierbox.V(0, 1, 0), degReport(wave)))
			if err := r.Hierarchy.SetLocal(id, tr); err != nil {
				return IncrementalDemo{}, err
			}
		}
		stats, err := r.Hierarchy.Update()
		if err != nil {
			return IncrementalDemo{}, err
		}
		demo.Stats = append(demo.Stats, stats)
		demo.MatrixUpdates = append(demo.MatrixUpdates, stats.MatrixUpdates)
		demo.BoundsUpdates = append(demo.BoundsUpdates, stats.BoundsUpdates)
		if stats.BoundsUpdates < demo.MinBoundsUpdates {
			demo.MinBoundsUpdates = stats.BoundsUpdates
		}
		if stats.BoundsUpdates > demo.MaxBoundsUpdates {
			demo.MaxBoundsUpdates = stats.BoundsUpdates
		}
		demo.StructuralRebuild += stats.StructuralRebuilds
	}
	if demo.MinBoundsUpdates == 1<<30 {
		demo.MinBoundsUpdates = 0
	}
	return demo, nil
}

func degReport(v float64) float64 { return v * 3.141592653589793 / 180 }
