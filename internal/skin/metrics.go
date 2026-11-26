package skin

import "math"

// AreaSample records the true 3D area of one deformed cross-section ring,
// normalized by that ring's bind-pose area.
type AreaSample struct {
	Ring      int
	X         float64
	BindArea  float64
	LBSArea   float64
	DQArea    float64
	LBSRatio  float64
	DQRatio   float64
	MeanUpper float64
	MeanFore  float64
}

// QuadSample records areal deformation for one surface quad. Values below one
// are compression; values above one are local bulging.
type QuadSample struct {
	Ring     int
	Segment  int
	LBSRatio float64
	DQRatio  float64
}

// Comparison contains all derived measurements for one pose.
type Comparison struct {
	Pose        Pose
	LBS         []Vec3
	DQ          []Vec3
	Areas       []AreaSample
	Quads       []QuadSample
	MaxLBSRatio float64
	MaxDQRatio  float64
	MinLBSRatio float64
	MinDQRatio  float64
}

func Analyze(m Mesh, pose Pose) Comparison {
	comparison := Comparison{
		Pose:        pose,
		LBS:         DeformLBS(m, pose),
		DQ:          DeformDQ(m, pose),
		Areas:       make([]AreaSample, 0, m.RingCount),
		Quads:       make([]QuadSample, 0, len(m.Quads)),
		MaxLBSRatio: 0,
		MaxDQRatio:  0,
		MinLBSRatio: math.Inf(1),
		MinDQRatio:  math.Inf(1),
	}

	bind := bindPoints(m)
	for ring := 0; ring < m.RingCount; ring++ {
		indices := m.RingIndices(ring)
		area := AreaSample{
			Ring:     ring,
			X:        m.RingX[ring],
			BindArea: polygonArea(bind, indices),
			LBSArea:  polygonArea(comparison.LBS, indices),
			DQArea:   polygonArea(comparison.DQ, indices),
		}
		if area.BindArea > 0 {
			area.LBSRatio = area.LBSArea / area.BindArea
			area.DQRatio = area.DQArea / area.BindArea
		}
		for _, index := range indices {
			area.MeanUpper += m.Vertices[index].Weights[0]
			area.MeanFore += m.Vertices[index].Weights[1]
		}
		area.MeanUpper /= float64(len(indices))
		area.MeanFore /= float64(len(indices))
		comparison.Areas = append(comparison.Areas, area)
		comparison.MinLBSRatio = math.Min(comparison.MinLBSRatio, area.LBSRatio)
		comparison.MinDQRatio = math.Min(comparison.MinDQRatio, area.DQRatio)
		comparison.MaxLBSRatio = math.Max(comparison.MaxLBSRatio, area.LBSRatio)
		comparison.MaxDQRatio = math.Max(comparison.MaxDQRatio, area.DQRatio)
	}

	for _, quad := range m.Quads {
		bindArea := quadArea(bind, quad)
		sample := QuadSample{Ring: quad.Ring, Segment: quad.Segment}
		if bindArea > 0 {
			sample.LBSRatio = quadArea(comparison.LBS, quad) / bindArea
			sample.DQRatio = quadArea(comparison.DQ, quad) / bindArea
		}
		comparison.Quads = append(comparison.Quads, sample)
	}
	return comparison
}

func polygonArea(points []Vec3, indices []int) float64 {
	if len(indices) < 3 {
		return 0
	}
	var center Vec3
	for _, index := range indices {
		center = center.Add(points[index])
	}
	center = center.Scale(1 / float64(len(indices)))

	area := 0.0
	for offset, index := range indices {
		next := indices[(offset+1)%len(indices)]
		a := points[index].Sub(center)
		b := points[next].Sub(center)
		area += 0.5 * a.Cross(b).Length()
	}
	return area
}

func quadArea(points []Vec3, quad Quad) float64 {
	a := points[quad.A]
	b := points[quad.B]
	c := points[quad.C]
	d := points[quad.D]
	return 0.5*b.Sub(a).Cross(c.Sub(a)).Length() +
		0.5*c.Sub(a).Cross(d.Sub(a)).Length()
}

// BindPoseMismatch returns max Euclidean difference and whether the two point
// arrays are bit-for-bit identical.
func BindPoseMismatch(lbs, dq []Vec3) (maxDistance float64, exact bool) {
	exact = len(lbs) == len(dq)
	count := len(lbs)
	if len(dq) < count {
		count = len(dq)
	}
	for index := 0; index < count; index++ {
		difference := lbs[index].Sub(dq[index]).Length()
		maxDistance = math.Max(maxDistance, difference)
		if lbs[index] != dq[index] {
			exact = false
		}
	}
	return maxDistance, exact
}
