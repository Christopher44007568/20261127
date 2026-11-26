package main

import "math"

// Slope is an infinite inclined line. Along-coordinate s follows Uphill;
// world point = Origin + Uphill*s.
type Slope struct {
	Ang    float64
	Uphill Vec
	Normal Vec
	Origin Vec
}

func NewSlope(angRad float64) Slope {
	return Slope{
		Ang:    angRad,
		Uphill: vec(math.Cos(angRad), math.Sin(angRad)),
		Normal: vec(-math.Sin(angRad), math.Cos(angRad)),
	}
}

func (s Slope) Point(along float64) Vec {
	return s.Origin.add(s.Uphill.mul(along))
}

// GaitConfig contains all physical and timing parameters.
type GaitConfig struct {
	Slope Slope
	Leg   LegSpec

	CycleT float64 // seconds per gait cycle
	Speed  float64 // meters per second along the slope
	Duty   float64 // stance fraction of one cycle
	Lead   float64 // touchdown lead ahead of pelvis, meters

	PelvisH   float64 // pelvis distance above ground along normal
	BobAmp    float64
	AnkleH    float64 // ankle height above sole line along normal
	FootLen   float64
	HeelBack  float64 // ankle projection -> heel distance backward
	SwingLift float64
	ToeUp     float64 // extra swing-phase toe rotation, radians

	FootOffsets [2]float64
	FootNames   [2]string
	FootColors  [3]RGB // normal far, normal near? indexed by foot
}

// Stride is the same-foot ground advance per cycle.
func (c GaitConfig) Stride() float64 { return c.Speed * c.CycleT }

func DefaultConfig() GaitConfig {
	slope := NewSlope(rad(14))
	leg := LegSpec{
		Name:       "leg",
		L1:         0.45,
		L2:         0.45,
		HipLimit:   Limit{Min: rad(-42), Max: rad(78)},
		KneeLimit:  Limit{Min: 0, Max: rad(108)},
		AnkleLimit: Limit{Min: rad(-70), Max: rad(70)},
	}
	return GaitConfig{
		Slope:       slope,
		Leg:         leg,
		CycleT:      1.0,
		Speed:       0.55,
		Duty:        0.62,
		Lead:        0.18,
		PelvisH:     0.97,
		BobAmp:      0.018,
		AnkleH:      0.10,
		FootLen:     0.27,
		HeelBack:    0.075,
		SwingLift:   0.11,
		ToeUp:       rad(16),
		FootOffsets: [2]float64{0.5, 0.0}, // left/far shifted half cycle
		FootNames:   [2]string{"L", "R"},
	}
}

// FootSample is one foot's fully solved state at one instant.
type FootSample struct {
	Index      int
	Name       string
	Phase      float64
	Stance     bool
	PlantS     float64
	CmdAnkle   Vec
	CmdFootAng float64
	SoleAnchor Vec
	Heel       Vec
	Toe        Vec
	Pose       LegPose
}

// Frame is the whole character pose at one instant.
type Frame struct {
	Time    float64
	Cycle   float64
	PelvisS float64
	Pelvis  Vec
	Feet    [2]FootSample
}

// plantS is the frozen along-slope coordinate of cycle k's footfall.
func (c GaitConfig) plantS(offset, k float64) float64 {
	return c.Stride()*(k-offset) + c.Lead
}

// SampleFrame evaluates the analytic, deterministic gait at time t.
//
// During stance the commanded ankle equals Slope.Point(PlantS)+AnkleH*N,
// which is independent of t: the sole is locked to the ground in world
// coordinates for the whole stance interval.
func (c GaitConfig) SampleFrame(t float64) Frame {
	f := Frame{Time: t, Cycle: t / c.CycleT, PelvisS: c.Speed * t}

	// Two rises per cycle, peaking around each foot's mid-stance (p~0.31).
	bob := c.BobAmp * math.Cos(4*math.Pi*(t/c.CycleT-0.31))
	f.Pelvis = c.Slope.Point(f.PelvisS).add(c.Slope.Normal.mul(c.PelvisH + bob))

	for i := 0; i < 2; i++ {
		off := c.FootOffsets[i]
		z := t/c.CycleT + off
		k := math.Floor(z)
		p := z - k

		fs := FootSample{Index: i, Name: c.FootNames[i], Phase: p}

		var ankle Vec
		var footAng float64

		if p < c.Duty {
			// Stance: one frozen plant coordinate, no time dependence.
			fs.Stance = true
			fs.PlantS = c.plantS(off, k)
			ankle = c.Slope.Point(fs.PlantS).add(c.Slope.Normal.mul(c.AnkleH))
			footAng = c.Slope.Ang
		} else {
			// Swing: smooth transfer to the next frozen plant, with lift.
			q := (p - c.Duty) / (1 - c.Duty)
			q = clamp01(q)
			fromS := c.plantS(off, k)
			toS := c.plantS(off, k+1)
			s := fromS + (toS-fromS)*smoothStep(q)
			lift := c.SwingLift * math.Sin(math.Pi*q)
			fs.PlantS = toS
			ankle = c.Slope.Point(s).add(c.Slope.Normal.mul(c.AnkleH + lift))
			footAng = c.Slope.Ang + c.ToeUp*math.Sin(math.Pi*q)
		}

		fs.CmdAnkle = ankle
		fs.CmdFootAng = footAng

		args := SolveArgs{
			Hip:       f.Pelvis,
			Target:    ankle,
			Pole:      c.Slope.Uphill,
			PelvisAng: c.Slope.Ang,
			FootAng:   footAng,
		}
		fs.Pose = c.Leg.Solve(args)

		// Build the sole from the solved end effector and enforced foot angle.
		footDir := vec(math.Cos(fs.Pose.FootAng), math.Sin(fs.Pose.FootAng))
		footUp := footDir.rotate(math.Pi / 2)
		fs.SoleAnchor = fs.Pose.Ankle.sub(footUp.mul(c.AnkleH))
		fs.Heel = fs.SoleAnchor.sub(footDir.mul(c.HeelBack))
		fs.Toe = fs.SoleAnchor.add(footDir.mul(c.FootLen - c.HeelBack))

		f.Feet[i] = fs
	}
	return f
}
