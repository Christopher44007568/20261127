package report

type PoseSummary struct {
	Name               string  `json:"name"`
	Title              string  `json:"title"`
	Vertices           int     `json:"vertices"`
	Covered            bool    `json:"covered"`
	MaxOwnerOutside    float64 `json:"maxOwnerOutside"`
	MaxRootOutside     float64 `json:"maxRootOutside"`
	NaiveViolations    int     `json:"rootOnlyViolations"`
	NaiveMaxOutside    float64 `json:"rootOnlyMaxOutside"`
	MatrixUpdates      int     `json:"matrixUpdates"`
	BoundsUpdates      int     `json:"boundsUpdates"`
	StructuralRebuilds int     `json:"structuralRebuilds"`
}

type CullSummary struct {
	Pose              string `json:"pose"`
	FrustumTests      int    `json:"frustumTests"`
	TotalNodes        int    `json:"totalNodes"`
	VisibleBones      int    `json:"visibleBones"`
	CulledBones       int    `json:"culledBones"`
	SavedFrustumTests int    `json:"savedFrustumTests"`
}

type Summary struct {
	AllVerticesCovered  bool              `json:"allVerticesCovered"`
	StandardLibraryOnly bool              `json:"standardLibraryOnly"`
	Poses               []PoseSummary     `json:"poses"`
	Culling             []CullSummary     `json:"culling"`
	Incremental         IncrementalDemo   `json:"incrementalTailDemo"`
	Files               map[string]string `json:"files"`
}

func buildSummary(b *Bundle) Summary {
	s := Summary{
		AllVerticesCovered:  true,
		StandardLibraryOnly: b.StandardLibraryOnly,
		Incremental:         b.IncrementalTail,
		Files:               b.Files,
	}
	for _, p := range b.Poses {
		if !p.Coverage.Covered {
			s.AllVerticesCovered = false
		}
		s.Poses = append(s.Poses, PoseSummary{
			Name: p.Name, Title: poseTitle(p.Name), Vertices: p.Coverage.VertexCount,
			Covered: p.Coverage.Covered, MaxOwnerOutside: p.Coverage.MaxOwnerOutside,
			MaxRootOutside: p.Coverage.MaxRootOutside, NaiveViolations: p.RootOnly.Violations,
			NaiveMaxOutside: p.RootOnly.MaxOutside, MatrixUpdates: p.Stats.MatrixUpdates,
			BoundsUpdates: p.Stats.BoundsUpdates, StructuralRebuilds: p.Stats.StructuralRebuilds,
		})
	}
	for _, c := range b.Culling {
		s.Culling = append(s.Culling, CullSummary{
			Pose: c.Pose, FrustumTests: c.Stats.FrustumTests, TotalNodes: c.Stats.TotalNodes,
			VisibleBones: c.Stats.VisibleBones, CulledBones: c.Stats.CulledBones,
			SavedFrustumTests: c.Stats.SavedFrustumTests,
		})
	}
	return s
}
