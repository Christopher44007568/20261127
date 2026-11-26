// Package cases constructs the three acceptance scenarios with ground truth:
// a closed hole loop, an open sharp crease and a tangent plane/cylinder join.
package cases

import (
	"fmt"

	"featurelines/internal/geom"
	"featurelines/internal/linalg"
	"featurelines/internal/mesh"
	"featurelines/internal/topo"
)

// Vec3 alias for convenience.
type Vec3 = linalg.Vec3

// CircleTruth is the ideal closed hole rim.
type CircleTruth struct {
	Center Vec3
	Radius float64
}

// SegmentTruth is the ideal open feature segment.
type SegmentTruth struct {
	A, B Vec3
}

// Case is one acceptance scenario.
type Case struct {
	ID       string
	Title    string
	Subtitle string
	Mesh     *mesh.Mesh

	// Kind is "hole", "sharp" or "tangent".
	Kind string

	Circle  *CircleTruth
	Segment *SegmentTruth
	Seam    *SegmentTruth

	// KappaObs is the deterministic noisy curvature observation.
	KappaObs []float64
	// Failed are hard-failed vertices for the stress run.
	Failed map[int]bool
	// FailVertex is the highlighted single-vertex failure.
	FailVertex int

	// NaiveTau is the "slightly high" curvature threshold for the baseline.
	NaiveTau float64

	// Reference feature vertices.
	Rim       []int
	Crease    []int
	SeamVerts []int
}

// LCG is a tiny deterministic pseudo-random source (no math/rand seeding
// surprises across Go versions).
type LCG struct{ state uint64 }

// NewLCG seeds the generator.
func NewLCG(seed uint64) *LCG { return &LCG{state: seed} }

// Float returns a pseudo-random float64 in [0,1).
func (g *LCG) Float() float64 {
	g.state = g.state*6364136223846793005 + 1442695040888963407
	return float64(g.state>>33) / float64(1<<31)
}

// Hole builds the closed-hole case. Curvature observations carry deterministic
// estimation noise and a few localized dips; one rim vertex is hard-failed.
func Hole() *Case {
	cfg := mesh.DefaultHolePlate()
	m, h := mesh.BuildHolePlate(cfg)
	tp := topo.Build(m)
	clean := geom.Curvature(m, tp)
	rng := NewLCG(1127)

	obs := make([]float64, len(clean))
	for i, k := range clean {
		obs[i] = k * (0.88 + 0.24*rng.Float()) // +-12% estimation noise
	}

	n := len(h.TopRim)
	dipFracs := []float64{0.16, 0.39, 0.60}
	for _, f := range dipFracs {
		v := h.TopRim[int(float64(n)*f)%n]
		obs[v] *= 0.38
	}
	failV := h.TopRim[int(float64(n)*0.83)]

	failed := map[int]bool{failV: true}
	obs[failV] = -1

	topZ := cfg.Thick * 0.5
	return &Case{
		ID: "hole", Title: "Closed circular hole",
		Subtitle: "rim loop survives a failed vertex",
		Mesh:     m, Kind: "hole",
		Circle:     &CircleTruth{Center: Vec3{X: 0, Y: 0, Z: topZ}, Radius: cfg.HoleR},
		KappaObs:   obs,
		Failed:     failed,
		FailVertex: failV,
		NaiveTau:   0.72,
		Rim:        h.TopRim,
	}
}

// Sharp builds the open sharp-edge case with one failed mid-crease vertex.
func Sharp() *Case {
	cfg := mesh.DefaultFold()
	m, h := mesh.BuildFold(cfg)
	tp := topo.Build(m)
	clean := geom.Curvature(m, tp)
	rng := NewLCG(2207)
	obs := make([]float64, len(clean))
	for i, k := range clean {
		obs[i] = k * (0.92 + 0.16*rng.Float())
	}
	failV := h.Crease[len(h.Crease)/2]
	obs[failV] = -1

	return &Case{
		ID: "sharp", Title: "Open sharp edge",
		Subtitle: "one crease, two true endpoints",
		Mesh:     m, Kind: "sharp",
		Segment: &SegmentTruth{
			A: Vec3{X: 0, Y: -cfg.Length * 0.5, Z: 0},
			B: Vec3{X: 0, Y: cfg.Length * 0.5, Z: 0},
		},
		KappaObs:   obs,
		Failed:     map[int]bool{failV: true},
		FailVertex: failV,
		NaiveTau:   0.8,
		Crease:     h.Crease,
	}
}

// Tangent builds the tangent plane/cylinder case (negative acceptance).
func Tangent() *Case {
	cfg := mesh.DefaultTangent()
	m, h := mesh.BuildTangent(cfg)
	tp := topo.Build(m)
	clean := geom.Curvature(m, tp)
	rng := NewLCG(3307)
	obs := make([]float64, len(clean))
	for i, k := range clean {
		obs[i] = k * (0.95 + 0.10*rng.Float())
	}

	seamLen := cfg.Width
	return &Case{
		ID: "tangent", Title: "Tangent surfaces",
		Subtitle: "curvature jump, agreeing normals: no crease",
		Mesh:     m, Kind: "tangent",
		Seam: &SegmentTruth{
			A: Vec3{X: 0, Y: -seamLen * 0.5, Z: 0},
			B: Vec3{X: 0, Y: seamLen * 0.5, Z: 0},
		},
		KappaObs:  obs,
		Failed:    map[int]bool{},
		NaiveTau:  0.8,
		SeamVerts: h.Seam,
	}
}

// All returns the three acceptance cases in display order.
func All() []*Case {
	return []*Case{Hole(), Sharp(), Tangent()}
}

// Describe returns a short human-readable summary.
func (c *Case) Describe() string {
	switch c.Kind {
	case "hole":
		return fmt.Sprintf("%s: circle r=%.1f", c.ID, c.Circle.Radius)
	case "sharp":
		return fmt.Sprintf("%s: segment len=%.1f", c.ID, c.Segment.A.Dist(c.Segment.B))
	default:
		return fmt.Sprintf("%s: seam len=%.1f", c.ID, c.Seam.A.Dist(c.Seam.B))
	}
}
