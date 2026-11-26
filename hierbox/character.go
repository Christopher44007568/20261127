package hierbox

import "math"

// Runner bone identifiers. IDs are stable and also used as slice indices.
const (
	BoneRoot BoneID = iota
	BoneSpine
	BoneChest
	BoneNeck
	BoneHead
	BoneTail0
	BoneTail1
	BoneTail2
	BoneTailTip
	BoneThighL
	BoneShinL
	BoneFootL
	BoneToeL
	BoneThighR
	BoneShinR
	BoneFootR
	BoneToeR
	BoneUpperArmL
	BoneForearmL
	BoneHandL
	BoneUpperArmR
	BoneForearmR
	BoneHandR
)

// Pose names required by the acceptance brief.
const (
	PoseSquat     = "squat"
	PosePush      = "push_off"
	PoseTailSwing = "tail_swing"
)

type Runner struct {
	Name      string
	Skeleton  *Skeleton
	Mesh      *Mesh
	Hierarchy *Hierarchy
	RestPose  []Transform
	Poses     map[string][]Transform
	PoseOrder []string
}

func restTRS(x, y, z float64) Transform {
	return Transform{Position: V(x, y, z), Rotation: IdentityQuat(), Scale: V(1, 1, 1)}
}

// NewRunner builds a stylized biped runner with a four-segment tail. Geometry
// is rigidly attached to its named owner bone; hierarchical boxes remain
// conservative under arbitrary bone rotations.
func NewRunner() (*Runner, error) {
	bones := []Bone{
		{ID: BoneRoot, Name: "root", Parent: NoBone, Rest: restTRS(0, 0.94, 0)},
		{ID: BoneSpine, Name: "spine", Parent: BoneRoot, Rest: restTRS(0.03, 0.10, 0)},
		{ID: BoneChest, Name: "chest", Parent: BoneSpine, Rest: restTRS(0.03, 0.25, 0)},
		{ID: BoneNeck, Name: "neck", Parent: BoneChest, Rest: restTRS(0.05, 0.24, 0)},
		{ID: BoneHead, Name: "head", Parent: BoneNeck, Rest: restTRS(0.04, 0.11, 0)},

		{ID: BoneTail0, Name: "tail_0", Parent: BoneRoot, Rest: restTRS(-0.17, 0.05, 0)},
		{ID: BoneTail1, Name: "tail_1", Parent: BoneTail0, Rest: restTRS(-0.29, 0.10, 0)},
		{ID: BoneTail2, Name: "tail_2", Parent: BoneTail1, Rest: restTRS(-0.30, 0.11, 0)},
		{ID: BoneTailTip, Name: "tail_tip", Parent: BoneTail2, Rest: restTRS(-0.25, 0.08, 0)},

		{ID: BoneThighL, Name: "thigh_l", Parent: BoneRoot, Rest: restTRS(0, -0.05, 0.12)},
		{ID: BoneShinL, Name: "shin_l", Parent: BoneThighL, Rest: restTRS(0.02, -0.44, 0)},
		{ID: BoneFootL, Name: "foot_l", Parent: BoneShinL, Rest: restTRS(0.04, -0.42, 0)},
		{ID: BoneToeL, Name: "toe_l", Parent: BoneFootL, Rest: restTRS(0.20, -0.03, 0)},

		{ID: BoneThighR, Name: "thigh_r", Parent: BoneRoot, Rest: restTRS(0, -0.05, -0.12)},
		{ID: BoneShinR, Name: "shin_r", Parent: BoneThighR, Rest: restTRS(0.02, -0.44, 0)},
		{ID: BoneFootR, Name: "foot_r", Parent: BoneShinR, Rest: restTRS(0.04, -0.42, 0)},
		{ID: BoneToeR, Name: "toe_r", Parent: BoneFootR, Rest: restTRS(0.20, -0.03, 0)},

		{ID: BoneUpperArmL, Name: "upper_arm_l", Parent: BoneChest, Rest: restTRS(0.00, 0.20, 0.22)},
		{ID: BoneForearmL, Name: "forearm_l", Parent: BoneUpperArmL, Rest: restTRS(0, -0.30, 0)},
		{ID: BoneHandL, Name: "hand_l", Parent: BoneForearmL, Rest: restTRS(0.03, -0.27, 0)},

		{ID: BoneUpperArmR, Name: "upper_arm_r", Parent: BoneChest, Rest: restTRS(0.00, 0.20, -0.22)},
		{ID: BoneForearmR, Name: "forearm_r", Parent: BoneUpperArmR, Rest: restTRS(0, -0.30, 0)},
		{ID: BoneHandR, Name: "hand_r", Parent: BoneForearmR, Rest: restTRS(0.03, -0.27, 0)},
	}
	s, err := NewSkeleton(bones)
	if err != nil {
		return nil, err
	}

	mesh := buildRunnerMesh()
	localBoxes := mesh.LocalBoxes(len(bones), 0.012)
	h, err := NewHierarchy(s, localBoxes, 0)
	if err != nil {
		return nil, err
	}

	rest := make([]Transform, len(bones))
	for i, b := range bones {
		rest[i] = b.Rest
	}
	r := &Runner{
		Name:      "fox_runner",
		Skeleton:  s,
		Mesh:      mesh,
		Hierarchy: h,
		RestPose:  rest,
		Poses:     make(map[string][]Transform),
		PoseOrder: []string{PoseSquat, PosePush, PoseTailSwing},
	}
	r.Poses[PoseSquat] = r.squatPose()
	r.Poses[PosePush] = r.pushPose()
	r.Poses[PoseTailSwing] = r.tailSwingPose()
	return r, nil
}

