package skin

import (
	"math"
	"testing"
)

func TestDualQuatMatchesRigidTransform(t *testing.T) {
	transform := Transform{
		Rotation:    QuatFromAxisAngle(Vec3{X: 1, Y: 2, Z: -1}, 1.13),
		Translation: Vec3{X: 0.7, Y: -1.2, Z: 2.4},
	}
	dq := DualQuatFromTransform(transform)
	points := []Vec3{
		{X: 0.3, Y: -0.8, Z: 1.1},
		{X: -2.0, Y: 0.4, Z: 0.2},
		{},
	}
	for _, point := range points {
		got := dq.TransformPoint(point)
		want := transform.Apply(point)
		if distance := got.Sub(want).Length(); distance > 1e-12 {
			t.Fatalf("DualQuat transform mismatch: got %+v, want %+v, distance %.3g", got, want, distance)
		}
	}
}

func TestBlendDualQuaternionsNormalizes(t *testing.T) {
	blend := blendDualQuats([]DualQuat{
		DualQuatFromTransform(IdentityTransform()),
		DualQuatFromTransform(Transform{
			Rotation:    QuatFromAxisAngle(Vec3{Z: 1}, math.Pi/2),
			Translation: Vec3{X: 1, Y: 2, Z: 3},
		}),
	}, []float64{0.35, 0.65})

	if math.Abs(blend.Real.Length()-1) > 1e-12 {
		t.Fatalf("real part is not normalized: %.16g", blend.Real.Length())
	}
	if math.Abs(blend.Real.Dot(blend.Dual)) > 1e-12 {
		t.Fatalf("dual part is not orthogonal to real part: %.16g", blend.Real.Dot(blend.Dual))
	}
	transformed := blend.TransformPoint(Vec3{X: 0.25, Y: -0.3, Z: 0.5})
	if math.IsNaN(transformed.X) || math.IsInf(transformed.X, 0) {
		t.Fatalf("non-finite transformed point: %+v", transformed)
	}
}

func TestBindPoseIsBitExact(t *testing.T) {
	mesh := BuildTube()
	for index := range mesh.Vertices {
		forearm := float64(index%101) / 100
		mesh.Vertices[index].Weights = [defaultBoneCount]float64{1 - forearm, forearm}
	}
	pose := ElbowPose("bind", 0, 0)
	lbs := DeformLBS(mesh, pose)
	dq := DeformDQ(mesh, pose)

	distance, exact := BindPoseMismatch(lbs, dq)
	if !exact || distance != 0 {
		t.Fatalf("bind pose must be bit-exact: exact=%t distance=%.16g", exact, distance)
	}
	for index, vertex := range mesh.Vertices {
		if lbs[index] != vertex.Bind || dq[index] != vertex.Bind {
			t.Fatalf("vertex %d was modified in bind pose: lbs=%+v dq=%+v bind=%+v", index, lbs[index], dq[index], vertex.Bind)
		}
	}
}

func TestWeightsAreSharedAndNormalized(t *testing.T) {
	mesh := BuildTube()
	for index, vertex := range mesh.Vertices {
		sum := vertex.Weights[0] + vertex.Weights[1]
		if math.Abs(sum-1) > 1e-12 {
			t.Fatalf("vertex %d weights sum to %.16g", index, sum)
		}
		if vertex.Weights[0] < 0 || vertex.Weights[1] < 0 {
			t.Fatalf("vertex %d has negative weight: %+v", index, vertex.Weights)
		}
	}

	pose := ElbowPose("test", 1.1, 0.7)
	lbs := DeformLBS(mesh, pose)
	dq := DeformDQ(mesh, pose)
	for index, vertex := range mesh.Vertices {
		if vertex.Weights[1] == 1 {
			if distance := lbs[index].Sub(dq[index]).Length(); distance > 1e-12 {
				t.Fatalf("fully forearm-weighted vertex %d differs: %.3g", index, distance)
			}
		}
	}
}

func TestElbowShowsLBSCompressionAndDQBulge(t *testing.T) {
	mesh := BuildTube()
	comparison := Analyze(mesh, ElbowPose("final", math.Pi/2, 2*math.Pi/3))

	if comparison.MinLBSRatio >= 0.60 {
		t.Fatalf("expected visible LBS compression, got minimum %.3f", comparison.MinLBSRatio)
	}
	if comparison.MaxDQRatio <= 1.05 {
		t.Fatalf("expected visible DQ bulge, got maximum %.3f", comparison.MaxDQRatio)
	}
	if comparison.MinDQRatio <= comparison.MinLBSRatio {
		t.Fatalf("DQ should preserve more area than LBS: dq %.3f lbs %.3f", comparison.MinDQRatio, comparison.MinLBSRatio)
	}
	for _, area := range comparison.Areas {
		if area.BindArea <= 0 || math.IsNaN(area.LBSRatio) || math.IsNaN(area.DQRatio) {
			t.Fatalf("invalid area sample: %+v", area)
		}
	}
}

func TestTrajectoryStartsAtExactBindPose(t *testing.T) {
	mesh := BuildTube()
	markers := MarkerVertexIndices(mesh)
	samples := BuildTrajectories(mesh, markers, math.Pi/2, 2*math.Pi/3, 7)

	if len(samples) != 7*len(markers)*2 {
		t.Fatalf("unexpected trajectory sample count: %d", len(samples))
	}
	for _, marker := range markers {
		var lbs, dq *TrajectorySample
		for index := range samples {
			sample := &samples[index]
			if sample.AngleDeg != 0 || sample.Vertex != marker {
				continue
			}
			if sample.Method == "LBS" {
				lbs = sample
			} else {
				dq = sample
			}
		}
		if lbs == nil || dq == nil {
			t.Fatalf("marker %d has no zero-angle samples", marker)
		}
		if lbs.Position != dq.Position || lbs.Position != mesh.Vertices[marker].Bind {
			t.Fatalf("marker %d does not start at exact bind pose: lbs=%+v dq=%+v", marker, lbs.Position, dq.Position)
		}
	}
}
