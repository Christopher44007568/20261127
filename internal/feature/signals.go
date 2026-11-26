// Package feature implements curvature-difference + normal-angle feature
// tracing with hysteresis and one-vertex gap bridging.
package feature

import (
	"math"

	"featurelines/internal/geom"
	"featurelines/internal/mesh"
	"featurelines/internal/topo"
)

// Signal holds the per-edge evidence used by the tracer.
type Signal struct {
	Edge int
	// Theta is the dihedral angle between adjacent face normals (radians).
	// Boundary edges get Theta = Pi.
	Theta float64
	// CDiff is the normalized curvature jump across the edge, measured
	// between the two face-opposite vertices.
	CDiff float64
	// KMax is the largest normalized endpoint curvature.
	KMax float64
	// Boundary marks a topological boundary/contour edge.
	Boundary bool
	// HasFail marks an edge touching an observation-failed vertex.
	HasFail bool
}

// Signals bundles the topology with one signal record per edge.
type Signals struct {
	Topo   *topo.Topology
	KappaN []float64 // normalized observed curvature per vertex (-1 = invalid)
	KRef   float64
	List   []Signal
}

// ComputeSignals estimates curvature, normalizes it and evaluates every edge.
// kappaObs may contain negative values to mark invalid/missing samples;
// failed vertices are additionally forced invalid.
func ComputeSignals(m *mesh.Mesh, tp *topo.Topology, kappaObs []float64, failed map[int]bool) *Signals {
	clean := geom.Curvature(m, tp)
	if kappaObs == nil {
		kappaObs = clean
	}
	for v := range failed {
		if v >= 0 && v < len(kappaObs) {
			kappaObs[v] = -1
		}
	}

	kRef := geom.FeatureScale(kappaObs, tp)
	kn := make([]float64, len(m.V))
	for i, k := range kappaObs {
		if k < 0 || failed[i] {
			kn[i] = -1
		} else {
			kn[i] = k / kRef
		}
	}

	sig := &Signals{Topo: tp, KappaN: kn, KRef: kRef, List: make([]Signal, len(tp.Edges))}

	// Opposite vertex lookup per edge face.
	opposite := func(e *topo.Edge, faceIdx int) int {
		t := m.T[faceIdx]
		for _, v := range t {
			if v != e.A && v != e.B {
				return v
			}
		}
		return -1
	}

	for ei, e := range tp.Edges {
		s := Signal{Edge: ei}
		s.HasFail = failed[e.A] || failed[e.B] || kn[e.A] < 0 || kn[e.B] < 0
		s.KMax = math.Max(valid0(kn[e.A]), valid0(kn[e.B]))

		if len(e.Faces) == 0 {
			continue
		}
		if len(e.Faces) == 1 {
			s.Boundary = true
			s.Theta = math.Pi
		} else {
			f1, f2 := e.Faces[0], e.Faces[1]
			s.Theta = angleBetween(m.FN[f1], m.FN[f2])
			o1 := opposite(e, f1)
			o2 := opposite(e, f2)
			dOpp := 0.0
			if o1 >= 0 && o2 >= 0 && kn[o1] >= 0 && kn[o2] >= 0 {
				dOpp = math.Abs(kn[o1] - kn[o2])
			}
			dEnd := 0.0
			if kn[e.A] >= 0 && kn[e.B] >= 0 {
				dEnd = 0.5 * math.Abs(kn[e.A]-kn[e.B])
			}
			s.CDiff = math.Max(dOpp, dEnd)
		}
		sig.List[ei] = s
	}
	return sig
}

func valid0(v float64) float64 {
	if v < 0 {
		return 0
	}
	return v
}

func angleBetween(a, b mesh.Vec3) float64 {
	d := a.Dot(b)
	if d > 1 {
		d = 1
	}
	if d < -1 {
		d = -1
	}
	return math.Acos(d)
}

// MedianEdgeLength returns the median geometric edge length of the mesh.
func MedianEdgeLength(tp *topo.Topology, m *mesh.Mesh) float64 {
	lens := make([]float64, len(tp.Edges))
	for i, e := range tp.Edges {
		lens[i] = m.V[e.A].Dist(m.V[e.B])
	}
	sortFloats(lens)
	return lens[len(lens)/2]
}

func sortFloats(a []float64) {
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j-1] > a[j]; j-- {
			a[j-1], a[j] = a[j], a[j-1]
		}
	}
}
