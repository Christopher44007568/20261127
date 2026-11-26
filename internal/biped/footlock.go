package biped

import "math"

type SlipSample struct {
	RawStepM      float64 `json:"raw_step_m"`
	LockedStepM   float64 `json:"locked_step_m"`
	RawSpeed      float64 `json:"raw_speed_mps"`
	LockedSpeed   float64 `json:"locked_speed_mps"`
	RawTravelM    float64 `json:"raw_travel_m"`
	LockedTravelM float64 `json:"locked_travel_m"`
}

type footLockState struct {
	anchored            bool
	anchorWorldX        float64
	swingFromRelX       float64
	hasPrevious         bool
	previousContact     bool
	previousRawX        float64
	previousLockedX     float64
	rawContactTravel    float64
	lockedContactTravel float64
}

type FootLocker struct {
	feet [2]footLockState
}

func NewFootLocker() *FootLocker {
	return &FootLocker{}
}

func (l *FootLocker) Update(rootX, dt float64, side Side, raw FootTarget) (FootTarget, SlipSample) {
	state := &l.feet[side]
	rawWorldX := rootX + raw.RelX
	locked := raw
	var lockedWorldX float64

	if raw.Contact {
		if !state.anchored {
			state.anchored = true
			state.anchorWorldX = rawWorldX
			state.hasPrevious = false
		}
		locked.RelX = state.anchorWorldX - rootX
		lockedWorldX = state.anchorWorldX
	} else {
		if state.anchored {
			state.swingFromRelX = state.anchorWorldX - rootX
			state.anchored = false
		}
		locked.RelX = lerp(state.swingFromRelX, raw.RelX, smoothStep(raw.SwingU))
		lockedWorldX = rootX + locked.RelX
	}

	sample := SlipSample{}
	if state.hasPrevious && state.previousContact && raw.Contact && dt > 0 {
		sample.RawStepM = rawWorldX - state.previousRawX
		sample.LockedStepM = lockedWorldX - state.previousLockedX
		sample.RawSpeed = math.Abs(sample.RawStepM) / dt
		sample.LockedSpeed = math.Abs(sample.LockedStepM) / dt
		state.rawContactTravel += math.Abs(sample.RawStepM)
		state.lockedContactTravel += math.Abs(sample.LockedStepM)
	}
	sample.RawTravelM = state.rawContactTravel
	sample.LockedTravelM = state.lockedContactTravel

	state.hasPrevious = true
	state.previousContact = raw.Contact
	state.previousRawX = rawWorldX
	state.previousLockedX = lockedWorldX
	return locked, sample
}

func (l *FootLocker) Travel(side Side) (raw, locked float64) {
	state := l.feet[side]
	return state.rawContactTravel, state.lockedContactTravel
}
