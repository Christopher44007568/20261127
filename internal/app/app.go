package app

import (
	"flag"
	"fmt"

	"characterbounds/hierbox"
	"characterbounds/internal/report"
)

// Run executes the command-line report generator.
func Run(argv []string) error {
	fs := flag.NewFlagSet("characterbounds", flag.ContinueOnError)
	out := fs.String("out", "artifacts", "directory for SVG, JSON, CSV and HTML reports")
	if err := fs.Parse(argv); err != nil {
		return err
	}

	runner, err := hierbox.NewRunner()
	if err != nil {
		return err
	}
	bundle, err := report.Generate(runner, *out)
	if err != nil {
		return err
	}

	fmt.Printf("generated reports in %s\n", *out)
	for _, pose := range bundle.Poses {
		fmt.Printf("  %-10s vertices=%d covered=%v ownerGap=%.2e rootOnlyLeaks=%d\n",
			pose.Name, pose.Coverage.VertexCount, pose.Coverage.Covered,
			pose.Coverage.MaxOwnerOutside, pose.RootOnly.Violations)
	}
	fmt.Printf("incremental tail: bounds updates %d-%d of %d nodes per frame; structural rebuilds=%d\n",
		bundle.IncrementalTail.MinBoundsUpdates, bundle.IncrementalTail.MaxBoundsUpdates,
		bundle.IncrementalTail.NodeCount, bundle.IncrementalTail.StructuralRebuild)
	for _, c := range bundle.Culling {
		fmt.Printf("cull %-10s tests=%d/%d visible=%d culled=%d saved=%d\n",
			c.Pose, c.Stats.FrustumTests, c.Stats.TotalNodes,
			c.Stats.VisibleBones, c.Stats.CulledBones, c.Stats.SavedFrustumTests)
	}
	return nil
}
