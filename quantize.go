package main

import "math"

// scalarRange describes the global min/max envelope of one component.
type scalarRange struct {
	Min, Max float64
}

func encodeScalar(r scalarRange, v float64, bits uint) uint64 {
	levels := float64(int(1)<<bits - 1)
	if r.Max <= r.Min {
		return 0
	}
	u := (v - r.Min) / (r.Max - r.Min)
	u = clamp(u, 0, 1)
	return uint64(math.Round(u * levels))
}

func decodeScalar(r scalarRange, code uint64, bits uint) float64 {
	levels := float64(int(1)<<bits - 1)
	if r.Max <= r.Min {
		return r.Min
	}
	return r.Min + float64(code)/levels*(r.Max-r.Min)
}

// positionRanges collects one envelope per joint-component (17*3 values).
func positionRanges(clip *Clip) [][3]scalarRange {
	ranges := make([][3]scalarRange, JointCount)
	for j := 0; j < JointCount; j++ {
		for c := 0; c < 3; c++ {
			ranges[j][c] = scalarRange{Min: math.Inf(1), Max: math.Inf(-1)}
		}
	}
	for _, f := range clip.Frames {
		for j := 0; j < JointCount; j++ {
			v := f.Pose.Positions[j]
			comp := [3]float64{v.X, v.Y, v.Z}
			for c := 0; c < 3; c++ {
				if comp[c] < ranges[j][c].Min {
					ranges[j][c].Min = comp[c]
				}
				if comp[c] > ranges[j][c].Max {
					ranges[j][c].Max = comp[c]
				}
			}
		}
	}
	return ranges
}

func encodePosition(ranges [][3]scalarRange, joint int, v Vec3, bits uint) [3]uint64 {
	comp := [3]float64{v.X, v.Y, v.Z}
	var codes [3]uint64
	for c := 0; c < 3; c++ {
		codes[c] = encodeScalar(ranges[joint][c], comp[c], bits)
	}
	return codes
}

func decodePosition(ranges [][3]scalarRange, joint int, codes [3]uint64, bits uint) Vec3 {
	return Vec3{
		X: decodeScalar(ranges[joint][0], codes[0], bits),
		Y: decodeScalar(ranges[joint][1], codes[1], bits),
		Z: decodeScalar(ranges[joint][2], codes[2], bits),
	}
}

// quatIndices maps the two-bit largest-component index to the three
// remaining components in their original order.
var quatIndices = [4][3]int{
	{1, 2, 3}, // largest is W  (index 0)
	{0, 2, 3}, // largest is X  (index 1)
	{0, 1, 3}, // largest is Y  (index 2)
	{0, 1, 2}, // largest is Z  (index 3)
}

func quatComponents(q Quat) [4]float64 {
	return [4]float64{q.W, q.X, q.Y, q.Z}
}

func quatFromComponents(c [4]float64) Quat {
	return Quat{W: c[0], X: c[1], Y: c[2], Z: c[3]}
}

// encodeQuat stores the three non-largest components. The largest is
// reconstructed algebraically and always kept non-negative, which removes
// the sign bit entirely (a standard smallest-three packing).
func encodeQuat(q Quat) (maxIdx uint64, codes [3]uint64, bits uint) {
	c := quatComponents(q.Normalized())
	maxIdx = 0
	for i := 1; i < 4; i++ {
		if math.Abs(c[i]) > math.Abs(c[maxIdx]) {
			maxIdx = uint64(i)
		}
	}
	neg := c[maxIdx] < 0
	r := quatIndices[maxIdx]
	for k := 0; k < 3; k++ {
		v := c[r[k]]
		if neg {
			v = -v
		}
		codes[k] = uint64(math.Round(((v + 1) / 2) * float64(rotCodeMax)))
	}
	return maxIdx, codes, rotBitsPerComp
}

func decodeQuat(maxIdx uint64, codes [3]uint64) Quat {
	var c [4]float64
	r := quatIndices[maxIdx]
	sum := 1.0
	for k := 0; k < 3; k++ {
		v := float64(codes[k])/float64(rotCodeMax)*2 - 1
		c[r[k]] = v
		sum -= v * v
	}
	c[maxIdx] = math.Sqrt(math.Max(0, sum))
	return quatFromComponents(c).Normalized()
}

const (
	rotBitsPerComp = 11
	rotCodeMax     = (1 << rotBitsPerComp) - 1
)
