package hierbox

import "testing"

func TestRecommendedCameraProducesHierarchyCulls(t *testing.T) {
	r, err := NewRunner()
	if err != nil {
		t.Fatal(err)
	}
	cam := RecommendedCullCamera()
	for _, name := range r.PoseOrder {
		if _, err := r.EvaluatePose(name); err != nil {
			t.Fatal(err)
		}
		res := r.Hierarchy.Cull(cam)
		t.Logf("%s: tested=%d visible=%d culled=%d saved=%d accepted=%d pruned=%d",
			name, res.Stats.FrustumTests, res.Stats.VisibleBones, res.Stats.CulledBones,
			res.Stats.SavedFrustumTests, res.Stats.AcceptedSubtrees, res.Stats.PrunedOutsideRoots)
		for _, n := range res.Tested {
			if n.Status != CullInside {
				t.Logf("  tested %s depth=%d status=%s", n.Name, n.Depth, n.Status)
			}
		}
		if res.Stats.CulledBones == 0 {
			t.Fatalf("%s: expected at least one culled bone", name)
		}
		if res.Stats.FrustumTests >= res.Stats.TotalNodes {
			t.Fatalf("%s: expected hierarchy culling to save tests", name)
		}
	}
}
