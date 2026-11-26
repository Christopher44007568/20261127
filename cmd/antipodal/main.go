// Command antipodal isolates the exact-antipodal repeated-pose case: two
// keyframes describe one orientation but store opposite quaternion signs.
// It prints raw nlerp's midpoint collapse against canonical slerp holding
// perfectly still, without touching the main adversarial scenario.
package main

import (
	"fmt"

	"orientation-interp/interp"
	"orientation-interp/report"
	"orientation-interp/rotmath"
)

func main() {
	pose := rotmath.Euler{Yaw: 0.7}
	keys := []interp.Keyframe{
		{T: 0, Euler: pose, Quat: rotmath.FromEuler(pose)},
		{T: 1, Euler: pose, Quat: rotmath.FromEuler(pose).Neg()},
	}

	nlerp := report.BuildSeries(interp.MethodNlerp, keys, 0.02)
	slerp := report.BuildSeries(interp.MethodSlerp, keys, 0.02)

	fmt.Println("exact repeated pose, second keyframe stored as -q")
	fmt.Printf("raw nlerp : degenerate samples=%d, swept=%.4f rad, min blend norm=%.4f\n",
		nlerp.Degenerate, nlerp.SweptAngle, minNorm(nlerp))
	fmt.Printf("stable slerp: degenerate samples=%d, swept=%.3e rad, max |omega|=%.3e rad/s\n",
		slerp.Degenerate, slerp.SweptAngle, slerp.MaxSpeed)
}

func minNorm(s report.Series) float64 {
	min := s.BlendNorm[0]
	for _, v := range s.BlendNorm {
		if v < min {
			min = v
		}
	}
	return min
}
