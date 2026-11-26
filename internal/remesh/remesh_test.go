package remesh

import (
	"math"
	"testing"

	"meshdiagnostics/internal/diag"
	"meshdiagnostics/internal/mesh"
	"meshdiagnostics/internal/sample"
)

func TestConvertProducesConformingQuads(t *testing.T) {
	src := sample.Organic()
	q, stats, _ := Convert(src, Options{CreaseAngleDeg: 40, Smooth: true})
	if err := q.Validate(); err != nil {
		t.Fatal(err)
	}
	if len(q.Faces) != len(src.Faces)*3 {
		t.Fatalf("got %d quads, want %d", len(q.Faces), len(src.Faces)*3)
	}
	if len(q.Faces) == 0 || stats.QuadFaces != len(q.Faces) {
		t.Fatal("bad quad face count")
	}
	edges := q.Edges()
	for _, f := range q.Faces {
		for i := 0; i < 4; i++ {
			key := mesh.Edge(f[i], f[(i+1)%4])
			if count := edges[key]; count != 2 {
				t.Fatalf("non-conforming edge %v has %d faces", key, count)
			}
		}
	}
}

func TestOrganicHasCollapseSignal(t *testing.T) {
	src := sample.Organic()
	q, _, _ := Convert(src, Options{CreaseAngleDeg: 40, Smooth: true})
	a := diag.Analyze(q, 18)
	t.Logf("collapsed %d/%d, worst %.2f, p5 %.2f", len(a.Collapsed), a.FaceCount,
		a.WorstMetrics.MinAngleDeg, a.AngleStats.P5)
	var mins []float64
	for _, f := range src.Faces {
		mins = append(mins, triangleMinAngle(src, f))
	}
	ts := diag.Summarize(mins)
	t.Logf("tri min angle: min %.2f p5 %.2f median %.2f p95 %.2f", ts.Min, ts.P5, ts.Median, ts.P95)
	if len(a.Collapsed) == 0 {
		t.Fatal("sample should expose at least one collapsed quad")
	}
}

func triangleMinAngle(m *mesh.TriangleMesh, f mesh.Tri) float64 {
	minA := 180.0
	for i := 0; i < 3; i++ {
		a, b, c := m.Vertices[f[i]], m.Vertices[f[(i+1)%3]], m.Vertices[f[(i+2)%3]]
		e1, e2 := b.Sub(a), c.Sub(a)
		d := math.Max(-1, math.Min(1, e1.Normalize().Dot(e2.Normalize())))
		minA = math.Min(minA, math.Acos(d)*180/math.Pi)
	}
	return minA
}
