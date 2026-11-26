package main

import "math"

// JointMetrics holds per-joint summary errors over the whole clip.
type JointMetrics struct {
	Name        string
	PosMax      float64
	PosMean     float64
	AngleMaxRad float64
	AngleMean   float64
}

// HeatMatrix is frame x joint position error in metres.
type HeatMatrix struct {
	Values [][]float64
}

// quatAngle is the geodesic angle between two unit quaternions.
func quatAngle(a, b Quat) float64 {
	d := a.Dot(b)
	if d < 0 {
		d = -d
	}
	return 2 * math.Acos(clamp(d, -1, 1))
}

// ComputeMetrics compares world poses and returns summaries plus the full
// heat matrix used by the heat-map image.
func ComputeMetrics(sk *Skeleton, orig, comp *Clip) ([]JointMetrics, HeatMatrix) {
	worldA := sk.WorldClip(orig)
	worldB := sk.WorldClip(comp)
	n := len(worldA)

	metrics := make([]JointMetrics, JointCount)
	heat := HeatMatrix{Values: make([][]float64, n)}
	for j := range metrics {
		metrics[j].Name = sk.Joints[j].Name
	}

	for f := 0; f < n; f++ {
		heat.Values[f] = make([]float64, JointCount)
		for j := 0; j < JointCount; j++ {
			pe := worldA[f].Positions[j].Sub(worldB[f].Positions[j]).Len()
			ae := quatAngle(worldA[f].Rotations[j], worldB[f].Rotations[j])
			metrics[j].PosMean += pe
			metrics[j].AngleMean += ae
			if pe > metrics[j].PosMax {
				metrics[j].PosMax = pe
			}
			if ae > metrics[j].AngleMaxRad {
				metrics[j].AngleMaxRad = ae
			}
			heat.Values[f][j] = pe
		}
	}
	for j := range metrics {
		metrics[j].PosMean /= float64(n)
		metrics[j].AngleMean /= float64(n)
	}
	return metrics, heat
}

// SlipCurve accumulates per-frame ankle drift while a contact is active.
// At every retained contact transition the compressed ankle becomes the
// new anchor; drift between anchors is the codec-induced foot slide. The
// original clip holds an exactly stationary anchor, so its curve is zero.
type SlipCurve struct {
	Left  []float64
	Right []float64
}

func ComputeSlip(sk *Skeleton, clip *Clip) SlipCurve {
	world := sk.WorldClip(clip)
	out := SlipCurve{
		Left:  make([]float64, len(clip.Frames)),
		Right: make([]float64, len(clip.Frames)),
	}
	var anchor [2]Vec3
	active := [2]bool{false, false}

	for f, frame := range clip.Frames {
		for foot, joint := range [2]int{JLAnkle, JRAnkle} {
			if frame.Contacts[foot] {
				if !active[foot] {
					anchor[foot] = world[f].Positions[joint]
					active[foot] = true
				}
				drift := world[f].Positions[joint].Sub(anchor[foot]).Len()
				if foot == 0 {
					out.Left[f] = drift
				} else {
					out.Right[f] = drift
				}
			} else {
				active[foot] = false
				if foot == 0 {
					out.Left[f] = math.NaN()
				} else {
					out.Right[f] = math.NaN()
				}
			}
		}
	}
	return out
}

// VerifyKeyPoses returns the first mismatching key frame index, or -1 when
// every key pose restores to identical float64 bits.
func VerifyKeyPoses(sk *Skeleton, orig, comp *Clip) (int, string) {
	for i := range orig.Frames {
		if !orig.Frames[i].Key {
			continue
		}
		a, b := orig.Frames[i].Pose, comp.Frames[i].Pose
		for j := 0; j < JointCount; j++ {
			av, bv := a.Positions[j], b.Positions[j]
			if av != bv {
				return i, sk.Joints[j].Name
			}
			if a.Rotations[j] != b.Rotations[j] {
				return i, sk.Joints[j].Name
			}
		}
	}
	return -1, ""
}
