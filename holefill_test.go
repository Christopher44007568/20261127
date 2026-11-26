package holefill

import (
	"bytes"
	"math"
	"testing"
)

func classifiedDemo(t *testing.T) (*Mesh, []*BoundaryLoop, SceneLayout) {
	t.Helper()
	m, layout := BuildDemoMesh()
	loops, err := ExtractBoundaryLoops(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(loops) != 7 {
		t.Fatalf("boundary loops = %d, want 7 (outer + six defects)", len(loops))
	}
	ClassifyLoops(m, loops)
	return m, loops, layout
}

func TestBoundaryClassification(t *testing.T) {
	_, loops, _ := classifiedDemo(t)
	counts := map[HoleClass]int{}
	pairIDs := map[int]int{}
	for _, l := range loops {
		counts[l.Class]++
		if l.Class == ClassClosePair {
			pairIDs[l.Metrics.PairID]++
		}
	}
	if counts[ClassOuter] != 1 {
		t.Fatalf("outer loops = %d, want 1", counts[ClassOuter])
	}
	if counts[ClassSlit] != 1 {
		t.Fatalf("slits = %d, want 1", counts[ClassSlit])
	}
	if counts[ClassLNotch] != 1 {
		t.Fatalf("L notches = %d, want 1", counts[ClassLNotch])
	}
	if counts[ClassClosePair] != 3 {
		t.Fatalf("close holes = %d, want thin pair plus outer-near hole", counts[ClassClosePair])
	}
	if counts[ClassOrdinaryHole] != 1 {
		t.Fatalf("ordinary holes = %d, want 1", counts[ClassOrdinaryHole])
	}
	hasThinPair := false
	for id, n := range pairIDs {
		if id > 0 && n == 2 {
			hasThinPair = true
		}
	}
	if !hasThinPair {
		t.Fatalf("pair IDs=%v, want one ID shared by two loops", pairIDs)
	}
}

func TestPatchClosesInteriorLoopsAndPreservesOuter(t *testing.T) {
	m, loops, _ := classifiedDemo(t)
	before := m.ComputeNormals()
	oldVertexCount := len(m.V)
	outerEdges := 0
	for _, l := range loops {
		if l.Class == ClassOuter {
			outerEdges = len(l.Verts)
		}
	}

	reports, err := PatchAllLoops(m, loops, DefaultPatchOptions())
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 6 {
		t.Fatalf("patch reports = %d, want 6", len(reports))
	}

	remaining, err := ExtractBoundaryLoops(m)
	if err != nil {
		t.Fatal(err)
	}
	if len(remaining) != 1 || remaining[0].Class != ClassUnknown {
		t.Fatalf("remaining loops = %+v, want only the unclassified outer contour", remaining)
	}
	if got := len(m.BoundaryEdges()); got != outerEdges {
		t.Fatalf("boundary edges after patch = %d, want outer count %d", got, outerEdges)
	}

	uses := m.EdgeUseCounts()
	for _, l := range loops {
		if l.Class == ClassOuter {
			continue
		}
		for i, a := range l.Verts {
			b := l.Verts[(i+1)%len(l.Verts)]
			if uses[undirectedEdge(a, b)] != 2 {
				t.Fatalf("loop %d boundary edge %d-%d uses %d, want 2",
					l.ID, a, b, uses[undirectedEdge(a, b)])
			}
		}
		for _, v := range l.Verts {
			if angle := normalAngle(before[v], m.Norm[v]); angle > 1e-6 {
				t.Fatalf("boundary normal changed by %.6g rad at vertex %d", angle, v)
			}
		}
	}
	for i := oldVertexCount; i < len(m.V); i++ {
		if m.Norm[i].Z < 0.45 {
			t.Fatalf("new patch normal %+v at %d is not oriented with the scan surface", m.Norm[i], i)
		}
	}
}

func TestThickenedShellAndSection(t *testing.T) {
	m, loops, layout := classifiedDemo(t)
	if _, err := PatchAllLoops(m, loops, DefaultPatchOptions()); err != nil {
		t.Fatal(err)
	}
	shell := Thicken(m, layout.Thickness)
	if !shell.IsClosedManifold() {
		t.Fatal("thickened shell is not a closed edge-manifold")
	}

	segs := BuildSection(shell, layout.SectionY)
	found := map[HoleClass]bool{}
	bottom := 0
	for _, s := range segs {
		if s.Tag.Kind == KindPatch {
			found[s.Tag.Class] = true
		}
		if s.Tag.Kind == KindBottom {
			bottom++
		}
	}
	for _, class := range []HoleClass{ClassSlit, ClassLNotch, ClassClosePair} {
		if !found[class] {
			t.Fatalf("section has no %s patch segment", class)
		}
	}
	if bottom < 100 {
		t.Fatalf("bottom section segments = %d, want a continuous underside", bottom)
	}
}

func TestNoAlphaWrapAlsoCloses(t *testing.T) {
	m, loops, _ := classifiedDemo(t)
	opts := DefaultPatchOptions()
	opts.AlphaWrap = false
	reports, err := PatchAllLoops(m, loops, opts)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range reports {
		if r.Wrapped {
			t.Fatalf("report %+v unexpectedly wrapped", r)
		}
	}
	outerEdges := 0
	for _, l := range loops {
		if l.Class == ClassOuter {
			outerEdges = len(l.Verts)
		}
	}
	if len(m.BoundaryEdges()) != outerEdges {
		t.Fatalf("boundary edges = %d, want only outer loop", len(m.BoundaryEdges()))
	}
}

func TestPPMHeader(t *testing.T) {
	cv := NewCanvas(16, 8, RGB{1, 2, 3})
	var buf bytes.Buffer
	if err := cv.WritePPM(&buf); err != nil {
		t.Fatal(err)
	}
	data := buf.Bytes()
	header := []byte("P6\n16 8\n255\n")
	if !bytes.HasPrefix(data, header) {
		t.Fatalf("missing P6 16x8 header: %q", data[:minInt(20, len(data))])
	}
	if len(data) != len(header)+16*8*3 {
		t.Fatalf("PPM length = %d, want header %d plus %d pixels", len(data), len(header), 16*8*3)
	}
}

func normalAngle(a, b Vec3) float64 {
	d := math.Max(-1, math.Min(1, a.Normalize().Dot(b.Normalize())))
	return math.Acos(d)
}
