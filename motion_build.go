package main

import "math"

// buildSpineAndArms evaluates the upper body with simple heading-relative
// pendulum arm swings. Legs are placed independently by IK.
func buildSpineAndArms(sk *Skeleton, p *Pose, f int, fwd Vec3) {
	p.Pos[JSpine] = p.Pos[JPelvis].Add(v3(0, 0, sk.Length[JSpine]))
	p.Pos[JNeck] = p.Pos[JSpine].Add(v3(0, 0, sk.Length[JNeck]))
	p.Pos[JHead] = p.Pos[JNeck].Add(v3(0, 0, sk.Length[JHead]))

	phase := 2 * math.Pi * float64(f) / float64(CycleDur)
	for _, side := range []Side{Left, Right} {
		sign := 1.0
		armHip, armElbow, armHand := JLShoulder, JLElbow, JLHand
		if side == Right {
			sign = -1
			armHip, armElbow, armHand = JRShoulder, JRElbow, JRHand
		}
		// Fore-aft pendulum swing around the shoulder; arms stay roughly
		// vertical and never point upward.
		swing := sign * 0.45 * math.Cos(phase)
		shoulder := p.Pos[JSpine].Add(v3(0, sign*sk.HipOffset*1.55, sk.Length[JSpine]*0.92))
		dir := fwd.Scale(math.Sin(swing)).Add(v3(0, 0, -math.Cos(swing)))
		p.Pos[armHip] = shoulder
		p.Pos[armElbow] = shoulder.Add(dir.Normalized().Scale(sk.Length[armElbow]))
		elbowSwing := swing*0.7 - 0.35
		elbowDir := fwd.Scale(math.Sin(elbowSwing)).Add(v3(0, 0, -math.Cos(elbowSwing)))
		p.Pos[armHand] = p.Pos[armElbow].Add(elbowDir.Normalized().Scale(sk.Length[armHand]))
	}
}

// plantedAnkle returns the ankle/toe targets for a sole locked in world space.
func plantedAnkle(sk *Skeleton, side Side, st footState) (Vec3, Vec3) {
	lg := legJoints(side)
	lockFwd := v3(math.Cos(st.lockHeading), math.Sin(st.lockHeading), 0)
	ankle := st.lockSole.Add(lockFwd.Scale(-sk.Length[lg.Toe] * 0.5))
	ankle.Z = 0
	toe := ankle.Add(lockFwd.Scale(sk.Length[lg.Toe]))
	return ankle, toe
}

// swingAnkle arcs the ankle from the previous sole point to the next one.
func swingAnkle(sk *Skeleton, side Side, f, cs int, fwd, root Vec3) (Vec3, Vec3) {
	lg := legJoints(side)
	prevCs := cs
	nextCs := cs + CycleDur
	t := Clamp(float64(f-prevCs-ContactDur)/float64(nextCs-prevCs-ContactDur), 0, 1)

	var startSole Vec3
	if prevCs < 0 {
		startSole = root.Add(v3(0, lateralY(side, sk.HipOffset), 0))
	} else {
		startSole = rootAt(prevCs).
			Add(forwardAt(prevCs).Scale(stepForwardOffset())).
			Add(v3(0, lateralY(side, sk.HipOffset), 0))
	}
	endSole := rootAt(nextCs).
		Add(forwardAt(nextCs).Scale(stepForwardOffset())).
		Add(v3(0, lateralY(side, sk.HipOffset), 0))

	sole := Lerp(startSole, endSole, Smoothstep(t))
	sole.Z = 0.16 * math.Sin(math.Pi*t) // foot lift arc
	ankle := sole.Add(fwd.Scale(-sk.Length[lg.Toe] * 0.5))
	ankle.Z = sole.Z
	toe := ankle.Add(fwd.Scale(sk.Length[lg.Toe]))
	return ankle, toe
}

func rootAt(f int) Vec3 {
	pos := v3(0, 0, 0)
	for k := 1; k <= f && k < FrameCount; k++ {
		pos = pos.Add(forwardAt(k).Scale(WalkSpeed * FrameDT))
	}
	if f >= FrameCount {
		// Extrapolate the final straight segment for the last swing target.
		pos = pos.Add(forwardAt(FrameCount - 1).
			Scale(WalkSpeed * FrameDT * float64(f-FrameCount+1)))
	}
	return pos
}

