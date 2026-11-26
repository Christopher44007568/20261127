package main

import (
	"math"
)

// Limit is a closed angular interval in radians.
type Limit struct {
	Min, Max float64
}

func (l Limit) contains(x float64) bool {
	return x >= l.Min-1e-9 && x <= l.Max+1e-9
}

func (l Limit) clamp(x float64) float64 { return clampF(x, l.Min, l.Max) }

// ReachStatus classifies how the solver treated a commanded target.
type ReachStatus int

const (
	StatusOK ReachStatus = iota
	// StatusOverReach means d > L1+L2: the chain was fully straightened
	// and the endpoint clamped onto the reach sphere. Render in warning red.
	StatusOverReach
	// StatusTooClose means the target folds the knee past its max-flex limit.
	StatusTooClose
	// StatusLimit means some joint angle limit had to be enforced by clamping.
	StatusLimit
)

func (s ReachStatus) label() string {
	switch s {
	case StatusOverReach:
		return "OVER-REACH"
	case StatusTooClose:
		return "FLEX-LIMIT"
	case StatusLimit:
		return "JOINT-LIMIT"
	default:
		return "OK"
	}
}

// LegSpec is a planar two-bone leg with angular joint limits.
//
// Angle conventions (all signed, radians):
//   - HipFlex: signed angle from the pelvis down axis (-normal) to the
//     thigh; positive is flexion toward the walking direction.
//   - KneeFlex: 0 is a fully straight knee, positive is the only allowed
//     bend direction (knee points toward the pole). A negative value would
//     be hyperextension and is structurally forbidden.
//   - AnkleDev: sole-to-shin angle minus 90 degrees; 0 is neutral.
type LegSpec struct {
	Name       string
	L1, L2     float64
	HipLimit   Limit
	KneeLimit  Limit
	AnkleLimit Limit
}

// Reach is the maximum hip-to-ankle distance, L1+L2.
func (leg LegSpec) Reach() float64 { return leg.L1 + leg.L2 }

// FlexFloor is the minimum hip-ankle distance allowed by the knee's
// maximum-flexion limit (law of cosines).
func (leg LegSpec) FlexFloor() float64 {
	gammaMin := math.Pi - leg.KneeLimit.Max
	v := leg.L1*leg.L1 + leg.L2*leg.L2 - 2*leg.L1*leg.L2*math.Cos(gammaMin)
	return math.Sqrt(math.Max(v, 0))
}

// SolveArgs groups one IK request.
type SolveArgs struct {
	Hip       Vec // root joint
	Target    Vec // commanded ankle/end-effector position
	Pole      Vec // unit direction the knee must bulge toward (forward)
	PelvisAng float64
	FootAng   float64 // commanded sole orientation (for ankle angle)
}

// LegPose is the solved, limit-enforced pose plus diagnostics.
type LegPose struct {
	Hip   Vec
	Knee  Vec
	Ankle Vec

	ThighAng float64
	ShinAng  float64
	FootAng  float64
	HipFlex  float64
	KneeFlex float64
	AnkleDev float64

	Status        ReachStatus
	Flags         []string
	CmdTarget     Vec
	SolvedTarget  Vec
	EndError      float64 // |Ankle - CmdTarget|
	CmdDistance   float64
	SolveDistance float64
}

// Solve runs analytic two-bone IK (law of cosines), picks the knee branch
// toward the pole vector, clamps distance into the feasible annulus and
// enforces every joint limit by angle clamping + forward kinematics.
func (leg LegSpec) Solve(args SolveArgs) LegPose {
	out := LegPose{Hip: args.Hip, CmdTarget: args.Target}

	to := args.Target.sub(args.Hip)
	d := to.len()
	out.CmdDistance = d

	reach := leg.Reach()
	floor := math.Max(math.Abs(leg.L1-leg.L2)+1e-6, leg.FlexFloor())

	dir := to.norm()
	if d < 1e-9 {
		// Degenerate target at the hip: fall back to the pelvis down axis.
		downAng := args.PelvisAng - math.Pi/2
		dir = vec(math.Cos(downAng), math.Sin(downAng))
	}

	dSolve := d
	switch {
	case d > reach:
		// Beyond reach: fully extend and pin the endpoint to the reach sphere.
		out.Status = StatusOverReach
		out.Flags = append(out.Flags, "TARGET BEYOND L1+L2; CHAIN STRAIGHTENED")
		dSolve = reach
	case d < floor:
		out.Status = StatusTooClose
		out.Flags = append(out.Flags, "TARGET INSIDE FLEX FLOOR; KNEE AT MAX FLEX")
		dSolve = floor
	}
	out.SolveDistance = dSolve
	solvedTarget := args.Hip.add(dir.mul(dSolve))
	out.SolvedTarget = solvedTarget

	// Angle between H->T and H->K.
	cosDelta := (leg.L1*leg.L1 + dSolve*dSolve - leg.L2*leg.L2) / (2 * leg.L1 * dSolve)
	cosDelta = clampF(cosDelta, -1, 1)
	delta := math.Acos(cosDelta)
	phi := dir.angle()

	kA := args.Hip.add(vec(math.Cos(phi+delta), math.Sin(phi+delta)).mul(leg.L1))
	kB := args.Hip.add(vec(math.Cos(phi-delta), math.Sin(phi-delta)).mul(leg.L1))
	knee := kA
	if kB.sub(args.Hip).dot(args.Pole) > kA.sub(args.Hip).dot(args.Pole) {
		knee = kB
	}

	thighAng := knee.sub(args.Hip).angle()
	shinAng := solvedTarget.sub(knee).angle()

	hipFlex := wrapAngle(thighAng - (args.PelvisAng - math.Pi/2))
	kneeFlex := wrapAngle(thighAng - shinAng)
	// The pole branch must always yield a non-negative flexion; guard against
	// numerical noise so a knee can never be reported in hyperextension.
	if kneeFlex < 0 {
		kneeFlex = 0
	}
	ankleDev := wrapAngle(args.FootAng-shinAng) - math.Pi/2

	// ---- Joint limit enforcement via angle clamp + forward kinematics. ----
	limited := false
	if !leg.HipLimit.contains(hipFlex) {
		hipFlex = leg.HipLimit.clamp(hipFlex)
		out.Flags = append(out.Flags, "HIP LIMIT CLAMP")
		limited = true
	}
	if !leg.KneeLimit.contains(kneeFlex) {
		kneeFlex = leg.KneeLimit.clamp(kneeFlex)
		out.Flags = append(out.Flags, "KNEE LIMIT CLAMP")
		limited = true
	}
	if !leg.AnkleLimit.contains(ankleDev) {
		ankleDev = leg.AnkleLimit.clamp(ankleDev)
		out.Flags = append(out.Flags, "ANKLE LIMIT CLAMP")
		limited = true
	}
	if limited && out.Status == StatusOK {
		out.Status = StatusLimit
	}

	thighAng = args.PelvisAng - math.Pi/2 + hipFlex
	shinAng = thighAng - kneeFlex
	knee = args.Hip.add(vec(math.Cos(thighAng), math.Sin(thighAng)).mul(leg.L1))
	ankle := knee.add(vec(math.Cos(shinAng), math.Sin(shinAng)).mul(leg.L2))
	footAng := shinAng + math.Pi/2 + ankleDev

	out.Knee = knee
	out.Ankle = ankle
	out.ThighAng = thighAng
	out.ShinAng = shinAng
	out.FootAng = footAng
	out.HipFlex = hipFlex
	out.KneeFlex = kneeFlex
	out.AnkleDev = ankleDev
	out.EndError = ankle.sub(args.Target).len()
	return out
}
