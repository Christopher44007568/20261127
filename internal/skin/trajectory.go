package skin

import "math"

// TrajectorySample is one observed vertex position during a bend/twist sweep.
type TrajectorySample struct {
	AngleDeg float64
	Method   string
	Vertex   int
	Ring     int
	Segment  int
	Position Vec3
	Weights  [defaultBoneCount]float64
}

// BuildTrajectories samples selected vertices while bend and twist are swept
// together. At angle zero both algorithms return the exact bind pose.
func BuildTrajectories(m Mesh, markers []int, maxBend, maxTwist float64, steps int) []TrajectorySample {
	if steps < 2 {
		steps = 2
	}
	samples := make([]TrajectorySample, 0, steps*len(markers)*2)
	for step := 0; step < steps; step++ {
		t := float64(step) / float64(steps-1)
		bend := maxBend * t
		twist := maxTwist * t
		pose := ElbowPose("sweep", bend, twist)
		lbs := DeformLBS(m, pose)
		dq := DeformDQ(m, pose)
		for _, index := range markers {
			vertex := m.Vertices[index]
			samples = append(samples,
				TrajectorySample{
					AngleDeg: bend * 180 / math.Pi,
					Method:   "LBS",
					Vertex:   index,
					Ring:     vertex.Ring,
					Segment:  vertex.Segment,
					Position: lbs[index],
					Weights:  vertex.Weights,
				},
				TrajectorySample{
					AngleDeg: bend * 180 / math.Pi,
					Method:   "DQ",
					Vertex:   index,
					Ring:     vertex.Ring,
					Segment:  vertex.Segment,
					Position: dq[index],
					Weights:  vertex.Weights,
				})
		}
	}
	return samples
}

// MarkerVertexIndices chooses three vertices around the blend zone: one on
// each side of the bend plane and one on the lateral surface.
func MarkerVertexIndices(m Mesh) []int {
	ring := 0
	bestDistance := math.Inf(1)
	for candidate, x := range m.RingX {
		distance := math.Abs(x - 0.08)
		if distance < bestDistance {
			bestDistance = distance
			ring = candidate
		}
	}
	sides := []int{0, m.Segments / 4, m.Segments / 2}
	markers := make([]int, len(sides))
	for index, segment := range sides {
		markers[index] = m.VertexIndex(ring, segment)
	}
	return markers
}
