package report

import (
	"fmt"
	"strings"

	"characterbounds/hierbox"
)

func UpdateTraceCSV(demo IncrementalDemo) string {
	var b strings.Builder
	b.WriteString("frame,matrix_updates,bounds_updates,structural_rebuilds,changed_bones,bounds_order\n")
	for frame, stats := range demo.Stats {
		fmt.Fprintf(&b, "%d,%d,%d,%d,%q,%q\n",
			frame, stats.MatrixUpdates, stats.BoundsUpdates, stats.StructuralRebuilds,
			joinBoneIDs(stats.ChangedBones), joinBoneIDs(stats.BoundsOrder))
	}
	return b.String()
}

func joinBoneIDs(ids []hierbox.BoneID) string {
	parts := make([]string, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("%d", id)
	}
	return strings.Join(parts, "|")
}
