package biped

import "math"

const (
	ScenarioDuration = 7.2
	ScenarioFPS      = 60
)

type SpeedSchedule func(time float64) float64

func DefaultSchedule(time float64) float64 {
	switch {
	case time < 0.8:
		return 0
	case time < 2.8:
		return 1.5
	case time < 5.2:
		return 4.2
	default:
		return 1.15
	}
}

type Scenario struct {
	Duration float64
	FPS      int
	Schedule SpeedSchedule `json:"-"`
}

type Result struct {
	Scenario        Scenario      `json:"scenario"`
	Frames          []Frame       `json:"frames"`
	RawEvents       []EventFrame  `json:"raw_events"`
	Events          []MergedEvent `json:"events"`
	TriggeredEvents []MergedEvent `json:"triggered_events"`
	Naive           []NaiveSlip   `json:"naive_slip"`
	Summary         Summary       `json:"summary"`
}

func RunScenario(scenario Scenario) *Result {
	if scenario.Duration <= 0 {
		scenario.Duration = ScenarioDuration
	}
	if scenario.FPS <= 0 {
		scenario.FPS = ScenarioFPS
	}
	if scenario.Schedule == nil {
		scenario.Schedule = DefaultSchedule
	}

	space := NewDefaultBlendSpace()
	animator := NewAnimator(space)
	dt := 1 / float64(scenario.FPS)
	frameCount := int(math.Round(scenario.Duration * float64(scenario.FPS)))
	frames := make([]Frame, 0, frameCount)

	for i := 0; i < frameCount; i++ {
		animator.TargetSpeed = scenario.Schedule(animator.Time)
		frames = append(frames, animator.AdvanceFrame(dt))
	}

	events := MergeEventFrames(animator.RawEvents, eventProximity)
	naive := MeasureNaiveSlip(space, frames)
	result := &Result{
		Scenario:        scenario,
		Frames:          frames,
		RawEvents:       append([]EventFrame(nil), animator.RawEvents...),
		Events:          events,
		TriggeredEvents: append([]MergedEvent(nil), animator.EventLog...),
		Naive:           naive,
	}
	result.Summary = Summarize(result)
	return result
}

type FilmStrip struct {
	Speed     float64 `json:"speed_mps"`
	Label     string  `json:"label"`
	FrameStep int     `json:"frame_step"`
	Frames    []Frame `json:"frames"`
}

func BuildFilmStrip(speed float64, count, frameStep int) FilmStrip {
	space := NewDefaultBlendSpace()
	animator := NewAnimator(space)
	animator.TargetSpeed = speed
	animator.Speed = speed
	animator.Phase = 0.03
	animator.RootX = 0
	dt := 1 / float64(DefaultFrameRate)
	if count <= 0 {
		count = 12
	}
	if frameStep <= 0 {
		frameStep = 5
	}

	frames := make([]Frame, 0, count)
	for i := 0; i < count; i++ {
		var frame Frame
		var sampledEvents []MergedEvent
		for j := 0; j < frameStep; j++ {
			animator.TargetSpeed = speed
			frame = animator.AdvanceFrame(dt)
			sampledEvents = append(sampledEvents, frame.Triggered...)
		}
		frame.Triggered = sampledEvents
		frames = append(frames, frame)
	}

	label := "待机"
	switch {
	case speed >= 3:
		label = "跑步"
	case speed > 0.2:
		label = "行走"
	}
	return FilmStrip{Speed: speed, Label: label, FrameStep: frameStep, Frames: frames}
}

type Summary struct {
	FrameCount            int     `json:"frame_count"`
	DurationS             float64 `json:"duration_s"`
	WeightSumMin          float64 `json:"weight_sum_min"`
	WeightSumMax          float64 `json:"weight_sum_max"`
	WeightSumMaxError     float64 `json:"weight_sum_max_error"`
	MaxFrameSpeedJump     float64 `json:"max_frame_speed_jump_mps"`
	MaxLockedSlipSpeed    float64 `json:"max_locked_slip_speed_mps"`
	MeanLockedSlipSpeed   float64 `json:"mean_locked_slip_speed_mps"`
	MaxRawSyncedSlipSpeed float64 `json:"max_raw_synced_slip_speed_mps"`
	MaxNaiveSlipSpeed     float64 `json:"max_naive_slip_speed_mps"`
	RawEventCount         int     `json:"raw_event_count"`
	MergedEventCount      int     `json:"merged_event_count"`
	TriggeredEventCount   int     `json:"triggered_event_count"`
}

func Summarize(result *Result) Summary {
	summary := Summary{
		FrameCount:       len(result.Frames),
		RawEventCount:    len(result.RawEvents),
		MergedEventCount: len(result.Events),
	}
	if len(result.Frames) == 0 {
		return summary
	}
	summary.DurationS = result.Frames[len(result.Frames)-1].Time
	summary.WeightSumMin = math.Inf(1)
	summary.WeightSumMax = math.Inf(-1)

	var lockedSum, lockedCount float64
	for i, frame := range result.Frames {
		summary.WeightSumMin = math.Min(summary.WeightSumMin, frame.WeightSum)
		summary.WeightSumMax = math.Max(summary.WeightSumMax, frame.WeightSum)
		summary.WeightSumMaxError = math.Max(summary.WeightSumMaxError, math.Abs(frame.WeightSum-1))
		summary.MaxLockedSlipSpeed = math.Max(summary.MaxLockedSlipSpeed, frame.LeftSlip.LockedSpeed)
		summary.MaxLockedSlipSpeed = math.Max(summary.MaxLockedSlipSpeed, frame.RightSlip.LockedSpeed)
		summary.MaxRawSyncedSlipSpeed = math.Max(summary.MaxRawSyncedSlipSpeed, frame.LeftSlip.RawSpeed)
		summary.MaxRawSyncedSlipSpeed = math.Max(summary.MaxRawSyncedSlipSpeed, frame.RightSlip.RawSpeed)
		lockedSum += frame.LeftSlip.LockedSpeed + frame.RightSlip.LockedSpeed
		lockedCount += 2
		if i > 0 {
			jump := math.Abs(frame.Speed - result.Frames[i-1].Speed)
			summary.MaxFrameSpeedJump = math.Max(summary.MaxFrameSpeedJump, jump)
		}
	}
	if lockedCount > 0 {
		summary.MeanLockedSlipSpeed = lockedSum / lockedCount
	}
	for _, point := range result.Naive {
		summary.MaxNaiveSlipSpeed = math.Max(summary.MaxNaiveSlipSpeed, point.LeftSlipSpeed)
		summary.MaxNaiveSlipSpeed = math.Max(summary.MaxNaiveSlipSpeed, point.RightSlipSpeed)
	}

	summary.TriggeredEventCount = len(result.TriggeredEvents)
	return summary
}
