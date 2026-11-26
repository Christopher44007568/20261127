package hierbox

import (
	"fmt"
	"testing"
)

func TestAllRequiredPosesCoverVertices(t *testing.T) {
	r, err := NewRunner()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range r.PoseOrder {
		result, err := r.EvaluatePose(name)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !result.Coverage.Covered || result.Coverage.HierarchyViolations != 0 {
			t.Fatalf("%s: %d owner, %d root, %d hierarchy violations; first=%+v",
				name, result.Coverage.OwnerViolations, result.Coverage.RootViolations,
				result.Coverage.HierarchyViolations, result.Coverage.Violations)
		}
		if result.Coverage.MaxOwnerOutside > 1e-6 || result.Coverage.MaxRootOutside > 1e-6 ||
			result.Coverage.MaxHierarchyOutside > 1e-6 {
			t.Fatalf("%s: unexpected box gap owner=%g root=%g hierarchy=%g",
				name, result.Coverage.MaxOwnerOutside, result.Coverage.MaxRootOutside,
				result.Coverage.MaxHierarchyOutside)
		}
		t.Logf("%s: vertices=%d maxOwnerGap=%.2e naiveViolations=%d",
			name, result.Coverage.VertexCount, result.Coverage.MaxOwnerOutside,
			result.RootOnly.Violations)
	}
}

func TestLeafChangeUpdatesOnlyAncestorBounds(t *testing.T) {
	r, err := NewRunner()
	if err != nil {
		t.Fatal(err)
	}
	ptrBefore := fmt.Sprintf("%p", &r.Hierarchy.Nodes[0])

	tr := r.RestPose[BoneTailTip]
	tr.Rotation = rotY(25)
	if err := r.Hierarchy.SetLocal(BoneTailTip, tr); err != nil {
		t.Fatal(err)
	}
	stats, err := r.Hierarchy.Update()
	if err != nil {
		t.Fatal(err)
	}

	wantBounds := []BoneID{BoneTailTip, BoneTail2, BoneTail1, BoneTail0, BoneRoot}
	if stats.BoundsUpdates != len(wantBounds) {
		t.Fatalf("bounds updates = %d (%v), want %d", stats.BoundsUpdates, stats.BoundsOrder, wantBounds)
	}
	if stats.MatrixUpdates != 1 {
		t.Fatalf("matrix updates = %d, want 1", stats.MatrixUpdates)
	}
	if stats.StructuralRebuilds != 0 || stats.FullTreeRebuilt {
		t.Fatalf("tree was rebuilt: %+v", stats)
	}
	ptrAfter := fmt.Sprintf("%p", &r.Hierarchy.Nodes[0])
	if ptrBefore != ptrAfter {
		t.Fatalf("node storage changed: %s -> %s", ptrBefore, ptrAfter)
	}

	if _, err := r.Hierarchy.Update(); err != nil {
		t.Fatal(err)
	}
	clean := r.Hierarchy.LastStats()
	if clean.BoundsUpdates != 0 || clean.MatrixUpdates != 0 {
		t.Fatalf("clean update did work: %+v", clean)
	}
}

func TestParentChangeUpdatesTopologySubtree(t *testing.T) {
	r, err := NewRunner()
	if err != nil {
		t.Fatal(err)
	}
	tr := r.RestPose[BoneTail0]
	tr.Rotation = rotY(20)
	if err := r.Hierarchy.SetLocal(BoneTail0, tr); err != nil {
		t.Fatal(err)
	}
	stats, err := r.Hierarchy.Update()
	if err != nil {
		t.Fatal(err)
	}

	if stats.MatrixUpdates != 4 {
		t.Fatalf("matrix updates = %d, want tail subtree size 4: %v", stats.MatrixUpdates, stats.ChangedBones)
	}
	if stats.BoundsUpdates != 5 {
		t.Fatalf("bounds updates = %d, want four tail nodes plus root: %v", stats.BoundsUpdates, stats.BoundsOrder)
	}
}
