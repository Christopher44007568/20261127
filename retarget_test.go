package main

import (
	"math"
	"testing"
)

func TestTwoBoneIKPreservesLengthsAndBend(t *testing.T) {
	hip := v3(0, 0, 1)
	ankle := v3(0.3, 0, 0.1) // within the 0.98 m reach envelope
	knee := TwoBoneIK(hip, ankle, 0.5, 0.48, v3(1, 0, 0))

	if got := knee.Sub(hip).Len(); math.Abs(got-0.5) > 1e-9 {
		t.Fatalf("upper length = %.4f, want 0.5", got)
	}
	if got := ankle.Sub(knee).Len(); math.Abs(got-0.48) > 1e-9 {
		t.Fatalf("lower length = %.4f, want 0.48", got)
	}
	if knee.X <= hip.X {
		t.Fatalf("knee should bend forward (+X), got %+v", knee)
	}
}

func TestTwoBoneIKHandlesOverreach(t *testing.T) {
	hip := v3(0, 0, 1)
	target := v3(5, 0, 1) // far beyond reach
	knee := TwoBoneIK(hip, target, 0.5, 0.48, v3(1, 0, 0))
	if math.IsNaN(knee.X) || math.IsNaN(knee.Y) || math.IsNaN(knee.Z) {
		t.Fatal("overreaching IK produced NaN")
	}
	upper, lower := 0.5, 0.48
	if got := knee.Sub(hip).Len(); math.Abs(got-upper) > 1e-9 {
		t.Fatalf("upper length under overreach = %.4f", got)
	}
	// The reachable ankle sits on the max-reach sphere along the target dir.
	reach := hip.Add(target.Sub(hip).Normalized().Scale(FullReach(upper, lower)))
	if got := reach.Sub(knee).Len(); math.Abs(got-lower) > 1e-9 {
		t.Fatalf("lower length under overreach = %.4f", got)
	}
}

func TestPlantedSolesDoNotSlide(t *testing.T) {
	src, dst := NewSourceSkeleton(), NewTargetSkeleton()
	source := GenerateSourceMotion(src)
	target := RetargetMotion(src, dst, source)

	for side := Left; side <= Right; side++ {
		idx := int(side)
		var locked Vec3
		locking := false
		for f := range source {
			if target[f].Contact[idx] {
				if !locking {
					locked = target[f].Sole[idx]
					locking = true
				} else if d := target[f].Sole[idx].Sub(locked).Len(); d > 1e-9 {
					t.Fatalf("side %d frame %d sole slid by %.6f m", idx, f, d)
				}
			} else {
				locking = false
			}
		}
	}
}

func TestGaitAlternatesContacts(t *testing.T) {
	src := NewSourceSkeleton()
	poses := GenerateSourceMotion(src)
	sawSwing := [2]bool{}
	for f := range poses {
		for side := Left; side <= Right; side++ {
			if !poses[f].Contact[int(side)] {
				sawSwing[int(side)] = true
			}
		}
	}
	if !sawSwing[0] || !sawSwing[1] {
		t.Fatalf("both legs must enter swing phase, got %+v", sawSwing)
	}
	// No flight phase: at least one foot is planted on every frame.
	for f, p := range poses {
		if !p.Contact[0] && !p.Contact[1] {
			t.Fatalf("frame %d has no planted foot", f)
		}
	}
}

func TestScaleCompensation(t *testing.T) {
	src, dst := NewSourceSkeleton(), NewTargetSkeleton()
	if dst.legLength() <= src.legLength() {
		t.Fatal("target skeleton should have longer legs")
	}
	source := GenerateSourceMotion(src)
	target := RetargetMotion(src, dst, source)

	last := len(source) - 1
	got := target[last].Root.X / source[last].Root.X
	want := dst.legLength() / src.legLength()
	if math.Abs(got-want) > 1e-9 {
		t.Fatalf("root path scale = %.4f, want %.4f", got, want)
	}

	// Every retargeted bone must keep the target skeleton's own length.
	for f := range target {
		for j := 1; j < JointCount; j++ {
			if j == JPelvis {
				continue // pelvis height carries the gait bob, not a rigid bone
			}
			parent := dst.Parent[j]
			got := target[f].Pos[j].Sub(target[f].Pos[parent]).Len()
			if math.Abs(got-dst.Length[j]) > 1e-6 {
				t.Fatalf("frame %d joint %d length %.4f != target %.4f",
					f, j, got, dst.Length[j])
			}
		}
	}
}

func TestNoHipJumpDuringTurn(t *testing.T) {
	src, dst := NewSourceSkeleton(), NewTargetSkeleton()
	target := RetargetMotion(src, dst, GenerateSourceMotion(src))

	var normal, turnMax float64
	for f := 1; f < FrameCount; f++ {
		jump := target[f].Pos[JPelvis].Sub(target[f-1].Pos[JPelvis]).Len()
		if f >= TurnStartF && f <= TurnEndF {
			if jump > turnMax {
				turnMax = jump
			}
		} else if jump > normal {
			normal = jump
		}
	}
	// Fast turning must not introduce a discontinuity larger than 3x the
	// ordinary per-frame pelvis travel, and must stay well under one stride.
	if turnMax > normal*3+1e-9 || turnMax > 0.25 {
		t.Fatalf("turn hip jump %.4f exceeds normal %.4f", turnMax, normal)
	}
}

func TestCentreOfPressureMoves(t *testing.T) {
	src, dst := NewSourceSkeleton(), NewTargetSkeleton()
	target := RetargetMotion(src, dst, GenerateSourceMotion(src))

	minX, maxX := math.Inf(1), math.Inf(-1)
	minY, maxY := minX, math.Inf(-1)
	for _, p := range target {
		minX = math.Min(minX, p.Cop.X)
		maxX = math.Max(maxX, p.Cop.X)
		minY = math.Min(minY, p.Cop.Y)
		maxY = math.Max(maxY, p.Cop.Y)
	}
	if maxX-minX < 0.5 {
		t.Fatalf("CoP barely travels in X: range %.3f", maxX-minX)
	}
	if maxY-minY < 0.05 {
		t.Fatalf("CoP barely shifts laterally: range %.3f", maxY-minY)
	}
}

func TestJointMapCoversEveryJoint(t *testing.T) {
	mapping := JointMap()
	seen := map[int]bool{}
	for src, dst := range mapping {
		if src != dst {
			t.Fatalf("mapping mismatch at %d -> %d", src, dst)
		}
		seen[dst] = true
	}
	if len(seen) != JointCount {
		t.Fatalf("mapping covers %d joints, want %d", len(seen), JointCount)
	}
}
