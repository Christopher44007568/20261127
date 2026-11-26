package skin

// DualQuat stores a unit dual quaternion. Real is the rotation; Dual encodes
// translation as 0.5*t*Real.
type DualQuat struct {
	Real Quat
	Dual Quat
}

func DualQuatFromTransform(transform Transform) DualQuat {
	translation := Quat{
		X: transform.Translation.X,
		Y: transform.Translation.Y,
		Z: transform.Translation.Z,
	}
	return DualQuat{
		Real: transform.Rotation,
		Dual: translation.Mul(transform.Rotation).Scale(0.5),
	}
}

func (dq DualQuat) TransformPoint(point Vec3) Vec3 {
	rotation := dq.Real.Rotate(point)
	translationQuat := dq.Dual.Mul(dq.Real.Conjugate()).Scale(2)
	return rotation.Add(Vec3{X: translationQuat.X, Y: translationQuat.Y, Z: translationQuat.Z})
}

// DeformLBS applies linear blend skinning with the vertex weights stored in m.
// The pose transforms are already relative to bind pose (inverse bind matrices
// are identity for this experiment).
func DeformLBS(m Mesh, pose Pose) []Vec3 {
	if pose.IsIdentity() {
		return bindPoints(m)
	}

	result := make([]Vec3, len(m.Vertices))
	for index, vertex := range m.Vertices {
		var sum Vec3
		var totalWeight float64
		for bone, weight := range vertex.Weights {
			if weight == 0 {
				continue
			}
			sum = sum.Add(pose.Transforms[bone].Apply(vertex.Bind).Scale(weight))
			totalWeight += weight
		}
		if totalWeight == 0 {
			result[index] = vertex.Bind
			continue
		}
		result[index] = sum.Scale(1 / totalWeight)
	}
	return result
}

// DeformDQ applies dual-quaternion linear blending. Before blending, a
// quaternion is negated when needed so both copies use the shortest arc.
func DeformDQ(m Mesh, pose Pose) []Vec3 {
	if pose.IsIdentity() {
		return bindPoints(m)
	}

	transforms := make([]DualQuat, len(pose.Transforms))
	for bone, transform := range pose.Transforms {
		transforms[bone] = DualQuatFromTransform(transform)
	}

	result := make([]Vec3, len(m.Vertices))
	for index, vertex := range m.Vertices {
		blended := blendDualQuats(transforms, vertex.Weights[:])
		result[index] = blended.TransformPoint(vertex.Bind)
	}
	return result
}

func blendDualQuats(transforms []DualQuat, weights []float64) DualQuat {
	var sum DualQuat
	var totalWeight float64
	haveReference := false

	for bone, weight := range weights {
		if weight == 0 {
			continue
		}
		current := transforms[bone]
		if haveReference && current.Real.Dot(sum.Real) < 0 {
			current.Real = current.Real.Negate()
			current.Dual = current.Dual.Negate()
		}
		if !haveReference {
			haveReference = true
		}
		sum.Real = sum.Real.Add(current.Real.Scale(weight))
		sum.Dual = sum.Dual.Add(current.Dual.Scale(weight))
		totalWeight += weight
	}

	if totalWeight == 0 {
		return DualQuat{Real: QuatIdentity()}
	}
	sum.Real = sum.Real.Scale(1 / totalWeight)
	sum.Dual = sum.Dual.Scale(1 / totalWeight)

	realLength := sum.Real.Length()
	if realLength < 1e-15 {
		return DualQuat{Real: QuatIdentity()}
	}
	inverseLength := 1 / realLength
	sum.Real = sum.Real.Scale(inverseLength)
	sum.Dual = sum.Dual.Scale(inverseLength)

	// A unit dual quaternion requires Real.Dual == 0. Project the dual part
	// back onto that constraint after normalization.
	projection := sum.Real.Dot(sum.Dual)
	sum.Dual = sum.Dual.Sub(sum.Real.Scale(projection))
	return sum
}

func bindPoints(m Mesh) []Vec3 {
	points := make([]Vec3, len(m.Vertices))
	for index := range points {
		points[index] = m.Vertices[index].Bind
	}
	return points
}
