package skin

import "math"

const defaultBoneCount = 2

// Vertex stores one bind-pose point and the normalized weights shared by every
// deformation algorithm. Index 0 drives the upper arm, index 1 the forearm.
type Vertex struct {
	Bind    Vec3
	Weights [defaultBoneCount]float64
	Ring    int
	Segment int
}

// Quad is one quad in the tube surface. A, B, C and D are counter-clockwise
// when viewed from outside the tube.
type Quad struct {
	A, B, C, D int
	Ring       int
	Segment    int
}

// Mesh is an open-ended tube aligned with +X in bind pose. Rings are ordered
// from the upper arm toward the hand.
type Mesh struct {
	Vertices     []Vertex
	Quads        []Quad
	RingX        []float64
	RingRadius   []float64
	RingVertices int
	RingCount    int
	Segments     int
}

// BuildTube creates a two-bone arm mesh. Weights vary slightly around the
// circumference near the elbow, as they do on a real joint. Both skinning
// algorithms consume these same values.
func BuildTube() Mesh {
	const (
		rings    = 57
		segments = 48
		firstX   = -1.55
		lastX    = 1.85
	)

	mesh := Mesh{
		RingX:        make([]float64, rings),
		RingRadius:   make([]float64, rings),
		RingVertices: segments,
		RingCount:    rings,
		Segments:     segments,
		Vertices:     make([]Vertex, 0, rings*segments),
		Quads:        make([]Quad, 0, (rings-1)*segments),
	}

	for ring := 0; ring < rings; ring++ {
		t := float64(ring) / float64(rings-1)
		x := firstX + (lastX-firstX)*t
		radius := 0.34
		if x > 0 {
			radius = 0.34 - 0.045*(x/lastX)
		}
		mesh.RingX[ring] = x
		mesh.RingRadius[ring] = radius

		for segment := 0; segment < segments; segment++ {
			angle := 2 * math.Pi * float64(segment) / float64(segments)
			sine, cosine := math.Sincos(angle)
			point := Vec3{X: x, Y: radius * cosine, Z: radius * sine}
			mesh.Vertices = append(mesh.Vertices, Vertex{
				Bind:    point,
				Weights: [defaultBoneCount]float64{1 - forearmWeight(x, point.Y), forearmWeight(x, point.Y)},
				Ring:    ring,
				Segment: segment,
			})
		}
	}

	for ring := 0; ring < rings-1; ring++ {
		for segment := 0; segment < segments; segment++ {
			next := (segment + 1) % segments
			a := ring*segments + segment
			b := ring*segments + next
			c := (ring+1)*segments + next
			d := (ring+1)*segments + segment
			mesh.Quads = append(mesh.Quads, Quad{A: a, B: b, C: c, D: d, Ring: ring, Segment: segment})
		}
	}

	return mesh
}

func forearmWeight(x, y float64) float64 {
	const (
		blendHalfWidth = 0.38
		sideBias       = 0.55
	)
	t := 0.5 + (x+sideBias*y)/(2*blendHalfWidth)
	return smoothstep(t)
}

func smoothstep(t float64) float64 {
	if t <= 0 {
		return 0
	}
	if t >= 1 {
		return 1
	}
	return t * t * (3 - 2*t)
}

// VertexIndex returns the index of a ring/segment pair.
func (m Mesh) VertexIndex(ring, segment int) int {
	return ring*m.RingVertices + segment%m.RingVertices
}

// RingIndices returns the vertex indices of one circular cross-section.
func (m Mesh) RingIndices(ring int) []int {
	indices := make([]int, m.RingVertices)
	for segment := range indices {
		indices[segment] = m.VertexIndex(ring, segment)
	}
	return indices
}

// Pose is the set of bone transforms. The bind transforms are intentionally
// identity: all authored vertices are already in skeleton space.
type Pose struct {
	Name       string
	Bend       float64
	Twist      float64
	Transforms []Transform
}

// ElbowPose builds a pose around the elbow at the origin. Bend rotates around
// Z; twist rotates the forearm around its local X axis. The final rotation is
// Rz(bend) * Rx(twist).
func ElbowPose(name string, bend, twist float64) Pose {
	bendRotation := QuatFromAxisAngle(Vec3{Z: 1}, bend)
	twistRotation := QuatFromAxisAngle(Vec3{X: 1}, twist)
	return Pose{
		Name:  name,
		Bend:  bend,
		Twist: twist,
		Transforms: []Transform{
			IdentityTransform(),
			{Rotation: bendRotation.Mul(twistRotation)},
		},
	}
}

func (p Pose) IsIdentity() bool {
	for _, transform := range p.Transforms {
		if !transform.IsIdentity() {
			return false
		}
	}
	return true
}
