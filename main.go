package main

import (
	"flag"
	"fmt"
)

func DefaultScenes() []Scene {
	return []Scene{
		{
			ID: "sphere", Title: "SMOOTH SPHERE (R=0.70)", Kind: KindSphere,
			Radius: 0.70, MaxDepth: 7, MinDepth: 4,
			Color: [3]float64{0.28, 0.55, 0.95}, Flat: false,
		},
		{
			ID: "box", Title: "HARD-EDGE BOX (B=0.62 X 0.46 X 0.54)", Kind: KindBox,
			Half: V(0.62, 0.46, 0.54), MaxDepth: 7, MinDepth: 4,
			Color: [3]float64{0.95, 0.58, 0.20}, Flat: true,
		},
		{
			ID: "shell", Title: "THIN-WALL BOX SHELL (T=0.060)", Kind: KindShell,
			Half: V(0.70, 0.58, 0.70), Thickness: 0.06, MaxDepth: 7, MinDepth: 4,
			Color: [3]float64{0.20, 0.78, 0.72}, Flat: true, Clip: true,
		},
	}
}

func main() {
	outDir := flag.String("out", "artifacts", "artifact output directory")
	flag.Parse()

	scenes := DefaultScenes()
	results := make([]*SceneResult, 0, len(scenes))
	for _, scene := range scenes {
		o := NewOctree(scene)
		top := ExtractTopology(o)
		naiveBoundary := BoundaryEdgeCount(top.NaiveTris)
		rawBoundary := BoundaryEdgeCount(top.Tris)
		top.RepairCracks(o)
		finalBoundary := BoundaryEdgeCount(top.Tris)
		res := BuildMetrics(scene, o, top)
		res.NaiveBoundary = naiveBoundary
		res.RawBoundary = rawBoundary
		res.FinalBoundary = finalBoundary
		results = append(results, res)
		fmt.Printf("%s leaves=%d quads=%d tris=%d naiveBoundary=%d rawBoundary=%d finalBoundary=%d maxErr=%.6f sharpBins=%d\n",
			scene.ID, len(o.Leaves), len(top.Quads), len(top.Tris),
			naiveBoundary, rawBoundary, finalBoundary, res.MaxSDF, len(res.SharpBins))
	}
	if err := WriteReport(*outDir, results); err != nil {
		panic(err)
	}
	if err := RenderAll(*outDir, results); err != nil {
		panic(err)
	}
}
