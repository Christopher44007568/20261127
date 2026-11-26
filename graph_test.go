package postfx

import (
	"strings"
	"testing"
)

func testTexture(width, height int) TextureSpec {
	return TextureSpec{Width: width, Height: height}
}

func addTestTexture(t *testing.T, graph *Graph, id ResourceID) {
	t.Helper()
	if err := graph.AddTexture(id, testTexture(8, 6)); err != nil {
		t.Fatal(err)
	}
}

func TestReachableExecutionReleasesIntermediates(t *testing.T) {
	graph := NewGraph()
	for _, id := range []ResourceID{"a", "b", "c"} {
		addTestTexture(t, graph, id)
	}
	for _, pass := range []*Pass{
		{ID: "one", Outputs: []ResourceID{"a"}},
		{ID: "two", Inputs: []ResourceID{"a"}, Outputs: []ResourceID{"b"}},
		{ID: "three", Inputs: []ResourceID{"b"}, Outputs: []ResourceID{"c"}},
	} {
		if err := graph.AddPass(pass); err != nil {
			t.Fatal(err)
		}
	}
	frame, err := graph.Execute(ExecuteOptions{
		Mode:  ExecuteReachable,
		Roots: []ResourceID{"c"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(frame.Report.Executed) != 3 {
		t.Fatalf("expected 3 executed passes, got %v", frame.Report.Executed)
	}
	if frame.Report.TempTexturesAtEnd != 0 {
		t.Fatalf("intermediate textures leaked: %d", frame.Report.TempTexturesAtEnd)
	}
	if containsResource(frame.Report.ReleasedOutputs, "a") == false ||
		containsResource(frame.Report.ReleasedOutputs, "b") == false {
		t.Fatalf("expected a and b to be released, got %v", frame.Report.ReleasedOutputs)
	}
}

func TestDisabledReturnNeverAllocatesTexture(t *testing.T) {
	graph := NewGraph()
	for _, id := range []ResourceID{"visible", "debug"} {
		addTestTexture(t, graph, id)
	}
	if err := graph.AddPass(&Pass{
		ID:      "write-both",
		Outputs: []ResourceID{"visible", "debug"},
	}); err != nil {
		t.Fatal(err)
	}
	frame, err := graph.Execute(ExecuteOptions{
		Mode:            ExecuteAllActive,
		Roots:           []ResourceID{"visible"},
		DisabledOutputs: map[ResourceID]bool{"debug": true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if containsResource(frame.Report.AllocatedOutputs, "debug") {
		t.Fatalf("disabled return allocated a texture: %v", frame.Report.AllocatedOutputs)
	}
	if !containsResource(frame.Report.AllocatedOutputs, "visible") {
		t.Fatalf("enabled return was not allocated: %v", frame.Report.AllocatedOutputs)
	}
	if frame.Report.TempTexturesAtEnd != 0 {
		t.Fatalf("intermediate textures leaked: %d", frame.Report.TempTexturesAtEnd)
	}
}

func TestReachableModeDoesNotAllocateUnreadOutputs(t *testing.T) {
	graph := NewGraph()
	for _, id := range []ResourceID{"visible", "unused"} {
		addTestTexture(t, graph, id)
	}
	if err := graph.AddPass(&Pass{
		ID:      "write-both",
		Outputs: []ResourceID{"visible", "unused"},
	}); err != nil {
		t.Fatal(err)
	}
	frame, err := graph.Execute(ExecuteOptions{
		Mode:  ExecuteReachable,
		Roots: []ResourceID{"visible"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if containsResource(frame.Report.AllocatedOutputs, "unused") {
		t.Fatalf("unread output allocated a texture: %v", frame.Report.AllocatedOutputs)
	}
}

func TestWeakDependencyDoesNotKeepDeadPassAlive(t *testing.T) {
	graph := NewGraph()
	for _, id := range []ResourceID{"a", "b", "c"} {
		addTestTexture(t, graph, id)
	}
	for _, pass := range []*Pass{
		{ID: "a", Outputs: []ResourceID{"a"}},
		{ID: "b", WeakAfter: []PassID{"a"}, Outputs: []ResourceID{"b"}},
		{ID: "c", Inputs: []ResourceID{"a"}, Outputs: []ResourceID{"c"}},
	} {
		if err := graph.AddPass(pass); err != nil {
			t.Fatal(err)
		}
	}
	frame, err := graph.Execute(ExecuteOptions{
		Mode:  ExecuteReachable,
		Roots: []ResourceID{"c"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !containsNode(frame.Report, "a") || !containsNode(frame.Report, "c") {
		t.Fatalf("strongly read pass was culled: %v", frame.Report.Executed)
	}
	if containsNode(frame.Report, "b") {
		t.Fatalf("weak-only dependent should remain culled: %v", frame.Report.Executed)
	}
}

func TestWeakDependencyOrdersLivePasses(t *testing.T) {
	graph := NewGraph()
	for _, id := range []ResourceID{"a", "b", "c"} {
		addTestTexture(t, graph, id)
	}
	for _, pass := range []*Pass{
		{ID: "a", Outputs: []ResourceID{"a"}},
		{ID: "b", WeakAfter: []PassID{"a"}, Outputs: []ResourceID{"b"}},
		{ID: "c", Inputs: []ResourceID{"a", "b"}, Outputs: []ResourceID{"c"}},
	} {
		if err := graph.AddPass(pass); err != nil {
			t.Fatal(err)
		}
	}
	frame, err := graph.Execute(ExecuteOptions{
		Mode:  ExecuteReachable,
		Roots: []ResourceID{"c"},
	})
	if err != nil {
		t.Fatal(err)
	}
	if indexOfNode(frame.Report, "a") > indexOfNode(frame.Report, "b") {
		t.Fatalf("weak order a -> b was not preserved: %v", frame.Report.Executed)
	}
}

func TestDemoCullingIsPixelIdentical(t *testing.T) {
	comparison, err := RenderComparison(160, 90)
	if err != nil {
		t.Fatal(err)
	}
	if !comparison.PixelIdentical || comparison.MaxDifference != 0 {
		t.Fatalf("culling changed the final image: max diff %g", comparison.MaxDifference)
	}
	if len(comparison.CulledReport.Executed) >= len(comparison.FullReport.Executed) {
		t.Fatalf("culling did not remove work: full=%d culled=%d",
			len(comparison.FullReport.Executed), len(comparison.CulledReport.Executed))
	}
	for _, id := range comparison.DisabledReturns {
		if containsResource(comparison.FullReport.AllocatedOutputs, id) {
			t.Fatalf("disabled return %q allocated a texture", id)
		}
	}
	if comparison.FullReport.TempTexturesAtEnd != 0 ||
		comparison.CulledReport.TempTexturesAtEnd != 0 {
		t.Fatalf("intermediate textures leaked: full=%d culled=%d",
			comparison.FullReport.TempTexturesAtEnd, comparison.CulledReport.TempTexturesAtEnd)
	}
	for _, path := range []string{PathBloom, PathFog, PathUI} {
		if !executedPath(comparison.FullReport, path) {
			t.Fatalf("full graph did not execute %s path", path)
		}
	}
	expectedSkipped := map[PassID]bool{
		"bloom-preview": true,
		"fog-preview":   true,
		"ui-preview":    true,
	}
	for _, skipped := range comparison.CulledReport.Skipped {
		if !expectedSkipped[skipped.ID] {
			t.Fatalf("unexpected skipped pass %q", skipped.ID)
		}
	}
}

func executedPath(report ExecutionReport, path string) bool {
	for _, node := range report.Executed {
		if node.Path == path {
			return true
		}
	}
	return false
}

func containsResource(resources []ResourceID, target ResourceID) bool {
	for _, resource := range resources {
		if resource == target {
			return true
		}
	}
	return false
}

func containsNode(report ExecutionReport, id PassID) bool {
	return indexOfNode(report, id) >= 0
}

func indexOfNode(report ExecutionReport, id PassID) int {
	for i, node := range report.Executed {
		if node.ID == id {
			return i
		}
	}
	return -1
}

func TestSkippedReasonsAreHumanReadable(t *testing.T) {
	comparison, err := RenderComparison(64, 36)
	if err != nil {
		t.Fatal(err)
	}
	for _, skipped := range comparison.CulledReport.Skipped {
		if !strings.Contains(skipped.Reason, "not reachable") {
			t.Fatalf("unexpected skip reason for %s: %s", skipped.ID, skipped.Reason)
		}
	}
}
