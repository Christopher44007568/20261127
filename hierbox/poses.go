package hierbox

func (r *Runner) squatPose() []Transform {
	p := r.copyRestPose()
	p[BoneRoot].Position = V(0.02, 0.58, 0)

	// Deep bilateral flexion: thighs forward, shins folded back relative to
	// their parents, and feet counter-rotated to stay near horizontal.
	for _, side := range []struct{ thigh, shin, foot BoneID }{
		{BoneThighL, BoneShinL, BoneFootL},
		{BoneThighR, BoneShinR, BoneFootR},
	} {
		p[side.thigh] = withRotation(p[side.thigh], rotZ(74))
		p[side.shin] = withRotation(p[side.shin], rotZ(-94))
		p[side.foot] = withRotation(p[side.foot], rotZ(20))
		p[side.foot].Position.Y -= 0.015
	}

	p[BoneSpine] = withRotation(p[BoneSpine], rotZ(-8))
	p[BoneChest] = withRotation(p[BoneChest], rotZ(-10))
	p[BoneNeck] = withRotation(p[BoneNeck], rotZ(12))
	p[BoneHead] = withRotation(p[BoneHead], rotZ(8))

	p[BoneUpperArmL] = withRotation(p[BoneUpperArmL], rotZ(92))
	p[BoneForearmL] = withRotation(p[BoneForearmL], rotZ(-28))
	p[BoneUpperArmR] = withRotation(p[BoneUpperArmR], rotZ(86))
	p[BoneForearmR] = withRotation(p[BoneForearmR], rotZ(-22))

	// Tail balances the forward torso and remains extended behind the squat.
	p[BoneTail0] = withRotation(p[BoneTail0], rotZ(-10))
	p[BoneTail1] = withRotation(p[BoneTail1], rotZ(-8))
	p[BoneTail2] = withRotation(p[BoneTail2], rotZ(6))
	p[BoneTailTip] = withRotation(p[BoneTailTip], rotZ(10))
	return p
}

func (r *Runner) pushPose() []Transform {
	p := r.copyRestPose()
	p[BoneRoot].Position = V(0.03, 0.92, 0)
	p[BoneRoot].Rotation = rotZ(-9)

	// Right/rear leg pushes against the ground; left leg drives forward.
	p[BoneThighR] = withRotation(p[BoneThighR], rotZ(-34))
	p[BoneShinR] = withRotation(p[BoneShinR], rotZ(23))
	p[BoneFootR] = withRotation(p[BoneFootR], rotZ(11))
	p[BoneToeR] = withRotation(p[BoneToeR], rotZ(-8))

	p[BoneThighL] = withRotation(p[BoneThighL], rotZ(64))
	p[BoneShinL] = withRotation(p[BoneShinL], rotZ(-116))
	p[BoneFootL] = withRotation(p[BoneFootL], rotZ(42))

	// Sprint counter-swing in the arms.
	p[BoneUpperArmR] = withRotation(p[BoneUpperArmR], rotZ(78))
	p[BoneForearmR] = withRotation(p[BoneForearmR], rotZ(-42))
	p[BoneUpperArmL] = withRotation(p[BoneUpperArmL], rotZ(-38))
	p[BoneForearmL] = withRotation(p[BoneForearmL], rotZ(34))

	// Forward lean through the spine rather than one root-only rigid box.
	p[BoneSpine] = withRotation(p[BoneSpine], rotZ(-10))
	p[BoneChest] = withRotation(p[BoneChest], rotZ(-12))
	p[BoneNeck] = withRotation(p[BoneNeck], rotZ(16))
	p[BoneHead] = withRotation(p[BoneHead], rotZ(8))

	p[BoneTail0] = withRotation(p[BoneTail0], rotZ(14))
	p[BoneTail1] = withRotation(p[BoneTail1], rotZ(8))
	p[BoneTail2] = withRotation(p[BoneTail2], rotZ(-4))
	return p
}

func (r *Runner) tailSwingPose() []Transform {
	p := r.copyRestPose()
	p[BoneRoot].Position = V(0, 0.95, 0)

	// Easy running stride while the tail curls strongly to character-left.
	p[BoneThighL] = withRotation(p[BoneThighL], rotZ(34))
	p[BoneShinL] = withRotation(p[BoneShinL], rotZ(-48))
	p[BoneFootL] = withRotation(p[BoneFootL], rotZ(15))

	p[BoneThighR] = withRotation(p[BoneThighR], rotZ(-28))
	p[BoneShinR] = withRotation(p[BoneShinR], rotZ(34))
	p[BoneFootR] = withRotation(p[BoneFootR], rotZ(-8))

	p[BoneUpperArmL] = withRotation(p[BoneUpperArmL], rotZ(-34))
	p[BoneForearmL] = withRotation(p[BoneForearmL], rotZ(30))
	p[BoneUpperArmR] = withRotation(p[BoneUpperArmR], rotZ(48))
	p[BoneForearmR] = withRotation(p[BoneForearmR], rotZ(-30))

	p[BoneSpine] = withRotation(p[BoneSpine], rotZ(-5))
	p[BoneChest] = withRotation(p[BoneChest], rotZ(-6))

	p[BoneTail0] = withRotation(p[BoneTail0], rotY(18))
	p[BoneTail1] = withRotation(p[BoneTail1], rotY(24))
	p[BoneTail2] = withRotation(p[BoneTail2], rotY(24))
	p[BoneTailTip] = withRotation(p[BoneTailTip], rotY(18))
	return p
}

// ApplyPose replaces the current hierarchy pose and performs one incremental
// update. It returns the update work counters.
func (r *Runner) ApplyPose(pose []Transform) (UpdateStats, error) {
	if err := r.Hierarchy.SetLocals(pose); err != nil {
		return UpdateStats{}, err
	}
	return r.Hierarchy.Update()
}

// ApplyNamedPose applies one of the three brief poses.
func (r *Runner) ApplyNamedPose(name string) (UpdateStats, error) {
	pose, ok := r.Poses[name]
	if !ok {
		return UpdateStats{}, errUnknownPose(name)
	}
	return r.ApplyPose(pose)
}
