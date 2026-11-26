package main

import "math"

// JointMap is the explicit source->target joint correspondence. Both
// skeletons share topology, so it is an identity index table; keeping it
// explicit documents the mapping and lets tests assert full coverage.
func JointMap() [JointCount]int {
	return [JointCount]int{
		JRoot: JRoot, JPelvis: JPelvis, JSpine: JSpine, JNeck: JNeck,
		JHead:      JHead,
		JLShoulder: JLShoulder, JLElbow: JLElbow, JLHand: JLHand,
		JRShoulder: JRShoulder, JRElbow: JRElbow, JRHand: JRHand,
		JLHip: JLHip, JLKnee: JLKnee, JLAnkle: JLAnkle, JLToe: JLToe,
		JRHip: JRHip, JRKnee: JRKnee, JRAnkle: JRAnkle, JRToe: JRToe,
	}
}

type targetFoot struct {
	contact     bool
	lockSole    Vec3
	lockHeading float64
}
type legTarget struct {
	ankle, sole Vec3
	heading     float64
	contact     bool
}

// recenterTargetPelvis eases the pelvis horizontally toward the mean planted
// ankle, matching how the source keeps its mass over the support base.
func recenterTargetPelvis(tp *Pose, targets [2]legTarget) {
	support := Vec3{}
	count := 0.0
	for side := Left; side <= Right; side++ {
		idx := int(side)
		if tp.Contact[idx] {
			support = support.Add(targets[idx].ankle)
			count++
		}
	}
	if count == 0 {
		return
	}
	support = support.Scale(1 / count)
	const blend = 0.65
	pelvis := tp.Pos[JPelvis]
	pelvis.X += (support.X - pelvis.X) * blend
	pelvis.Y += (support.Y - pelvis.Y) * blend
	tp.Pos[JPelvis] = pelvis
	tp.Pos[JRoot] = pelvis
}

// RetargetMotion converts source motion onto the target skeleton.
//
// Compensation is per-bone rather than a single uniform scale: each target
// segment keeps its own length. Root horizontal translation is scaled by the
// leg-length ratio, while planted soles are pinned to their world contact
// points and the longer legs are folded with two-bone IK, eliminating sliding.
func RetargetMotion(src, dst *Skeleton, source []Pose) []Pose {
	out := make([]Pose, len(source))
	mapping := JointMap()
	for i := range mapping {
		if mapping[i] != i {
			panic("retarget: joint mapping must cover every joint")
		}
	}

	rootScale := dst.legLength() / src.legLength()
	feet := [2]targetFoot{}
	var smoothedPelvis Vec3
	haveSmoothed := false

	for f, sp := range source {
		tp := Pose{Heading: sp.Heading, Contact: sp.Contact}

		// Root: scale horizontal path offsets; Z stays on the ground plane.
		tp.Root = v3(sp.Root.X*rootScale, sp.Root.Y*rootScale, 0)
		tp.Pos[JRoot] = tp.Root

		// Root->pelvis must use the target's own standing height, not the
		// source height: this is the vertical part of scale compensation.
		tp.Pos[JPelvis] = tp.Root.Add(v3(0, 0,
			(dst.legLength()-0.03)+pelvisVerticalBob(sp, dst, src)))
		tp.Pos[JRoot] = tp.Pos[JPelvis]

		retargetLegs(src, dst, sp, &tp, &feet, rootScale)
		// Smooth pelvis travel so a support switch (including the fast turn)
		// eases in instead of teleporting the hip.
		if !haveSmoothed {
			smoothedPelvis = tp.Pos[JPelvis]
			haveSmoothed = true
		} else {
			smoothedPelvis = smoothedPelvis.
				Add(tp.Pos[JPelvis].Sub(smoothedPelvis).Scale(0.3))
		}
		delta := smoothedPelvis.Sub(tp.Pos[JPelvis])
		tp.Pos[JPelvis] = smoothedPelvis
		tp.Pos[JRoot] = smoothedPelvis
		// Re-solve legs against the smoothed pelvis while keeping ankles fixed.
		resmoothedLegs(dst, &tp, delta)
		retargetUpperBody(dst, sp, &tp)

		for side := Left; side <= Right; side++ {
			idx := int(side)
			tp.KneeDeg[idx] = KneeAngle(tp.Pos, side)
		}
		tp.Cop = centreOfPressure(tp)
		out[f] = tp
	}
	return out
}

func (sk *Skeleton) legLength() float64 {
	return sk.Length[JLKnee] + sk.Length[JLAnkle]
}

