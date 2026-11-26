package biped

type Clip struct {
	ID             string  `json:"id"`
	Label          string  `json:"label"`
	Speed          float64 `json:"speed"`
	Cadence        float64 `json:"cadence_hz"`
	NativeFrames   int     `json:"native_frames"`
	StanceFraction float64 `json:"stance_fraction"`
	Reach          float64 `json:"reach_m"`
	FootLift       float64 `json:"foot_lift_m"`
	HipHeight      float64 `json:"hip_height_m"`
	Lean           float64 `json:"lean_rad"`
	ArmSwing       float64 `json:"arm_swing_rad"`
	ElbowBend      float64 `json:"elbow_bend_rad"`
	Bob            float64 `json:"bob_m"`
}

func IdleClip() Clip {
	return Clip{
		ID:             "idle",
		Label:          "待机",
		Speed:          0,
		Cadence:        0,
		NativeFrames:   48,
		StanceFraction: 1,
		Reach:          0,
		FootLift:       0,
		HipHeight:      0.96,
		Lean:           0.02,
		ArmSwing:       0.03,
		ElbowBend:      0.10,
		Bob:            0.006,
	}
}

func WalkClip() Clip {
	const speed = 1.5
	const stance = 0.62
	const cadence = 0.88
	return Clip{
		ID:             "walk",
		Label:          "行走",
		Speed:          speed,
		Cadence:        cadence,
		NativeFrames:   28,
		StanceFraction: stance,
		Reach:          speed * stance / (2 * cadence),
		FootLift:       0.16,
		HipHeight:      0.94,
		Lean:           0.08,
		ArmSwing:       0.48,
		ElbowBend:      0.18,
		Bob:            0.025,
	}
}

func RunClip() Clip {
	const speed = 4.2
	const stance = 0.40
	const cadence = 1.38
	return Clip{
		ID:             "run",
		Label:          "跑步",
		Speed:          speed,
		Cadence:        cadence,
		NativeFrames:   18,
		StanceFraction: stance,
		Reach:          speed * stance / (2 * cadence),
		FootLift:       0.30,
		HipHeight:      0.90,
		Lean:           0.20,
		ArmSwing:       0.90,
		ElbowBend:      0.72,
		Bob:            0.055,
	}
}

type MotionParams struct {
	Speed          float64
	Cadence        float64
	StanceFraction float64
	Reach          float64
	FootLift       float64
	HipHeight      float64
	Lean           float64
	ArmSwing       float64
	ElbowBend      float64
	Bob            float64
}

func BlendMotion(space *BlendSpace, speed float64, weights []Weight) MotionParams {
	param := MotionParams{Speed: speed}
	for _, weight := range weights {
		clip, ok := space.Clip(weight.ID)
		if !ok {
			continue
		}
		w := weight.Value
		param.Cadence += w * clip.Cadence
		param.StanceFraction += w * clip.StanceFraction
		param.Reach += w * clip.Reach
		param.FootLift += w * clip.FootLift
		param.HipHeight += w * clip.HipHeight
		param.Lean += w * clip.Lean
		param.ArmSwing += w * clip.ArmSwing
		param.ElbowBend += w * clip.ElbowBend
		param.Bob += w * clip.Bob
	}
	param.StanceFraction = clamp(param.StanceFraction, 0.05, 1)
	return param
}
