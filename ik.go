package main

import "math"

// ReachEnvelope returns the working hip->ankle distance. It is deliberately
// slightly below full extension so a stance leg keeps a small flexion instead
// of locking into a mathematically straight 180 degree knee.
func ReachEnvelope(upper, lower float64) float64 { return (upper + lower) * 0.985 }

// FullReach is the true maximum span; the analytic IK keeps both bone lengths
// exact up to this geometric limit.
func FullReach(upper, lower float64) float64 { return upper + lower }

// ClampAnkleToReach pulls an ankle target onto the reach sphere of a hip if
// it is too far away, preserving its bearing. Footprint points of planted
// feet are never clamped; only swing targets are, so locking stays exact.
func ClampAnkleToReach(hip, ankle Vec3, upper, lower float64) Vec3 {
	maxReach := ReachEnvelope(upper, lower) * (1 - 1e-4)
	delta := ankle.Sub(hip)
	dist := delta.Len()
	if dist <= maxReach {
		return ankle
	}
	return hip.Add(delta.Scale(maxReach / dist))
}

// TwoBoneIK positions a hip->knee->ankle chain so the ankle reaches target
// while preserving the two bone lengths. bendHint defines which way the knee
// should flex (it is projected onto the plane perpendicular to hip->ankle).
// It returns the knee world position.
func TwoBoneIK(hip, ankleTarget Vec3, upper, lower float64, bendHint Vec3) Vec3 {
	toAnkle := ankleTarget.Sub(hip)
	dist := toAnkle.Len()
	maxReach := FullReach(upper, lower)

	// Degenerate/over-reach: pull the target back onto the reachable sphere so
	// the chain keeps valid lengths instead of producing NaN angles.
	if dist < 1e-9 {
		return hip.Add(bendHint.Normalized().Scale(upper))
	}
	dir := toAnkle.Scale(1 / dist)
	if dist > maxReach {
		dist = maxReach
	}

	// Projection of the upper bone onto the hip->ankle axis (law of cosines),
	// then the perpendicular component. Both satisfy the two exact lengths.
	along := Clamp((upper*upper+dist*dist-lower*lower)/(2*dist),
		-upper, upper)
	perpendicular := math.Sqrt(math.Max(0, upper*upper-along*along))

	// Orthogonal bend direction inside the limb plane.
	side := bendHint.Sub(dir.Scale(bendHint.Dot(dir)))
	if side.Len() < 1e-9 {
		side = v3(1, 0, 0).Sub(dir.Scale(dir.X))
	}
	side = side.Normalized()

	// Analytic knee: along the hip->ankle axis by upper*cosHip and
	// perpendicular by upper*sinHip. Both bone lengths are exact by
	// construction, including the clamped overreach case.
	return hip.Add(dir.Scale(along)).Add(side.Scale(perpendicular))
}
