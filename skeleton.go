package main

import "math"

// Joint indices shared by both skeletons. The identical topology is what makes
// the source->target joint mapping a well-defined permutation.
const (
	JRoot = iota
	JPelvis
	JSpine
	JNeck
	JHead
	JLShoulder
	JLElbow
	JLHand
	JRShoulder
	JRElbow
	JRHand
	JLHip
	JLKnee
	JLAnkle
	JLToe
	JRHip
	JRKnee
	JRAnkle
	JRToe
	JointCount
)

// Side identifies a leg.
type Side int

const (
	Left Side = iota
	Right
)

// Skeleton is a named set of segment lengths and offsets in metres.
type Skeleton struct {
	Name   string
	Parent [JointCount]int
	// Rest positions of every joint relative to its parent, in a T-like
	// standing pose facing +X.
	Rest [JointCount]Vec3
	// Bone length from Parent[i] to i, derived from Rest.
	Length [JointCount]float64
	// HipOffset is the lateral distance from pelvis to each hip.
	HipOffset float64
}

func buildSkeleton(name string, hipY, thigh, shin, foot, torso, neck, headLen,
	upperArm, forearm, shoulderW, shoulderH float64) *Skeleton {
	sk := &Skeleton{Name: name, HipOffset: hipY}
	sk.Parent = [JointCount]int{
		JRoot: -1, JPelvis: JRoot, JSpine: JPelvis, JNeck: JSpine, JHead: JNeck,
		JLShoulder: JSpine, JLElbow: JLShoulder, JLHand: JLElbow,
		JRShoulder: JSpine, JRElbow: JRShoulder, JRHand: JRElbow,
		JLHip: JPelvis, JLKnee: JLHip, JLAnkle: JLKnee, JLToe: JLAnkle,
		JRHip: JPelvis, JRKnee: JRHip, JRAnkle: JRKnee, JRToe: JRAnkle,
	}
	legLen := thigh + shin
	sk.Rest = [JointCount]Vec3{
		JRoot:      v3(0, 0, 0),
		JPelvis:    v3(0, 0, legLen),
		JSpine:     v3(0, 0, torso),
		JNeck:      v3(0, 0, neck),
		JHead:      v3(0, 0, headLen),
		JLShoulder: v3(0, shoulderW, shoulderH),
		JLElbow:    v3(0, upperArm, 0),
		JLHand:     v3(0, forearm, 0),
		JRShoulder: v3(0, -shoulderW, shoulderH),
		JRElbow:    v3(0, -upperArm, 0),
		JRHand:     v3(0, -forearm, 0),
		JLHip:      v3(0, hipY, 0),
		JLKnee:     v3(0, 0, -thigh),
		JLAnkle:    v3(0, 0, -shin),
		JLToe:      v3(foot, 0, 0),
		JRHip:      v3(0, -hipY, 0),
		JRKnee:     v3(0, 0, -thigh),
		JRAnkle:    v3(0, 0, -shin),
		JRToe:      v3(foot, 0, 0),
	}
	for i := range sk.Rest {
		sk.Length[i] = sk.Rest[i].Len()
	}
	return sk
}

// NewSourceSkeleton is the shorter actor (1.70 m class proportions).
func NewSourceSkeleton() *Skeleton {
	return buildSkeleton("source A", 0.11, 0.42, 0.43, 0.16,
		0.50, 0.10, 0.18, 0.30, 0.27, 0.17, 0.06)
}

// NewTargetSkeleton is the taller, longer-limbed actor (1.95 m class).
// Thigh/shin and hip width differ non-uniformly, so uniform scale alone fails.
func NewTargetSkeleton() *Skeleton {
	return buildSkeleton("target B", 0.14, 0.52, 0.50, 0.18,
		0.55, 0.11, 0.20, 0.34, 0.30, 0.20, 0.07)
}

// LegJoints groups the joint indices of one leg.
type LegJoints struct {
	Hip, Knee, Ankle, Toe int
}

func legJoints(side Side) LegJoints {
	if side == Left {
		return LegJoints{JLHip, JLKnee, JLAnkle, JLToe}
	}
	return LegJoints{JRHip, JRKnee, JRAnkle, JRToe}
}

// KneeAngle returns the interior knee angle in degrees for an evaluated pose.
func KneeAngle(pos [JointCount]Vec3, side Side) float64 {
	lg := legJoints(side)
	up := pos[lg.Hip].Sub(pos[lg.Knee])
	down := pos[lg.Ankle].Sub(pos[lg.Knee])
	cos := up.Dot(down) / (up.Len() * down.Len())
	return math.Acos(Clamp(cos, -1, 1)) * 180 / math.Pi
}