type anklePlan struct {
	ankle, toe Vec3
	heading    float64
}

// planAnkles updates contact lock state and returns this frame's ankle
// targets; the pelvis is recentred and smoothed before the IK solve.
func planAnkles(sk *Skeleton, p *Pose, f int, fwd Vec3, rootPos Vec3,
	rootPath []Vec3, feet *[2]footState) [2]anklePlan {
	var plans [2]anklePlan
	for side := Left; side <= Right; side++ {
		idx := int(side)
		cs := contactStart(side, f)
		age := f - cs
		var ankle, toe Vec3
		heading := headingAt(f)
		if cs < 0 || age >= ContactDur {
			feet[idx].contact = false
			p.Contact[idx] = false
			ankle, toe = swingAnkle(sk, side, f, cs, fwd, rootPos)
			hipGuess := p.Pos[JPelvis].Add(v3(0, lateralY(side, sk.HipOffset), 0))
			ankle = ClampAnkleToReach(hipGuess, ankle,
				sk.Length[legJoints(side).Knee], sk.Length[legJoints(side).Ankle])
			toe = ankle.Add(fwd.Scale(sk.Length[legJoints(side).Toe]))
			toe.Z = ankle.Z
		} else {
			if !feet[idx].contact {
				feet[idx].contact = true
				feet[idx].startFrame = cs
				feet[idx].lockHeading = headingAt(cs)
				feet[idx].lockSole = rootPath[cs].
					Add(forwardAt(cs).Scale(stepForwardOffset())).
					Add(v3(0, lateralY(side, sk.HipOffset), 0))
			}
			p.Contact[idx] = true
			ankle, toe = plantedAnkle(sk, side, feet[idx])
			heading = feet[idx].lockHeading
		}
		plans[idx] = anklePlan{ankle, toe, heading}
	}
	return plans
}

// solveSourceLegs places hips and folds each leg with two-bone IK.
func solveSourceLegs(sk *Skeleton, p *Pose, plans [2]anklePlan) {
	for side := Left; side <= Right; side++ {
		idx := int(side)
		lg := legJoints(side)
		plan := plans[idx]
		hip := p.Pos[JPelvis].Add(v3(0, lateralY(side, sk.HipOffset), 0))
		ankle := ClampAnkleToReach(hip, plan.ankle,
			sk.Length[lg.Knee], sk.Length[lg.Ankle])
		p.Pos[lg.Hip] = hip
		p.Pos[lg.Ankle] = ankle
		hint := v3(math.Cos(plan.heading), math.Sin(plan.heading), 0)
		toe := ankle.Add(hint.Scale(sk.Length[lg.Toe]))
		toe.Z = ankle.Z
		p.Pos[lg.Toe] = toe
		p.Pos[lg.Knee] = TwoBoneIK(hip, ankle,
			sk.Length[lg.Knee], sk.Length[lg.Ankle], hint)
	}
}

// recenterPelvis moves the pelvis horizontally toward the mean support ankle
// so the body stays above its feet instead of drifting into an extreme lunge.
// The root path (used for scale compensation) is unchanged; only the pelvis
// joint position eases toward the support base by a fixed blend.
func recenterPelvis(p *Pose, plans [2]anklePlan) {
	support := Vec3{}
	count := 0.0
	for side := Left; side <= Right; side++ {
		if p.Contact[int(side)] {
			support = support.Add(plans[int(side)].ankle)
			count++
		}
	}
	if count == 0 {
		return
	}
	support = support.Scale(1 / count)
	const blend = 0.65
	pelvis := p.Pos[JPelvis]
	pelvis.X += (support.X - pelvis.X) * blend
	pelvis.Y += (support.Y - pelvis.Y) * blend
	p.Pos[JPelvis] = pelvis
	p.Pos[JRoot] = pelvis
}

// centreOfPressure weights planted soles equally; during double support the
// CoP travels between them, during single support it sits on the stance foot.
func centreOfPressure(p Pose) Vec3 {
	count := 0.0
	cop := Vec3{}
	for side := Left; side <= Right; side++ {
		if p.Contact[int(side)] {
			cop = cop.Add(p.Sole[int(side)])
			count++
		}
	}
	if count == 0 {
		return p.Pos[JPelvis]
	}
	cop = cop.Scale(1 / count)
	cop.Z = 0.02
	return cop
}
