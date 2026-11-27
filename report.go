package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func WriteReport(outDir string, results []*SceneResult) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	var b strings.Builder
	fmt.Fprintln(&b, "adaptive voxel isosurface - normal positioning / quad split / crack repair")
	fmt.Fprintln(&b, "domain [-1,1]^3  maxDepth 7  finest voxel h=2/128=0.015625")
	fmt.Fprintln(&b, "error = |SDF(vertex)|; sharp feature distance is to analytic box edge/corner network")
	fmt.Fprintln(&b)

	for _, r := range results {
		s := r.Scene
		fmt.Fprintf(&b, "== %s ==\n", s.ID)
		fmt.Fprintf(&b, "image: %s\n", filepath.Join(outDir, s.ID+".png"))
		fmt.Fprintf(&b, "leaf cells %d  depth distribution %s\n", r.LeafCount, formatHist(r.Histogram))
		fmt.Fprintf(&b, "vertices %d  quads %d  triangles %d (quad triangles=%d, crack patches=%d)\n",
			r.VertexCount, len(r.Top.Quads), len(r.Top.Tris), 2*len(r.Top.Quads), r.Top.RepairTriangles)
		fmt.Fprintf(&b, "boundary edges naive %d, before stitch %d, after crack repair %d\n",
			r.NaiveBoundary, r.RawBoundary, r.FinalBoundary)
		fmt.Fprintf(&b, "max |SDF(vertex)| normal-QEF %.6f, mass-point baseline %.6f\n", r.MaxSDF, r.BaselineMax)
		fmt.Fprintln(&b, "sharp feature local error grouped by local voxel size:")
		if len(r.SharpBins) == 0 {
			fmt.Fprintln(&b, "  no analytic sharp crease (smooth sphere control)")
			if w := r.WorstMarker; w != nil {
				fmt.Fprintf(&b, "  worst surface vertex: depth=%d h=%.4f err=%.6f\n", w.Depth, w.H, w.Err)
			}
		}
		for _, bin := range r.SharpBins {
			fmt.Fprintf(&b,
				"  depth=%d h=%.4f count=%d maxErr=%.6f meanErr=%.6f maxFeatureDist=%.6f meanFeatureDist=%.6f baselineMaxErr=%.6f\n",
				bin.Depth, bin.H, bin.Count, bin.MaxSDF, bin.MeanSDF(),
				bin.MaxFeatureDist, bin.MeanFeatureDist(), bin.BaselineMax)
		}
		if len(r.Markers) > 0 {
			fmt.Fprintln(&b, "  ground-truth corner markers:")
			for _, m := range r.Markers {
				kind := "outer"
				if m.Inner {
					kind = "inner"
				}
				fmt.Fprintf(&b, "    %-5s depth=%d h=%.4f sdfErr=%.6f featureDist=%.6f baselineErr=%.6f\n",
					kind, m.Depth, m.H, m.Err, m.FeatureDist, m.BaselineErr)
			}
		}
		fmt.Fprintln(&b)
	}
	return os.WriteFile(filepath.Join(outDir, "report.txt"), []byte(b.String()), 0o644)
}

func formatHist(h map[int]int) string {
	parts := []string{}
	for _, kv := range sortedDepthCounts(h) {
		parts = append(parts, fmt.Sprintf("%d:%d", kv.Depth, kv.Count))
	}
	return "map[" + strings.Join(parts, " ") + "]"
}
