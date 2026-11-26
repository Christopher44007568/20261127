package diag

import (
	"math"
	"sort"

	"meshdiagnostics/internal/mesh"
	"meshdiagnostics/internal/qmesh"
)

// Analysis contains geometric diagnostics for a quad mesh.
type Analysis struct {
	FaceCount      int
	FaceMetrics    []qmesh.Metrics
	Collapsed      []int
	WorstFace      int
	WorstMetrics   qmesh.Metrics
	Curvature      []float64
	FaceCurvature  []float64
	CurvBands      []int
	CurvThresholds []float64
	CurvStats      Summary
	EdgeLengths    []float64
	CornerAngles   []float64
	EdgeHist       Bins
	EdgeStats      Summary
	AngleHist      Bins
	AngleStats     Summary
	Scale          float64
}

// Analyze measures face angles, curvature bands and edge lengths. All values
// are in the input mesh's units except curvature, which is scale-normalized.
func Analyze(q *qmesh.Mesh, collapseAngleDeg float64) *Analysis {
	a := &Analysis{WorstFace: -1, FaceCount: len(q.Faces)}
	lo, hi := mesh.Bounds(q.Vertices)
	a.Scale = hi.Sub(lo).Len()
	if a.Scale < 1e-12 {
		a.Scale = 1
	}
	a.FaceMetrics = make([]qmesh.Metrics, len(q.Faces))
	for i, f := range q.Faces {
		m := q.Metrics(f)
		a.FaceMetrics[i] = m
		for k := 0; k < 4; k++ {
			a.CornerAngles = append(a.CornerAngles, m.AnglesDeg[k])
		}
		if m.MinAngleDeg < collapseAngleDeg || !m.Convex || m.MaxAngleDeg > 180-collapseAngleDeg {
			a.Collapsed = append(a.Collapsed, i)
		}
		if a.WorstFace < 0 || worseFace(m, a.WorstMetrics) {
			a.WorstFace = i
			a.WorstMetrics = m
		}
	}

	a.Curvature, a.FaceCurvature = curvatureMagnitude(q, a.Scale)
	a.CurvStats = Summarize(a.FaceCurvature)
	a.CurvThresholds = curvatureThresholds(a.FaceCurvature)
	a.CurvBands = make([]int, len(a.FaceCurvature))
	for i, c := range a.FaceCurvature {
		b := sort.Search(len(a.CurvThresholds), func(k int) bool { return c <= a.CurvThresholds[k] })
		if b >= 5 {
			b = 4
		}
		a.CurvBands[i] = b
	}

	edges := q.Edges()
	for key := range edges {
		a.EdgeLengths = append(a.EdgeLengths, q.Vertices[key.A].Sub(q.Vertices[key.B]).Len())
	}
	sort.Float64s(a.EdgeLengths)
	a.EdgeStats = Summarize(a.EdgeLengths)
	a.EdgeHist = Histogram(a.EdgeLengths, 28)
	a.AngleStats = Summarize(a.CornerAngles)
	a.AngleHist = Histogram(a.CornerAngles, 18)
	return a
}

func worseFace(candidate, current qmesh.Metrics) bool {
	if !candidate.Convex && current.Convex {
		return true
	}
	if candidate.Convex != current.Convex {
		return false
	}
	return candidate.MinAngleDeg < current.MinAngleDeg
}

// curvatureMagnitude estimates bending per unit length from dihedral angles.
// It is normalized by the mesh bounding-box diagonal so bands remain stable
// when identical shapes are exported at different scales.
func curvatureMagnitude(q *qmesh.Mesh, scale float64) ([]float64, []float64) {
	normals := make([]mesh.Vec3, len(q.Faces))
	for i, f := range q.Faces {
		normals[i] = q.FaceNormal(f).Normalize()
	}
	edgeFaces := make(map[mesh.EdgeKey][]int, len(q.Faces)*2)
	for fi, f := range q.Faces {
		for k := 0; k < 4; k++ {
			key := mesh.Edge(f[k], f[(k+1)%4])
			edgeFaces[key] = append(edgeFaces[key], fi)
		}
	}
	vertexBend := make([]float64, len(q.Vertices))
	vertexWeight := make([]float64, len(q.Vertices))
	for key, faces := range edgeFaces {
		if len(faces) != 2 {
			continue
		}
		n1, n2 := normals[faces[0]], normals[faces[1]]
		dot := math.Max(-1, math.Min(1, n1.Dot(n2)))
		angle := math.Acos(dot)
		if angle < 1e-12 {
			continue
		}
		l := q.Vertices[key.A].Sub(q.Vertices[key.B]).Len()
		if l < 1e-14 {
			continue
		}
		bend := angle * scale / l
		vertexBend[key.A] += bend
		vertexBend[key.B] += bend
		vertexWeight[key.A]++
		vertexWeight[key.B]++
	}
	for i := range vertexBend {
		if vertexWeight[i] > 0 {
			vertexBend[i] /= vertexWeight[i]
		}
	}
	neighbors := make([][]int, len(q.Vertices))
	for key := range edgeFaces {
		neighbors[key.A] = append(neighbors[key.A], key.B)
		neighbors[key.B] = append(neighbors[key.B], key.A)
	}
	// The raw dihedral estimator follows every tiny facet wobble. A few
	// local averaging passes expose the broad curvature bands that matter
	// visually while keeping sharp ridges visibly hot.
	for iter := 0; iter < 12; iter++ {
		next := append([]float64(nil), vertexBend...)
		for i, ns := range neighbors {
			if len(ns) == 0 {
				continue
			}
			var avg float64
			for _, j := range ns {
				avg += vertexBend[j]
			}
			avg /= float64(len(ns))
			next[i] = 0.45*vertexBend[i] + 0.55*avg
		}
		vertexBend = next
	}
	faceCurv := make([]float64, len(q.Faces))
	for i, f := range q.Faces {
		faceCurv[i] = (vertexBend[f[0]] + vertexBend[f[1]] + vertexBend[f[2]] + vertexBend[f[3]]) * 0.25
	}
	return vertexBend, faceCurv
}

func curvatureThresholds(values []float64) []float64 {
	return Quantiles(values, []float64{0.2, 0.4, 0.6, 0.8})
}

// FeatureLengthRatio is a simple preservation metric independent of face count.
func FeatureLengthRatio(original, quad float64) float64 {
	if original < 1e-14 {
		return 1
	}
	return quad / original
}
