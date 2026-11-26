package main

import "math"

// A step is one swing of one foot. Between landing and the next lift the
// ankle is locked to LandPos, which is what makes the source clip have
// zero foot slip during contacts.
type step struct {
	Foot             int
	Lift, Land       float64
	LiftPos, LandPos Vec3
	LiftYaw, LandYaw float64
}

type footPlanner struct {
	steps  []step
	anchor [2]Vec3
	yaw    [2]float64
}

const (
	walkSpeed  = 0.9
	cycleDur   = 0.5
	swingDur   = 0.22
	footSide   = 0.09
	stepHeight = 0.08
	landReach  = 0.04
)

func forwardVec(yaw float64) Vec3 {
	return Vec3{math.Sin(yaw), 0, math.Cos(yaw)}
}

// rootAt returns the root horizontal position and heading at time t.
func rootAt(t float64) (Vec3, float64) {
	const tTurn0, tTurn1 = 0.8, 1.4
	switch {
	case t < tTurn0:
		return Vec3{Z: walkSpeed * t}, 0
	case t < tTurn1:
		u := smoothstep((t - tTurn0) / (tTurn1 - tTurn0))
		return Vec3{Z: walkSpeed * tTurn0}, math.Pi * u
	default:
		pivot := Vec3{Z: walkSpeed * tTurn0}
		back := forwardVec(math.Pi).Scale(walkSpeed * (t - tTurn1))
		return pivot.Add(back), math.Pi
	}
}

func newFootPlanner() *footPlanner {
	fp := &footPlanner{}
	fp.anchor[0] = Vec3{X: footSide}
	fp.anchor[1] = Vec3{X: -footSide}
	fp.planWalkBlock(0, 0.66)
	// Re-plant both feet next to the pivot during the turn, then keep
	// them world-locked for the remainder of the rotation.
	fp.plantTurnFeet()
	fp.planWalkBlock(1.4, 2.2)
	return fp
}

func (fp *footPlanner) planWalkBlock(t0, t1 float64) {
	for foot := 0; foot < 2; foot++ {
		offset := 0.0
		if foot == 1 {
			offset = cycleDur / 2
		}
		t := t0 + offset
		for t+swingDur <= t1 {
			liftPos := fp.anchor[foot]
			liftYaw := fp.yaw[foot]
			landT := t + swingDur
			rootLand, yawLand := rootAt(landT)
			side := footSide
			if foot == 1 {
				side = -footSide
			}
			landPos := rootLand.
				Add(Vec3{X: side}).
				Add(forwardVec(yawLand).Scale(landReach))
			landPos.Y = 0
			fp.steps = append(fp.steps, step{
				Foot: foot, Lift: t, Land: landT,
				LiftPos: liftPos, LandPos: landPos,
				LiftYaw: liftYaw, LandYaw: yawLand,
			})
			fp.anchor[foot] = landPos
			fp.yaw[foot] = yawLand
			t += cycleDur
		}
	}
}

// plantTurnFeet adds short adjustment steps that bring both ankles to
// symmetric support positions next to the turn pivot. They swing during
// the start of the turn, then stay world-locked for the rest of it.
func (fp *footPlanner) plantTurnFeet() {
	const pivotZ = walkSpeed * 0.8
	pivot := Vec3{Z: pivotZ}
	for foot := 0; foot < 2; foot++ {
		side := footSide
		if foot == 1 {
			side = -footSide
		}
		lift := 0.66 + float64(foot)*0.08
		land := lift + swingDur
		landPos := pivot.Add(Vec3{X: side})
		fp.steps = append(fp.steps, step{
			Foot: foot, Lift: lift, Land: land,
			LiftPos: fp.anchor[foot], LandPos: landPos,
			LiftYaw: fp.yaw[foot], LandYaw: math.Pi,
		})
		fp.anchor[foot] = landPos
		fp.yaw[foot] = math.Pi
	}
}

func angleLerp(a, b, u float64) float64 {
	diff := b - a
	for diff > math.Pi {
		diff -= 2 * math.Pi
	}
	for diff < -math.Pi {
		diff += 2 * math.Pi
	}
	return a + diff*u
}

// FootTarget returns the ankle world position, facing yaw and contact flag.
func (fp *footPlanner) FootTarget(foot int, t float64) (Vec3, float64, bool) {
	for i := range fp.steps {
		s := fp.steps[i]
		if s.Foot != foot || t < s.Lift || t > s.Land {
			continue
		}
		u := smoothstep((t - s.Lift) / (s.Land - s.Lift))
		pos := lerpVec3(s.LiftPos, s.LandPos, u)
		pos.Y = stepHeight * math.Sin(math.Pi*u)
		return pos, angleLerp(s.LiftYaw, s.LandYaw, u), false
	}
	// Contact: use the most recent step that has already landed. Before
	// the first lift the foot rests on its initial anchor.
	landed := fp.initialAnchor(foot)
	landedYaw := 0.0
	for i := range fp.steps {
		s := fp.steps[i]
		if s.Foot == foot && t >= s.Land {
			landed = s.LandPos
			landedYaw = s.LandYaw
		}
	}
	return landed, landedYaw, true
}

func (fp *footPlanner) initialAnchor(foot int) Vec3 {
	if foot == 1 {
		return Vec3{X: -footSide}
	}
	return Vec3{X: footSide}
}

// solveTwoBone derives world rotations for a hip-knee-ankle chain so the
// ankle reaches target while the knee bends toward pole.
func solveTwoBone(hip, target, pole Vec3, l1, l2 float64) (Quat, Quat, Vec3) {
	toTarget := target.Sub(hip)
	dist := clamp(toTarget.Len(), 1e-6, l1+l2-1e-6)
	dir := toTarget.Scale(1 / toTarget.Len())
	a := (l1*l1 - l2*l2 + dist*dist) / (2 * dist)
	h := math.Sqrt(math.Max(0, l1*l1-a*a))
	kneeDir := pole.Sub(hip).Sub(dir.Scale(pole.Sub(hip).Dot(dir)))
	if kneeDir.Len() < 1e-8 {
		kneeDir = Vec3{Z: 1}
	}
	kneeDir = kneeDir.Normalized()
	knee := hip.Add(dir.Scale(a)).Add(kneeDir.Scale(h))
	ankle := hip.Add(dir.Scale(dist))
	bindDown := Vec3{Y: -1}
	thigh := FromTo(bindDown, knee.Sub(hip).Normalized())
	shin := FromTo(bindDown, ankle.Sub(knee).Normalized())
	return thigh, shin, ankle
}
