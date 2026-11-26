package main

import "math"

// Bone describes one joint. BindLocal is the bind-pose transform relative
// to the parent bone.
type Bone struct {
	Name      string
	Parent    int
	BindLocal Mat4
	BindWorld Mat4
	InvBind   Mat4
}

// Skeleton is a fixed hierarchy of bones.
type Skeleton struct {
	Bones []Bone
}

// PoseParams drives every joint for one frame. Rot is an intrinsic
// pitch/yaw/roll Euler angle; PelvisOffset translates the root.
type PoseParams struct {
	Rot          []Vec3
	PelvisOffset Vec3
}

func NewPoseParams(n int) PoseParams {
	return PoseParams{Rot: make([]Vec3, n)}
}

func NewSkeleton() *Skeleton {
	type def struct {
		name   string
		parent int
		pos    Vec3
	}
	defs := []def{
		{"pelvis", -1, Vec3{0, 1.05, 0}},
		{"spine", 0, Vec3{0, 0.22, 0}},
		{"chest", 1, Vec3{0, 0.22, 0}},
		{"neck", 2, Vec3{0, 0.18, 0}},
		{"head", 3, Vec3{0, 0.2, 0}},
		{"shoulderL", 2, Vec3{-0.17, 0.12, 0}},
		{"elbowL", 5, Vec3{-0.3, 0, 0}},
		{"wristL", 6, Vec3{-0.27, 0, 0}},
		{"shoulderR", 2, Vec3{0.17, 0.12, 0}},
		{"elbowR", 8, Vec3{0.3, 0, 0}},
		{"wristR", 9, Vec3{0.27, 0, 0}},
	}
	sk := &Skeleton{Bones: make([]Bone, len(defs))}
	for i, d := range defs {
		b := Bone{Name: d.name, Parent: d.parent, BindLocal: Translate(d.pos)}
		if d.parent < 0 {
			b.BindWorld = b.BindLocal
		} else {
			b.BindWorld = sk.Bones[d.parent].BindWorld.Mul(b.BindLocal)
		}
		b.InvBind = b.BindWorld.Invert()
		sk.Bones[i] = b
	}
	return sk
}

// WorldPose evaluates the full bone world matrices for a pose.
func (sk *Skeleton) WorldPose(p PoseParams) []Mat4 {
	out := make([]Mat4, len(sk.Bones))
	for i, b := range sk.Bones {
		rot := RotZ(p.Rot[i].Z).Mul(RotX(p.Rot[i].X)).Mul(RotY(p.Rot[i].Y))
		local := rot.Mul(b.BindLocal)
		if i == 0 {
			local = Translate(p.PelvisOffset).Mul(local)
		}
		if b.Parent < 0 {
			out[i] = local
		} else {
			out[i] = out[b.Parent].Mul(local)
		}
	}
	return out
}

// Weight binds a vertex to a bone with a blend weight.
type Weight struct {
	Bone   int
	Weight float64
}

// SkinVertex applies linear blend skinning in world space:
// p' = sum_i w_i * W_i * B_i^-1 * p.
func SkinVertex(bindPos Vec3, weights []Weight, world []Mat4, sk *Skeleton) Vec3 {
	acc := Vec3{}
	for _, wt := range weights {
		m := world[wt.Bone].Mul(sk.Bones[wt.Bone].InvBind)
		acc = acc.Add(m.TransformPoint(bindPos).Scale(wt.Weight))
	}
	return acc
}

// Animate builds the pose at clip frame f out of frameCount frames.
// The right arm waves quickly (3 cycles) while the whole torso turns
// slowly (1 cycle), so hand and body velocities differ by construction.
func Animate(f, frameCount int) PoseParams {
	n := len(BoneNames)
	p := NewPoseParams(n)
	t := float64(f) / float64(frameCount)
	tau := 2 * math.Pi * t

	turn := 0.38 * math.Sin(tau) // one slow turn over the whole clip
	p.Rot[0].Y = turn * 0.6
	p.Rot[1].Y = turn
	p.Rot[2].Y = turn
	p.Rot[3].Y = 0.15 * math.Sin(tau)
	p.Rot[4].Y = 0.2 * math.Sin(tau)

	// Right arm: raised and waving fast in the screen plane. Negative
	// Z rotation lifts the +X arm toward +Y.
	wave := tau * 3
	p.Rot[8].Z = -1.35 + 0.7*math.Sin(wave)
	p.Rot[9].Z = 0.5 + 0.75*math.Sin(wave+0.9)
	p.Rot[10].Z = -0.4 * math.Sin(wave)

	// Left arm: gentle low-frequency sway.
	p.Rot[5].Z = 0.3 + 0.12*math.Sin(tau)
	p.Rot[6].Z = -0.25

	p.Rot[1].X = 0.05 * math.Sin(tau*2)
	p.PelvisOffset = Vec3{Y: 0.025 * math.Sin(tau*2)}
	return p
}

// BoneNames mirrors NewSkeleton ordering and is used by mesh generation.
var BoneNames = []string{
	"pelvis", "spine", "chest", "neck", "head",
	"shoulderL", "elbowL", "wristL",
	"shoulderR", "elbowR", "wristR",
}
