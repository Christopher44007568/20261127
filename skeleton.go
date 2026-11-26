package main

// Joint indices for the 17-joint biped hierarchy.
const (
	JRoot = iota
	JPelvis
	JSpine
	JChest
	JNeck
	JHead
	JLShoulder
	JLElbow
	JLHand
	JRShoulder
	JRElbow
	JRHand
	JLUpLeg
	JLLeg
	JLAnkle
	JRUpLeg
	JRLeg
	JRAnkle
)

const JointCount = 18

// Joint describes one node of the rest-pose hierarchy.
type Joint struct {
	Name   string `json:"name"`
	Parent int    `json:"parent"`
	// Offset is the rest-pose vector from the parent, in metres.
	Offset Vec3 `json:"offset"`
}

// Skeleton is a static hierarchy shared by every clip.
type Skeleton struct {
	Joints []Joint `json:"joints"`
}

// LocalPose is one frame in local (parent-relative) space.
type LocalPose struct {
	Positions []Vec3 `json:"positions"`
	Rotations []Quat `json:"rotations"`
}

// WorldPose is one frame after forward kinematics.
type WorldPose struct {
	Positions []Vec3
	Rotations []Quat
}

// Frame bundles a pose with locomotion bookkeeping.
type Frame struct {
	Pose     LocalPose
	Contacts [2]bool // left, right
	Key      bool
	RootYaw  float64
	FootYaw  [2]float64
}

// Clip is an animation on top of a fixed skeleton.
type Clip struct {
	FPS    float64 `json:"fps"`
	Frames []Frame `json:"frames"`
}

// NewSkeleton builds the reference biped, roughly 1.8 m tall.
func NewSkeleton() *Skeleton {
	sk := &Skeleton{}
	add := func(name string, parent int, off Vec3) int {
		sk.Joints = append(sk.Joints, Joint{Name: name, Parent: parent, Offset: off})
		return len(sk.Joints) - 1
	}
	add("root", -1, Vec3{0, 0.72, 0})
	add("pelvis", JRoot, Vec3{0, 0.05, 0})
	add("spine", JPelvis, Vec3{0, 0.18, 0})
	add("chest", JSpine, Vec3{0, 0.16, 0})
	add("neck", JChest, Vec3{0, 0.10, 0})
	add("head", JNeck, Vec3{0, 0.12, 0})
	add("l_shoulder", JChest, Vec3{0.18, 0.05, 0})
	add("l_elbow", JLShoulder, Vec3{0.28, 0, 0})
	add("l_hand", JLElbow, Vec3{0.25, 0, 0})
	add("r_shoulder", JChest, Vec3{-0.18, 0.05, 0})
	add("r_elbow", JRShoulder, Vec3{-0.28, 0, 0})
	add("r_hand", JRElbow, Vec3{-0.25, 0, 0})
	add("l_up_leg", JPelvis, Vec3{0.10, -0.08, 0})
	add("l_leg", JLUpLeg, Vec3{0, -0.42, 0})
	add("l_ankle", JLLeg, Vec3{0, -0.40, 0})
	add("r_up_leg", JPelvis, Vec3{-0.10, -0.08, 0})
	add("r_leg", JRUpLeg, Vec3{0, -0.42, 0})
	add("r_ankle", JRLeg, Vec3{0, -0.40, 0})
	return sk
}

// ForwardKinematics converts a local pose into world-space transforms.
func (sk *Skeleton) ForwardKinematics(p LocalPose) WorldPose {
	w := WorldPose{
		Positions: make([]Vec3, JointCount),
		Rotations: make([]Quat, JointCount),
	}
	for j := range sk.Joints {
		parent := sk.Joints[j].Parent
		if parent < 0 {
			w.Positions[j] = p.Positions[j]
			w.Rotations[j] = p.Rotations[j].Normalized()
			continue
		}
		rot := w.Rotations[parent].Mul(p.Rotations[j])
		pos := w.Positions[parent].Add(w.Rotations[parent].Rotate(p.Positions[j]))
		w.Rotations[j] = rot
		w.Positions[j] = pos
	}
	return w
}

// WorldClip runs forward kinematics on every frame.
func (sk *Skeleton) WorldClip(c *Clip) []WorldPose {
	out := make([]WorldPose, len(c.Frames))
	for i := range c.Frames {
		out[i] = sk.ForwardKinematics(c.Frames[i].Pose)
	}
	return out
}

// NewLocalPose allocates an identity pose matching the skeleton.
func NewLocalPose(sk *Skeleton) LocalPose {
	p := LocalPose{
		Positions: make([]Vec3, len(sk.Joints)),
		Rotations: make([]Quat, len(sk.Joints)),
	}
	for j := range sk.Joints {
		p.Positions[j] = sk.Joints[j].Offset
		p.Rotations[j] = Quat{W: 1}
	}
	return p
}
