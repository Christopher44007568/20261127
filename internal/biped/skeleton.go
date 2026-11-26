package biped

import "math"

func (v Vec2) Add(o Vec2) Vec2      { return Vec2{X: v.X + o.X, Y: v.Y + o.Y} }
func (v Vec2) Sub(o Vec2) Vec2      { return Vec2{X: v.X - o.X, Y: v.Y - o.Y} }
func (v Vec2) Scale(s float64) Vec2 { return Vec2{X: v.X * s, Y: v.Y * s} }

type FootTarget struct {
	Side    Side    `json:"side"`
	RelX    float64 `json:"rel_x_m"`
	SoleY   float64 `json:"sole_y_m"`
	Contact bool    `json:"contact"`
	SwingU  float64 `json:"swing_u"`
}

type FootPose struct {
	Target  FootTarget `json:"target"`
	Ankle   Vec2       `json:"ankle"`
	Heel    Vec2       `json:"heel"`
	Toe     Vec2       `json:"toe"`
	Contact bool       `json:"contact"`
}

type Skeleton struct {
	Pelvis        Vec2     `json:"pelvis"`
	Chest         Vec2     `json:"chest"`
	Neck          Vec2     `json:"neck"`
	Head          Vec2     `json:"head"`
	HeadRadius    float64  `json:"head_radius"`
	ShoulderLeft  Vec2     `json:"shoulder_left"`
	ElbowLeft     Vec2     `json:"elbow_left"`
	HandLeft      Vec2     `json:"hand_left"`
	ShoulderRight Vec2     `json:"shoulder_right"`
	ElbowRight    Vec2     `json:"elbow_right"`
	HandRight     Vec2     `json:"hand_right"`
	HipLeft       Vec2     `json:"hip_left"`
	KneeLeft      Vec2     `json:"knee_left"`
	AnkleLeft     Vec2     `json:"ankle_left"`
	HipRight      Vec2     `json:"hip_right"`
	KneeRight     Vec2     `json:"knee_right"`
	AnkleRight    Vec2     `json:"ankle_right"`
	LeftFoot      FootPose `json:"left_foot"`
	RightFoot     FootPose `json:"right_foot"`
}

type Side uint8

const (
	Left Side = iota
	Right
)

func (s Side) String() string {
	if s == Left {
		return "left"
	}
	return "right"
}

func GaitFootTarget(phase float64, side Side, params MotionParams) FootTarget {
	offset := 0.0
	if side == Right {
		offset = 0.5
	}
	q := wrapPhase(phase + offset)
	stance := clamp(params.StanceFraction, 0.05, 1)
	if q < stance {
		u := q / stance
		return FootTarget{
			Side:    side,
			RelX:    params.Reach * (1 - 2*u),
			SoleY:   0,
			Contact: true,
		}
	}
	u := (q - stance) / (1 - stance)
	e := smoothStep(u)
	return FootTarget{
		Side:    side,
		RelX:    lerp(-params.Reach, params.Reach, e),
		SoleY:   params.FootLift * math.Sin(math.Pi*u),
		Contact: false,
		SwingU:  u,
	}
}

func twoBoneIK(hip, target Vec2, upperLength, lowerLength, bendSign float64) (Vec2, Vec2) {
	toTarget := target.Sub(hip)
	rawDistance := math.Hypot(toTarget.X, toTarget.Y)
	distance := clamp(rawDistance, 1e-5, upperLength+lowerLength-1e-4)
	var direction Vec2
	if rawDistance < 1e-8 {
		direction = Vec2{X: 0, Y: -1}
	} else {
		direction = toTarget.Scale(1 / rawDistance)
	}
	along := (upperLength*upperLength - lowerLength*lowerLength + distance*distance) / (2 * distance)
	height := math.Sqrt(math.Max(0, upperLength*upperLength-along*along))
	perpendicular := Vec2{X: -direction.Y, Y: direction.X}.Scale(bendSign)
	knee := hip.Add(direction.Scale(along)).Add(perpendicular.Scale(height))
	ankle := hip.Add(direction.Scale(distance))
	return knee, ankle
}