func buildRunnerMesh() *Mesh {
	m := NewMesh()

	m.AddBox(BoneRoot, V(0, 0.02, 0), V(0.36, 0.22, 0.31))
	m.AddEllipsoid(BoneRoot, V(-0.03, 0.02, 0), V(0.20, 0.13, 0.17), 10, 6)

	m.AddCylinder(BoneSpine, V(0, 0, 0), V(0.03, 0.25, 0), 0.115, 10)
	m.AddBox(BoneSpine, V(0.02, 0.12, 0), V(0.25, 0.25, 0.30))

	m.AddBox(BoneChest, V(0.04, 0.10, 0), V(0.31, 0.34, 0.43))
	m.AddEllipsoid(BoneChest, V(0.05, 0.13, 0), V(0.18, 0.19, 0.23), 10, 7)

	m.AddCylinder(BoneNeck, V(0, 0, 0), V(0.04, 0.11, 0), 0.075, 9)
	m.AddEllipsoid(BoneHead, V(0.07, 0.11, 0), V(0.165, 0.18, 0.155), 12, 8)
	m.AddBox(BoneHead, V(0.22, 0.075, 0), V(0.22, 0.105, 0.13))
	m.AddEllipsoid(BoneHead, V(0.34, 0.085, 0), V(0.045, 0.04, 0.075), 8, 5)
	m.AddBox(BoneHead, V(0.02, 0.27, 0.105), V(0.055, 0.16, 0.05))
	m.AddBox(BoneHead, V(0.02, 0.27, -0.105), V(0.055, 0.16, 0.05))

	m.AddCylinder(BoneTail0, V(0, 0, 0), V(-0.29, 0.10, 0), 0.075, 10)
	m.AddCylinder(BoneTail1, V(0, 0, 0), V(-0.30, 0.11, 0), 0.061, 10)
	m.AddCylinder(BoneTail2, V(0, 0, 0), V(-0.25, 0.08, 0), 0.047, 9)
	m.AddCylinder(BoneTailTip, V(0, 0, 0), V(-0.20, 0.06, 0), 0.032, 8)
	m.AddEllipsoid(BoneTailTip, V(-0.18, 0.055, 0), V(0.105, 0.060, 0.060), 9, 6)

	addLegMesh := func(thigh, shin, foot, toe BoneID, side float64) {
		m.AddCylinder(thigh, V(0, 0, 0), V(0.02, -0.44, 0), 0.087, 10)
		m.AddEllipsoid(thigh, V(0.01, -0.20, 0), V(0.105, 0.22, 0.105), 9, 7)
		m.AddEllipsoid(thigh, V(0.02, -0.44, 0), V(0.095, 0.09, 0.095), 8, 5)
		m.AddCylinder(shin, V(0, 0, 0), V(0.04, -0.42, 0), 0.064, 9)
		m.AddEllipsoid(shin, V(0.03, -0.22, 0), V(0.075, 0.20, 0.075), 8, 6)
		m.AddEllipsoid(shin, V(0.04, -0.42, 0), V(0.073, 0.068, 0.073), 8, 5)
		m.AddBox(foot, V(0.115, -0.045, 0), V(0.31, 0.105, 0.17))
		m.AddBox(foot, V(0.02, -0.005, 0), V(0.13, 0.12, 0.16))
		m.AddBox(toe, V(0.065, -0.018, 0), V(0.15, 0.075, 0.16))
		_ = side
	}
	addLegMesh(BoneThighL, BoneShinL, BoneFootL, BoneToeL, 1)
	addLegMesh(BoneThighR, BoneShinR, BoneFootR, BoneToeR, -1)

	addArmMesh := func(upper, fore, hand BoneID) {
		m.AddEllipsoid(upper, V(0, 0.0, 0), V(0.075, 0.075, 0.075), 8, 5)
		m.AddCylinder(upper, V(0, 0, 0), V(0, -0.30, 0), 0.055, 9)
		m.AddCylinder(fore, V(0, 0, 0), V(0.03, -0.27, 0), 0.046, 8)
		m.AddEllipsoid(hand, V(0.045, -0.07, 0), V(0.060, 0.085, 0.052), 8, 6)
	}
	addArmMesh(BoneUpperArmL, BoneForearmL, BoneHandL)
	addArmMesh(BoneUpperArmR, BoneForearmR, BoneHandR)

	return m
}

func (r *Runner) copyRestPose() []Transform {
	out := make([]Transform, len(r.RestPose))
	copy(out, r.RestPose)
	return out
}

func deg(v float64) float64 { return v * math.Pi / 180 }

func rotZ(angleDeg float64) Quat { return AxisAngle(V(0, 0, 1), deg(angleDeg)) }

func rotY(angleDeg float64) Quat { return AxisAngle(V(0, 1, 0), deg(angleDeg)) }

func withRotation(t Transform, q Quat) Transform {
	t.Rotation = q
	return t
}
