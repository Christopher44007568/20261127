package main

var (
	colorTorso = Vec3{0.32, 0.55, 0.78}
	colorLimb  = Vec3{0.45, 0.68, 0.85}
	colorSkin  = Vec3{0.86, 0.66, 0.5}
	colorJoint = Vec3{0.38, 0.6, 0.8}
)

// BuildHumanoid assembles an upper-body character from tapered capsules
// and spheres. Every generated vertex carries explicit bone weights.
func BuildHumanoid(sk *Skeleton) *Mesh {
	m := newMesh()
	pos := func(i int) Vec3 {
		t := sk.Bones[i].BindWorld.M
		return Vec3{t[3], t[7], t[11]}
	}
	pelvis, spine, chest, neck, head := pos(0), pos(1), pos(2), pos(3), pos(4)
	shL, elL, wrL := pos(5), pos(6), pos(7)
	shR, elR, wrR := pos(8), pos(9), pos(10)

	// Tapered torso across the three spine joints.
	m.addCapsule(pelvis, spine, 0.24, 0.2, 0, 1, colorTorso)
	m.addCapsule(spine, chest, 0.2, 0.25, 1, 2, colorTorso)

	// Neck and head.
	m.addCapsule(neck.Sub(Vec3{Y: 0.06}), neck.Add(Vec3{Y: 0.05}), 0.07, 0.07, 3, 3, colorSkin)
	m.addSphere(head.Add(Vec3{Y: 0.05}), Vec3{0.16, 0.2, 0.15}, 4, colorSkin)

	// Arms: shoulder -> elbow -> wrist, with hand spheres.
	m.addCapsule(shL, elL, 0.085, 0.07, 5, 6, colorLimb)
	m.addCapsule(elL, wrL, 0.07, 0.055, 6, 7, colorLimb)
	m.addSphere(wrL.Add(Vec3{X: -0.04}), Vec3{0.07, 0.09, 0.06}, 7, colorSkin)
	m.addCapsule(shR, elR, 0.085, 0.07, 8, 9, colorLimb)
	m.addCapsule(elR, wrR, 0.07, 0.055, 9, 10, colorLimb)
	m.addSphere(wrR.Add(Vec3{X: 0.04}), Vec3{0.07, 0.09, 0.06}, 10, colorSkin)

	// Rounded joints hide ring seams while remaining rigidly skinned.
	for _, j := range []struct {
		p     Vec3
		bone  int
		r     float64
		color Vec3
	}{
		{pelvis, 0, 0.2, colorJoint},
		{spine, 1, 0.16, colorJoint},
		{chest, 2, 0.2, colorJoint},
		{shL, 5, 0.1, colorJoint},
		{elL, 6, 0.08, colorJoint},
		{shR, 8, 0.1, colorJoint},
		{elR, 9, 0.08, colorJoint},
	} {
		m.addSphere(j.p, Vec3{j.r, j.r, j.r}, j.bone, j.color)
	}
	return m
}
