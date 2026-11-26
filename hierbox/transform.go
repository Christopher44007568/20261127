package hierbox

// PoseMatrix converts a TRS pose to a matrix and normalizes omitted rotation
// or scale fields to identity.
func PoseMatrix(t Transform) Mat4 {
	if t.Rotation.X == 0 && t.Rotation.Y == 0 && t.Rotation.Z == 0 && t.Rotation.W == 0 {
		t.Rotation = IdentityQuat()
	}
	if t.Scale.X == 0 && t.Scale.Y == 0 && t.Scale.Z == 0 {
		t.Scale = V(1, 1, 1)
	}
	return TRS(t.Position, t.Rotation, t.Scale)
}
