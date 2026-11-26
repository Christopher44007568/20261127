package biped

import "math"

type NaiveSlip struct {
	Time             float64 `json:"time_s"`
	LeftSlipSpeed    float64 `json:"left_slip_speed_mps"`
	RightSlipSpeed   float64 `json:"right_slip_speed_mps"`
	LeftCumulativeM  float64 `json:"left_cumulative_slip_m"`
	RightCumulativeM float64 `json:"right_cumulative_slip_m"`
	LeftContact      float64 `json:"left_contact_weight"`
	RightContact     float64 `json:"right_contact_weight"`
	LeftWorldX       float64 `json:"left_world_x_m"`
	RightWorldX      float64 `json:"right_world_x_m"`
}

func ClipMotionParams(clip Clip) MotionParams {
	return MotionParams{
		Speed:          clip.Speed,
		Cadence:        clip.Cadence,
		StanceFraction: clip.StanceFraction,
		Reach:          clip.Reach,
		FootLift:       clip.FootLift,
		HipHeight:      clip.HipHeight,
		Lean:           clip.Lean,
		ArmSwing:       clip.ArmSwing,
		ElbowBend:      clip.ElbowBend,
		Bob:            clip.Bob,
	}
}

func MeasureNaiveSlip(space *BlendSpace, frames []Frame) []NaiveSlip {
	out := make([]NaiveSlip, 0, len(frames))
	var cumulative [2]float64
	var previousRootX float64
	const substeps = physicsSubsteps

	for frameIndex, frame := range frames {
		point := NaiveSlip{Time: frame.Time}
		rootStart := previousRootX
		timeStart := frame.Time - frame.DeltaTime
		if frameIndex == 0 {
			timeStart = frame.Time
		}

		var travel [2]float64
		var previousWorld [2]float64
		var previousContact [2]float64
		for step := 0; step <= substeps; step++ {
			u := float64(step) / substeps
			time := lerp(timeStart, frame.Time, u)
			rootX := lerp(rootStart, frame.RootX, u)
			for _, side := range []Side{Left, Right} {
				relX, contactWeight := naiveFootPosition(space, frame.Weights, time, side)
				worldX := rootX + relX
				if step > 0 {
					gate := math.Min(contactWeight, previousContact[side])
					travel[side] += gate * math.Abs(worldX-previousWorld[side])
				}
				previousWorld[side] = worldX
				previousContact[side] = contactWeight
			}
		}

		for _, side := range []Side{Left, Right} {
			slipSpeed := 0.0
			if frame.DeltaTime > 0 {
				slipSpeed = travel[side] / frame.DeltaTime
			}
			cumulative[side] += travel[side]
			relX, contactWeight := naiveFootPosition(space, frame.Weights, frame.Time, side)
			worldX := frame.RootX + relX
			if side == Left {
				point.LeftWorldX = worldX
				point.LeftContact = contactWeight
				point.LeftSlipSpeed = slipSpeed
				point.LeftCumulativeM = cumulative[Left]
			} else {
				point.RightWorldX = worldX
				point.RightContact = contactWeight
				point.RightSlipSpeed = slipSpeed
				point.RightCumulativeM = cumulative[Right]
			}
		}
		out = append(out, point)
		previousRootX = frame.RootX
	}
	return out
}

func naiveFootPosition(space *BlendSpace, weights []Weight, time float64, side Side) (float64, float64) {
	var relX, contactWeight float64
	for _, weight := range weights {
		clip, ok := space.Clip(weight.ID)
		if !ok {
			continue
		}
		phase := naiveClipPhase(clip, time)
		target := GaitFootTarget(phase, side, ClipMotionParams(clip))
		relX += weight.Value * target.RelX
		if target.Contact {
			contactWeight += weight.Value
		}
	}
	return relX, contactWeight
}

func naiveClipPhase(clip Clip, time float64) float64 {
	switch clip.ID {
	case "walk":
		return time*clip.Cadence + 0.04
	case "run":
		return time*clip.Cadence + 0.29
	default:
		return 0
	}
}