// resmoothedLegs repositions hips after pelvis smoothing and re-folds the
// knees while ankles (and therefore locked soles) stay exactly in place.
func resmoothedLegs(dst *Skeleton, tp *Pose, delta Vec3) {
	for side := Left; side <= Right; side++ {
		idx := int(side)
		lg := legJoints(side)
		hip := tp.Pos[JPelvis].Add(v3(0, lateralY(side, dst.HipOffset), 0))
		tp.Pos[lg.Hip] = hip
		ankle := ClampAnkleToReach(hip, tp.Pos[lg.Ankle],
			dst.Length[lg.Knee], dst.Length[lg.Ankle])
		tp.Pos[lg.Ankle] = ankle
		heading := tp.Heading
		if tp.Contact[idx] {
			heading = feetHeadingSafe(tp, idx)
		}
		hint := v3(math.Cos(heading), math.Sin(heading), 0)
		tp.Pos[lg.Knee] = TwoBoneIK(hip, ankle,
			dst.Length[lg.Knee], dst.Length[lg.Ankle], hint)
		toe := ankle.Add(hint.Scale(dst.Length[lg.Toe]))
		toe.Z = ankle.Z
		tp.Pos[lg.Toe] = toe
	}
}

func feetHeadingSafe(tp *Pose, idx int) float64 {
	ankle := tp.Pos[legJoints(Side(idx)).Ankle]
	toe := tp.Pos[legJoints(Side(idx)).Toe]
	dir := toe.Sub(ankle)
	if dir.Len() < 1e-9 {
		return tp.Heading
	}
	return math.Atan2(dir.Y, dir.X)
}

func pelvisVerticalBob(sp Pose, dst, src *Skeleton) float64 {
	srcHip := sp.Pos[JPelvis].Sub(sp.Root).Z - (src.legLength() - 0.04)
	return srcHip * (dst.legLength() / src.legLength())
}

func retargetLegs(src, dst *Skeleton, sp Pose, tp *Pose, feet *[2]targetFoot, rootScale float64) {
	targets := [2]legTarget{}
	for side := Left; side <= Right; side++ {
		idx := int(side)
		lg := legJoints(side)
		var sole Vec3
		heading := sp.Heading
		if sp.Contact[idx] {
			if !feet[idx].contact {
				// Contact begins: pin the sole at the source's exact world
				// point so both skeletons share the same footprint.
				feet[idx].contact = true
				feet[idx].lockSole = sp.Sole[idx]
				feet[idx].lockHeading = sp.Heading
			}
			sole = feet[idx].lockSole
			heading = feet[idx].lockHeading
		} else {
			feet[idx].contact = false
			// Swing: scale the source sole offset relative to its root, which
			// preserves the step shape while matching the longer stride.
			rel := sp.Sole[idx].Sub(sp.Root)
			sole = tp.Root.Add(v3(rel.X*rootScale, rel.Y*rootScale, rel.Z*rootScale))
		}

		fwd := v3(math.Cos(heading), math.Sin(heading), 0)
		ankle := sole.Add(fwd.Scale(-dst.Length[lg.Toe] * 0.5))
		ankle.Z = sole.Z
		targets[idx] = legTarget{ankle: ankle, sole: sole, heading: heading, contact: sp.Contact[idx]}
	}

	// Keep the target pelvis above its support base, then clamp ankles to the
	// final hip positions so every IK target is inside the reach envelope.
	recenterTargetPelvis(tp, targets)
	for side := Left; side <= Right; side++ {
		idx := int(side)
		lg := legJoints(side)
		hip := tp.Pos[JPelvis].Add(v3(0, lateralY(side, dst.HipOffset), 0))
		targets[idx].ankle = ClampAnkleToReach(hip, targets[idx].ankle,
			dst.Length[lg.Knee], dst.Length[lg.Ankle])
	}

	for side := Left; side <= Right; side++ {
		idx := int(side)
		lg := legJoints(side)
		lt := targets[idx]
		hip := tp.Pos[JPelvis].Add(v3(0, lateralY(side, dst.HipOffset), 0))
		tp.Pos[lg.Hip] = hip
		fwd := v3(math.Cos(lt.heading), math.Sin(lt.heading), 0)
		tp.Pos[lg.Ankle] = lt.ankle
		toe := lt.ankle.Add(fwd.Scale(dst.Length[lg.Toe]))
		toe.Z = lt.ankle.Z
		tp.Pos[lg.Toe] = toe
		tp.Sole[idx] = lt.sole
		tp.Pos[lg.Knee] = TwoBoneIK(hip, lt.ankle,
			dst.Length[lg.Knee], dst.Length[lg.Ankle], fwd)
	}
}

// retargetUpperBody transfers source rotations by reproducing each parent->
// child direction at the target bone's own length (rotation mapping +
// per-bone scale compensation).
func retargetUpperBody(dst *Skeleton, sp Pose, tp *Pose) {
	order := []int{JSpine, JNeck, JHead,
		JLShoulder, JLElbow, JLHand, JRShoulder, JRElbow, JRHand}
	for _, j := range order {
		parent := dst.Parent[j]
		srcDir := sp.Pos[j].Sub(sp.Pos[parent]).Normalized()
		tp.Pos[j] = tp.Pos[parent].Add(srcDir.Scale(dst.Length[j]))
	}
}
