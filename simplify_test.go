package qem

import (
	"math"
	"testing"
)

func TestOrbCollapsePreservesLocksVolumeAndUndo(t *testing.T) {
	mesh := NewDetailOrb(3)
	original := mesh.Clone()
	opt := DefaultOptions()
	s, err := NewSimplifier(mesh, opt)
	if err != nil {
		t.Fatal(err)
	}
	initialFaces := mesh.ActiveTriangleCount()
	target := initialFaces / 3
	steps, collapseErr := s.CollapseToTriangles(target)
	if collapseErr != nil && collapseErr != ErrNoCollapsibleEdge {
		t.Fatalf("unexpected simplification error: %v", collapseErr)
	}
	if steps < 20 {
		t.Fatalf("expected many collapses, got %d", steps)
	}
	if collapseErr == nil && mesh.ActiveTriangleCount() > target+4 {
		t.Fatalf("did not approach target: active=%d target=%d", mesh.ActiveTriangleCount(), target)
	}
	t.Logf("orb collapses=%d active=%d target=%d err=%v", steps, mesh.ActiveTriangleCount(), target, collapseErr)

	for _, rec := range s.Records {
		if reason := s.protected[rec.Edge]; reason != "" {
			t.Fatalf("protected edge %+v collapsed: %s", rec.Edge, reason)
		}
		if rec.VolumeAfter < s.initialVolume-s.allowedLoss-1e-9 {
			t.Fatalf("volume floor violated: %.8f < %.8f", rec.VolumeAfter, s.initialVolume-s.allowedLoss)
		}
	}

	report := s.Verify(original)
	if len(report.DegenerateFaces) > 0 {
		t.Fatalf("degenerate faces: %v", report.DegenerateFaces[:min(5, len(report.DegenerateFaces))])
	}
	if len(report.MissingProtectedEdges) > 0 {
		t.Fatalf("missing protected edges: %v", report.MissingProtectedEdges[:min(5, len(report.MissingProtectedEdges))])
	}
	if len(report.InactiveProtectedVertices) > 0 {
		t.Fatalf("inactive protected vertices: %v", report.InactiveProtectedVertices[:5])
	}
	if len(report.FrozenPositionViolations) > 0 {
		t.Fatalf("frozen positions changed: %v", report.FrozenPositionViolations[:5])
	}
	if len(report.DuplicateTopologyKeys) > 0 {
		t.Fatalf("duplicate topology: %v", report.DuplicateTopologyKeys[:3])
	}
	if len(report.FlippedFaces) > 0 {
		t.Fatalf("faces reversed relative original: %v", report.FlippedFaces[:min(5, len(report.FlippedFaces))])
	}

	volumeRatio := math.Abs(mesh.SignedVolume()) / math.Abs(original.SignedVolume())
	if volumeRatio < 0.99 {
		t.Fatalf("volume shrank too much: ratio=%.5f", volumeRatio)
	}

	if err := s.UndoAll(); err != nil {
		t.Fatal(err)
	}
	assertMeshesEqual(t, original, mesh)
}

func TestBoundaryPatchLocksBoundaryEdges(t *testing.T) {
	mesh := NewWavyPatch(17, 17)
	original := mesh.Clone()
	s, err := NewSimplifier(mesh, DefaultOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(s.protected) == 0 {
		t.Fatal("expected boundary and material protected edges")
	}
	_, err = s.CollapseToTriangles(mesh.ActiveTriangleCount() / 4)
	if err != nil && err != ErrNoCollapsibleEdge {
		t.Fatal(err)
	}
	if len(s.Records) < 10 {
		t.Fatalf("expected interior collapses, got %d", len(s.Records))
	}
	for _, rec := range s.Records {
		if reason := s.protected[rec.Edge]; reason != "" {
			t.Fatalf("%s edge collapsed: %+v", reason, rec.Edge)
		}
	}
	report := s.Verify(original)
	if len(report.DegenerateFaces) > 0 || len(report.MissingProtectedEdges) > 0 ||
		len(report.FrozenPositionViolations) > 0 || len(report.DuplicateTopologyKeys) > 0 {
		t.Fatalf("bad verification: %+v", report)
	}
	if err := s.UndoAll(); err != nil {
		t.Fatal(err)
	}
	assertMeshesEqual(t, original, mesh)
}

func assertMeshesEqual(t *testing.T, want, got *Mesh) {
	t.Helper()
	if len(want.Vertices) != len(got.Vertices) || len(want.Triangles) != len(got.Triangles) {
		t.Fatalf("size mismatch: want v%d/t%d got v%d/t%d", len(want.Vertices), len(want.Triangles), len(got.Vertices), len(got.Triangles))
	}
	for i := range want.Vertices {
		if want.Vertices[i].Active != got.Vertices[i].Active || want.Vertices[i].Pos != got.Vertices[i].Pos {
			t.Fatalf("vertex %d not restored: want %+v got %+v", i, want.Vertices[i], got.Vertices[i])
		}
	}
	for i := range want.Triangles {
		if want.Triangles[i] != got.Triangles[i] {
			t.Fatalf("triangle %d not restored: want %+v got %+v", i, want.Triangles[i], got.Triangles[i])
		}
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
