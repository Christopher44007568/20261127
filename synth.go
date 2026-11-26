package main

import "math"

const (
	synthFPS        = 60
	synthFrameCount = 132
)

// SynthesizeClip creates a deterministic walk / 180-degree turn / walk back
// clip. Stance ankles are world-locked, so foot slip is exactly zero in the
// original clip and any slip after compression is purely a codec artifact.
func SynthesizeClip(sk *Skeleton) *Clip {
	planner := newFootPlanner()
	clip := &Clip{FPS: synthFPS}

	for f := 0; f < synthFrameCount; f++ {
		t := float64(f) / synthFPS
		pose := NewLocalPose(sk)
		rootPos, heading := rootAt(t)
		phase := 2 * math.Pi * t / cycleDur

		rootPos.Y = 0.72 + 0.003*math.Sin(2*phase)
		pose.Positions[JRoot] = rootPos
		pose.Rotations[JRoot] = RotY(heading)

		pose.Rotations[JPelvis] = RotY(0.12 * math.Sin(phase)).
			Mul(RotX(0.04 * math.Sin(phase)))
		pose.Rotations[JSpine] = RotX(-0.05 * math.Sin(phase))
		pose.Rotations[JChest] = RotY(-0.08 * math.Sin(phase))
		pose.Rotations[JNeck] = RotY(0.05 * math.Sin(phase))
		pose.Rotations[JHead] = RotY(-0.10 * math.Sin(phase)).
			Mul(RotX(0.03 * math.Cos(phase)))

		armSwingL := 0.55 * math.Sin(phase)
		armSwingR := -armSwingL
		pose.Rotations[JLShoulder] = RotX(armSwingL).Mul(RotZ(0.10))
		pose.Rotations[JRShoulder] = RotX(armSwingR).Mul(RotZ(-0.10))
		pose.Rotations[JLElbow] = RotX(0.5 + 0.25*math.Max(0, math.Sin(phase)))
		pose.Rotations[JRElbow] = RotX(0.5 + 0.25*math.Max(0, -math.Sin(phase)))

		frame := Frame{
			Pose:    pose,
			RootYaw: heading,
		}

		buildLeg(&frame, sk, planner, 0, t, heading)
		buildLeg(&frame, sk, planner, 1, t, heading)

		frame.Key = isKeyFrame(f, t, planner)
		clip.Frames = append(clip.Frames, frame)
	}
	clip.Frames[0].Key = true
	clip.Frames[synthFrameCount-1].Key = true
	return clip
}

func buildLeg(frame *Frame, sk *Skeleton, planner *footPlanner, foot int, t, heading float64) {
	upLeg, leg, ankleJoint := JLUpLeg, JLLeg, JLAnkle
	hipX := 0.10
	poleX := 0.12
	if foot == 1 {
		upLeg, leg, ankleJoint = JRUpLeg, JRLeg, JRAnkle
		hipX, poleX = -0.10, -0.12
	}

	target, footYaw, contact := planner.FootTarget(foot, t)
	frame.Contacts[foot] = contact
	frame.FootYaw[foot] = footYaw

	// Compute the pelvis world transform through FK so the hip pivot
	// includes the root translation and the small pelvis sway.
	rootRot := frame.Pose.Rotations[JRoot]
	pelvisRot := rootRot.Mul(frame.Pose.Rotations[JPelvis])
	pelvisWorld := frame.Pose.Positions[JRoot].
		Add(rootRot.Rotate(frame.Pose.Positions[JPelvis]))
	hip := pelvisWorld.Add(pelvisRot.Rotate(Vec3{X: hipX, Y: -0.08}))
	pole := hip.Add(pelvisRot.Rotate(Vec3{X: poleX, Z: 0.06}))

	l1 := sk.Joints[leg].Offset.Len()
	l2 := sk.Joints[ankleJoint].Offset.Len()
	thigh, shin, _ := solveTwoBone(hip, target, pole, l1, l2)

	frame.Pose.Rotations[upLeg] = pelvisRot.Inv().Mul(thigh)
	frame.Pose.Rotations[leg] = thigh.Inv().Mul(shin)

	footWorld := RotY(footYaw)
	frame.Pose.Rotations[ankleJoint] = shin.Inv().Mul(footWorld)
}

// isKeyFrame marks clip boundaries, turn boundaries and every heel strike.
func isKeyFrame(f int, t float64, planner *footPlanner) bool {
	if f == 0 || f == synthFrameCount-1 {
		return true
	}
	for _, mark := range []float64{0.8, 1.4} {
		if math.Abs(t-mark) < 1e-9 {
			return true
		}
	}
	for _, s := range planner.steps {
		if math.Abs(t-s.Land) < 1e-9 || math.Abs(t-s.Lift) < 1e-9 {
			return true
		}
	}
	return false
}

// RotZ returns a rotation of angle radians around the Z axis.
func RotZ(a float64) Quat {
	c, s := math.Cos(a/2), math.Sin(a/2)
	return Quat{Z: s, W: c}
}