func BuildSkeleton(params MotionParams, phase, animTime float64, leftTarget, rightTarget FootTarget) Skeleton {
	idle := 1 - clamp(params.Reach/0.12, 0, 1)
	breath := math.Sin(animTime*2.1)*0.004 + idle*math.Sin(animTime*1.9)*0.010
	gaitBob := math.Abs(math.Sin(2*math.Pi*phase)) * params.Bob
	hipY := params.HipHeight + breath - gaitBob
	swayX := idle * math.Sin(animTime*1.55) * 0.035
	pelvis := Vec2{X: swayX, Y: hipY}

	torsoLength := 0.52
	leanX := math.Sin(params.Lean) * torsoLength
	leanY := math.Cos(params.Lean) * torsoLength
	chest := Vec2{X: pelvis.X + leanX, Y: pelvis.Y + leanY}
	neck := Vec2{X: chest.X + math.Sin(params.Lean)*0.10, Y: chest.Y + math.Cos(params.Lean)*0.10}
	head := Vec2{X: neck.X + math.Sin(params.Lean)*0.08, Y: neck.Y + 0.10}

	hipLeft := Vec2{X: pelvis.X, Y: pelvis.Y - 0.025}
	hipRight := Vec2{X: pelvis.X, Y: pelvis.Y - 0.025}
	ankleLeftTarget := Vec2{X: leftTarget.RelX, Y: 0.115 + leftTarget.SoleY}
	ankleRightTarget := Vec2{X: rightTarget.RelX, Y: 0.115 + rightTarget.SoleY}

	kneeLeft, ankleLeft := twoBoneIK(hipLeft, ankleLeftTarget, 0.46, 0.45, 1)
	kneeRight, ankleRight := twoBoneIK(hipRight, ankleRightTarget, 0.46, 0.45, 1)

	shoulderLeft := Vec2{X: chest.X, Y: chest.Y + 0.025}
	shoulderRight := Vec2{X: chest.X, Y: chest.Y + 0.025}
	leftArmPhase := wrapPhase(phase + 0.5)
	rightArmPhase := phase
	elbowLeft, handLeft := buildArm(shoulderLeft, params, leftArmPhase, -1, animTime, idle)
	elbowRight, handRight := buildArm(shoulderRight, params, rightArmPhase, 1, animTime, idle)

	return Skeleton{
		Pelvis:        pelvis,
		Chest:         chest,
		Neck:          neck,
		Head:          head,
		HeadRadius:    0.115,
		ShoulderLeft:  shoulderLeft,
		ElbowLeft:     elbowLeft,
		HandLeft:      handLeft,
		ShoulderRight: shoulderRight,
		ElbowRight:    elbowRight,
		HandRight:     handRight,
		HipLeft:       hipLeft,
		KneeLeft:      kneeLeft,
		AnkleLeft:     ankleLeft,
		HipRight:      hipRight,
		KneeRight:     kneeRight,
		AnkleRight:    ankleRight,
		LeftFoot:      buildFootPose(ankleLeft, leftTarget),
		RightFoot:     buildFootPose(ankleRight, rightTarget),
	}
}

func buildArm(shoulder Vec2, params MotionParams, phase float64, side, animTime, idle float64) (Vec2, Vec2) {
	swing := math.Sin(2*math.Pi*phase) * params.ArmSwing * side
	swing += idle * math.Sin(animTime*1.55+side*0.7) * 0.055
	upperAngle := -0.18 + swing + params.Lean*0.25
	upperLength := 0.30
	lowerLength := 0.28
	elbow := Vec2{
		X: shoulder.X + math.Sin(upperAngle)*upperLength,
		Y: shoulder.Y - math.Cos(upperAngle)*upperLength,
	}
	lowerAngle := upperAngle + params.ElbowBend*side*0.35 + 0.18
	hand := Vec2{
		X: elbow.X + math.Sin(lowerAngle)*lowerLength,
		Y: elbow.Y - math.Cos(lowerAngle)*lowerLength,
	}
	return elbow, hand
}

func buildFootPose(ankle Vec2, target FootTarget) FootPose {
	pitch := 0.0
	if !target.Contact {
		pitch = -0.55 * math.Sin(math.Pi*target.SwingU)
	}
	heelOffset := rotateVec(Vec2{X: -0.075, Y: -0.055}, pitch)
	toeOffset := rotateVec(Vec2{X: 0.185, Y: -0.055}, pitch)
	return FootPose{
		Target:  target,
		Ankle:   ankle,
		Heel:    ankle.Add(heelOffset),
		Toe:     ankle.Add(toeOffset),
		Contact: target.Contact,
	}
}

func rotateVec(v Vec2, angle float64) Vec2 {
	c := math.Cos(angle)
	s := math.Sin(angle)
	return Vec2{X: v.X*c - v.Y*s, Y: v.X*s + v.Y*c}
}
