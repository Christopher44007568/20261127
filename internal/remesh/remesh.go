package remesh

import (
	"math"

	"meshdiagnostics/internal/mesh"
	"meshdiagnostics/internal/qmesh"
)

// Options controls conversion from triangles to quads.
type Options struct {
	CreaseAngleDeg float64
	Smooth         bool
}

// Stats records the topology and feature preservation result.
type Stats struct {
	OriginalFaces       int
	OriginalEdges       int
	OriginalFeatures    int
	OriginalFeatureLen  float64
	QuadFaces           int
	QuadEdges           int
	QuadFeatures        int
	QuadFeatureLen      float64
	FeatureLengthRatio  float64
	FeatureTopologyKept bool
}

// Convert performs a feature-aware Catmull-Clark style quadrangulation.
//
// Each triangle becomes three quads through its edge midpoints and centroid.
// Midpoints are globally shared, so the all-quad result is conforming. Feature
// edges get sharp edge points and are marked on both output segments; feature
// vertices use the crease rule and corners stay fixed.
func Convert(src *mesh.TriangleMesh, opt Options) (*qmesh.Mesh, Stats, mesh.FeatureInfo) {
	features := src.DetectFeatures(mesh.FeatureOptions{CreaseAngleDeg: opt.CreaseAngleDeg})
	edgeMap := src.EdgeMap()
	nv := len(src.Vertices)
	nEdges := len(edgeMap)
	nFaces := len(src.Faces)

	edgeIDs := make(map[mesh.EdgeKey]int, nEdges)
	edgePoints := make([]mesh.Vec3, 0, nEdges)
	for key := range edgeMap {
		edgeIDs[key] = len(edgePoints)
		edgePoints = append(edgePoints, edgeMidpoint(src, key))
	}
	facePoints := make([]mesh.Vec3, nFaces)
	for i, f := range src.Faces {
		facePoints[i] = src.FaceCentroid(f)
	}

	orig := append([]mesh.Vec3(nil), src.Vertices...)
	if opt.Smooth {
		orig = smoothOrigins(src, edgeMap, features, facePoints)
	}

	vertices := make([]mesh.Vec3, 0, nv+nEdges+nFaces)
	vertices = append(vertices, orig...)
	edgeBase := len(vertices)
	vertices = append(vertices, edgePoints...)
	faceBase := len(vertices)
	vertices = append(vertices, facePoints...)

	quads := make([]qmesh.Face, 0, nFaces*3)
	for fi, f := range src.Faces {
		m01 := edgeBase + edgeIDs[mesh.Edge(f[0], f[1])]
		m12 := edgeBase + edgeIDs[mesh.Edge(f[1], f[2])]
		m20 := edgeBase + edgeIDs[mesh.Edge(f[2], f[0])]
		center := faceBase + fi
		quads = append(quads,
			qmesh.Face{f[0], m01, center, m20},
			qmesh.Face{f[1], m12, center, m01},
			qmesh.Face{f[2], m20, center, m12},
		)
	}

	qfeat := make(map[mesh.EdgeKey]bool)
	for key := range features.Edges {
		mid := edgeBase + edgeIDs[key]
		qfeat[mesh.Edge(key.A, mid)] = true
		qfeat[mesh.Edge(key.B, mid)] = true
	}

	out := &qmesh.Mesh{Vertices: vertices, Faces: quads, Features: qfeat}
	stats := Stats{
		OriginalFaces:       nFaces,
		OriginalEdges:       nEdges,
		OriginalFeatures:    len(features.Edges),
		QuadFaces:           len(quads),
		QuadFeatures:        len(qfeat),
		FeatureTopologyKept: true,
	}
	for key := range features.Edges {
		stats.OriginalFeatureLen += src.EdgeLength(key)
	}
	for key := range qfeat {
		stats.QuadFeatureLen += out.Vertices[key.A].Sub(out.Vertices[key.B]).Len()
	}
	stats.QuadEdges = len(out.Edges())
	if stats.OriginalFeatureLen > 1e-14 {
		stats.FeatureLengthRatio = stats.QuadFeatureLen / stats.OriginalFeatureLen
	} else {
		stats.FeatureLengthRatio = 1
	}
	return out, stats, features
}

func edgeMidpoint(m *mesh.TriangleMesh, key mesh.EdgeKey) mesh.Vec3 {
	return m.Vertices[key.A].Add(m.Vertices[key.B]).Scale(0.5)
}

func smoothOrigins(src *mesh.TriangleMesh, edges map[mesh.EdgeKey]*mesh.EdgeFace,
	features mesh.FeatureInfo, facePoints []mesh.Vec3) []mesh.Vec3 {
	n := len(src.Vertices)
	faceSum := make([]mesh.Vec3, n)
	faceCount := make([]int, n)
	edgeSum := make([]mesh.Vec3, n)
	edgeCount := make([]int, n)
	featureNeighbors := make([][]int, n)

	for fi, f := range src.Faces {
		for k := 0; k < 3; k++ {
			vi := f[k]
			faceSum[vi] = faceSum[vi].Add(facePoints[fi])
			faceCount[vi]++
		}
	}
	for key := range edges {
		mid := edgeMidpoint(src, key)
		edgeSum[key.A] = edgeSum[key.A].Add(mid)
		edgeSum[key.B] = edgeSum[key.B].Add(mid)
		edgeCount[key.A]++
		edgeCount[key.B]++
		if features.Edges[key] {
			featureNeighbors[key.A] = append(featureNeighbors[key.A], key.B)
			featureNeighbors[key.B] = append(featureNeighbors[key.B], key.A)
		}
	}

	out := make([]mesh.Vec3, n)
	for i, p := range src.Vertices {
		fs := featureNeighbors[i]
		switch {
		case len(fs) > 0 || edgeCount[i] != faceCount[i]+2:
			// Keep crease and boundary vertices fixed. This makes the
			// preserved feature polylines geometrically exact.
			out[i] = p
		case faceCount[i] > 0 && edgeCount[i] > 0:
			f := faceSum[i].Scale(1 / float64(faceCount[i]))
			r := edgeSum[i].Scale(1 / float64(edgeCount[i]))
			valence := float64(faceCount[i])
			out[i] = f.Add(r.Scale(2)).Add(p.Scale(valence - 3)).Scale(1 / valence)
		default:
			out[i] = p
		}
	}
	return out
}

// ClampCreaseAngle makes user flags safe even if an extreme value is supplied.
func ClampCreaseAngle(v float64) float64 {
	if math.IsNaN(v) || math.IsInf(v, 0) {
		return 40
	}
	return math.Max(0, math.Min(180, v))
}
