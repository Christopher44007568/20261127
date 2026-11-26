package main

import "math"

// Pose is one evaluated skeleton: root position and absolute joint positions.
type Pose struct {
	Root    Vec3
	Heading float64
	Pos     [JointCount]Vec3
	Sole    [2]Vec3 // world sole point per side
	Contact [2]bool
	Cop     Vec3 // centre of pressure (weighted foot midpoint)
	KneeDeg [2]float64
}

// Frame couples source and target poses plus shared turn metadata.
type Frame struct {
	Source, Target Pose
	TurnRate       float64
}

const (
	FrameCount   = 96
	StepDur      = 12          // frames between opposite-foot steps
	CycleDur     = 4 * StepDur // frames between same-foot contacts (gait cycle)
	ContactDur   = 36          // frames a sole stays planted (covers double support)
	WalkSpeed    = 0.80        // metres per second
	FrameDT      = 1.0 / 30.0
	TurnStartF   = 30
	TurnEndF     = 50
	TurnTotalRad = 110 * math.Pi / 180
	// ContactOffset is how far ahead of the root a fresh sole is planted.
	// Kept inside the reach envelope so normal walking never over-extends;
	// ClampAnkleToReach only trims edge cases during the fast turn.
	ContactOffset = 0.22
)

type footState struct {
	contact     bool
	startFrame  int
	lockSole    Vec3
	lockHeading float64
}

// headingAt returns the root yaw; the turn is a smooth ramp so the angular
// velocity is finite and continuous (no hard corner at the turn boundaries).
func headingAt(f int) float64 {
	u := Clamp((float64(f)-TurnStartF)/(TurnEndF-TurnStartF), 0, 1)
	return TurnTotalRad * Smoothstep(u)
}

func forwardAt(f int) Vec3 {
	h := headingAt(f)
	return v3(math.Cos(h), math.Sin(h), 0)
}

// contactStart gives the most recent contact event for a side at frame f.
func contactStart(side Side, f int) int {
	// Left contacts at 0,48,96,...; right at 12,60,...
	first := 12 * int(side)
	if f < first {
		return -1
	}
	return first + ((f-first)/CycleDur)*CycleDur // CycleDur=48 per leg
}

func stepForwardOffset() float64 { return ContactOffset }

// GenerateSourceMotion builds 96 frames of curved-path walking for sk.
func GenerateSourceMotion(sk *Skeleton) []Pose {
	poses := make([]Pose, FrameCount)
	feet := [2]footState{}
	var smoothed Vec3
	haveSmoothed := false
	rootPath := make([]Vec3, FrameCount)
	headings := make([]float64, FrameCount)
	root := v3(0, 0, 0)
	for f := 0; f < FrameCount; f++ {
		headings[f] = headingAt(f)
		if f > 0 {
			root = root.Add(forwardAt(f).Scale(WalkSpeed * FrameDT))
		}
		rootPath[f] = root
	}

	for f := 0; f < FrameCount; f++ {
		heading := headings[f]
		fwd := forwardAt(f)
		rootPos := rootPath[f]
		p := Pose{Root: rootPos, Heading: heading}

		lgLeg := sk.Length[JLKnee] + sk.Length[JLAnkle]
		bob := 0.018 * math.Cos(2*math.Pi*float64(f)/StepDur)
		p.Pos[JRoot] = rootPos
		p.Pos[JPelvis] = rootPos.Add(v3(0, 0, lgLeg-0.03+bob))
		p.Pos[JRoot] = p.Pos[JPelvis]

		plans := planAnkles(sk, &p, f, fwd, rootPos, rootPath, &feet)
		recenterPelvis(&p, plans)
		if !haveSmoothed {
			smoothed = p.Pos[JPelvis]
			haveSmoothed = true
		} else {
			smoothed = smoothed.Add(p.Pos[JPelvis].Sub(smoothed).Scale(0.3))
		}
		p.Pos[JPelvis] = smoothed
		p.Pos[JRoot] = smoothed
		buildSpineAndArms(sk, &p, f, fwd)
		solveSourceLegs(sk, &p, plans)

		for side := Left; side <= Right; side++ {
			idx := int(side)
			lg := legJoints(side)
			p.KneeDeg[idx] = KneeAngle(p.Pos, side)
			h := heading
			if p.Contact[idx] {
				h = feet[idx].lockHeading
			}
			soleFwd := v3(math.Cos(h), math.Sin(h), 0)
			p.Sole[idx] = p.Pos[lg.Ankle].Add(soleFwd.Scale(sk.Length[lg.Toe] * 0.5))
		}

		p.Cop = centreOfPressure(p)
		poses[f] = p
	}
	return poses
}

func lateralY(side Side, hipOffset float64) float64 {
	if side == Left {
		return hipOffset
	}
	return -hipOffset
}
