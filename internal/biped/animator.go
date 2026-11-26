package biped

import "math"

const (
	DefaultFrameRate = 60
	physicsSubsteps  = 4
	smoothTimeConst  = 0.32
	eventProximity   = 1
	eventHoldFrames  = 1
)

type Frame struct {
	Index         int           `json:"index"`
	Time          float64       `json:"time_s"`
	DeltaTime     float64       `json:"delta_time_s"`
	TargetSpeed   float64       `json:"target_speed_mps"`
	Speed         float64       `json:"speed_mps"`
	RootX         float64       `json:"root_x_m"`
	Phase         float64       `json:"phase"`
	PhaseRate     float64       `json:"phase_rate_hz"`
	Params        MotionParams  `json:"params"`
	Weights       []Weight      `json:"weights"`
	TargetWeights []Weight      `json:"target_weights"`
	WeightSum     float64       `json:"weight_sum"`
	Skeleton      Skeleton      `json:"skeleton"`
	LeftSlip      SlipSample    `json:"left_slip"`
	RightSlip     SlipSample    `json:"right_slip"`
	MergedEvents  []MergedEvent `json:"merged_events,omitempty"`
	Triggered     []MergedEvent `json:"triggered_events,omitempty"`
}

type Animator struct {
	Space       *BlendSpace
	TimeConst   float64
	Time        float64
	Index       int
	TargetSpeed float64
	Speed       float64
	RootX       float64
	Phase       float64
	Locker      *FootLocker
	Gate        *EventGate
	RawEvents   []EventFrame
	activeRaw   []EventFrame
	EventLog    []MergedEvent
}

func NewAnimator(space *BlendSpace) *Animator {
	if space == nil {
		space = NewDefaultBlendSpace()
	}
	return &Animator{
		Space:     space,
		TimeConst: smoothTimeConst,
		Locker:    NewFootLocker(),
		Gate:      NewEventGate(),
	}
}

func (a *Animator) SnapToTarget() {
	a.Speed = a.TargetSpeed
}

func (a *Animator) AdvanceFrame(dt float64) Frame {
	if dt <= 0 {
		dt = 1 / float64(DefaultFrameRate)
	}
	h := dt / physicsSubsteps
	var leftLocked, rightLocked FootTarget
	var leftSlip, rightSlip SlipSample
	var frameRaw []EventFrame
	var phaseRate float64
	var weights []Weight
	var params MotionParams
	var leftRawTravel, leftLockedTravel float64
	var rightRawTravel, rightLockedTravel float64

	for sub := 0; sub < physicsSubsteps; sub++ {
		beforePhase := a.Phase
		a.Time += h
		a.Speed = expApproach(a.Speed, a.TargetSpeed, h, a.TimeConst)
		weights = a.Space.TargetWeights(a.Speed)
		params = BlendMotion(a.Space, a.Speed, weights)
		phaseRate = PhysicalPhaseRate(params)
		a.Phase = wrapPhase(a.Phase + phaseRate*h)
		a.RootX += a.Speed * h

		leftRaw := GaitFootTarget(a.Phase, Left, params)
		rightRaw := GaitFootTarget(a.Phase, Right, params)
		leftLocked, leftSlip = a.Locker.Update(a.RootX, h, Left, leftRaw)
		rightLocked, rightSlip = a.Locker.Update(a.RootX, h, Right, rightRaw)
		leftRawTravel += math.Abs(leftSlip.RawStepM)
		leftLockedTravel += math.Abs(leftSlip.LockedStepM)
		rightRawTravel += math.Abs(rightSlip.RawStepM)
		rightLockedTravel += math.Abs(rightSlip.LockedStepM)
		frameRaw = append(frameRaw, a.collectEvents(beforePhase, a.Phase, weights)...)
	}

	a.Index++
	a.RawEvents = append(a.RawEvents, frameRaw...)
	a.activeRaw = append(a.activeRaw, frameRaw...)
	activeRaw := make([]EventFrame, 0, len(a.activeRaw))
	for _, event := range a.activeRaw {
		if event.End() >= a.Index {
			activeRaw = append(activeRaw, event)
		}
	}
	a.activeRaw = activeRaw
	merged := MergeEventFrames(a.activeRaw, eventProximity)
	triggered := a.Gate.Poll(merged)
	a.EventLog = append(a.EventLog, triggered...)

	targetWeights := a.Space.TargetWeights(a.TargetSpeed)
	skeleton := BuildSkeleton(params, a.Phase, a.Time, leftLocked, rightLocked)
	leftSlip.RawSpeed = leftRawTravel / dt
	leftSlip.LockedSpeed = leftLockedTravel / dt
	leftSlip.RawStepM = leftRawTravel
	leftSlip.LockedStepM = leftLockedTravel
	rightSlip.RawSpeed = rightRawTravel / dt
	rightSlip.LockedSpeed = rightLockedTravel / dt
	rightSlip.RawStepM = rightRawTravel
	rightSlip.LockedStepM = rightLockedTravel
	return Frame{
		Index:         a.Index,
		Time:          a.Time,
		DeltaTime:     dt,
		TargetSpeed:   a.TargetSpeed,
		Speed:         a.Speed,
		RootX:         a.RootX,
		Phase:         a.Phase,
		PhaseRate:     phaseRate,
		Params:        params,
		Weights:       weights,
		TargetWeights: targetWeights,
		WeightSum:     WeightSum(weights),
		Skeleton:      skeleton,
		LeftSlip:      leftSlip,
		RightSlip:     rightSlip,
		MergedEvents:  merged,
		Triggered:     triggered,
	}
}

func SyncedPhaseRate(space *BlendSpace, weights []Weight) float64 {
	var rate float64
	for _, weight := range weights {
		clip, ok := space.Clip(weight.ID)
		if ok {
			rate += weight.Value * clip.Cadence
		}
	}
	return rate
}

func PhysicalPhaseRate(params MotionParams) float64 {
	if params.Reach < 1e-5 || params.StanceFraction < 1e-5 {
		return 0
	}
	return math.Max(0, params.Speed) * params.StanceFraction / (2 * params.Reach)
}

type eventTemplate struct {
	kind  EventKind
	phase float64
}

func clipEventTemplates(clip Clip) []eventTemplate {
	templates := []eventTemplate{
		{kind: EventLeftHeel, phase: 0},
		{kind: EventRightHeel, phase: 0.5},
	}
	if clip.StanceFraction < 0.98 && clip.Reach > 1e-4 {
		templates = append(templates,
			eventTemplate{kind: EventLeftToe, phase: clip.StanceFraction},
			eventTemplate{kind: EventRightToe, phase: wrapPhase(clip.StanceFraction - 0.5)},
		)
	}
	return templates
}

func (a *Animator) collectEvents(before, after float64, weights []Weight) []EventFrame {
	var events []EventFrame
	for _, weight := range weights {
		if weight.Value < 0.08 {
			continue
		}
		clip, ok := a.Space.Clip(weight.ID)
		if !ok || clip.Reach <= 1e-4 {
			continue
		}
		for _, template := range clipEventTemplates(clip) {
			if phaseCrossed(before, after, template.phase) {
				events = append(events, EventFrame{
					Frame:    a.Index + 1,
					EndFrame: a.Index + 1 + eventHoldFrames,
					Kind:     template.kind,
					Source:   clip.ID,
				})
			}
		}
	}
	return events
}

func phaseCrossed(before, after, eventPhase float64) bool {
	const epsilon = 1e-9
	if after >= before {
		return eventPhase > before+epsilon && eventPhase <= after+epsilon
	}
	return eventPhase > before+epsilon || eventPhase <= after+epsilon
}
