package main

import (
	"math"
	"testing"
)

func testLeg() LegSpec { return DefaultConfig().Leg }

// TestIKReachableAccuracy sweeps many feasible targets: an unclamped
// solution must hit the target to machine precision, preserve bone lengths
// and keep the knee on the pole-selected (forward) branch.
func TestIKReachableAccuracy(t *testing.T) {
	leg := testLeg()
	pole := vec(1, 0)
	hits := 0
	for dx := -0.30; dx <= 0.55; dx += 0.025 {
		for dy := -0.88; dy <= -0.30; dy += 0.025 {
			hip := vec(0, 1.0)
			target := hip.add(vec(dx, dy))
			p := leg.Solve(SolveArgs{
				Hip: hip, Target: target, Pole: pole,
				PelvisAng: 0, FootAng: 0,
			})
			if math.Abs(p.Knee.sub(hip).len()-leg.L1) > 1e-9 {
				t.Fatalf("L1 length broken: %v", p.Knee.sub(hip).len())
			}
			if math.Abs(p.Ankle.sub(p.Knee).len()-leg.L2) > 1e-9 {
				t.Fatalf("L2 length broken: %v", p.Ankle.sub(p.Knee).len())
			}
			// Knee must stay on the forward side of H->T.
			ht := target.sub(hip)
			hk := p.Knee.sub(hip)
			if ht.cross(hk) < -1e-9 {
				t.Fatalf("knee on wrong branch: cross=%.6f", ht.cross(hk))
			}
			if p.Status != StatusOK {
				continue
			}
			hits++
			if p.EndError > 1e-9 {
				t.Fatalf("reachable target error %.3e at dx=%.2f dy=%.2f", p.EndError, dx, dy)
			}
		}
	}
	if hits < 50 {
		t.Fatalf("too few clean reachable samples: %d", hits)
	}
}

// TestOverReachStraightAndFlagged verifies the required over-reach behavior:
// status flag, zero knee flexion, endpoint on the reach sphere and a
// residual error equal to D-R.
func TestOverReachStraightAndFlagged(t *testing.T) {
	leg := testLeg()
	hip := vec(0, 1)
	target := hip.add(vec(1.05, -0.55))
	d := target.sub(hip).len()
	p := leg.Solve(SolveArgs{
		Hip: hip, Target: target, Pole: vec(1, 0),
		PelvisAng: 0, FootAng: 0,
	})
	if p.Status != StatusOverReach {
		t.Fatalf("expected StatusOverReach, got %v", p.Status)
	}
	if math.Abs(p.KneeFlex) > 1e-7 {
		t.Fatalf("over-reach chain not straight: kneeFlex=%.6f", p.KneeFlex)
	}
	if got := p.Ankle.sub(hip).len(); math.Abs(got-leg.Reach()) > 1e-7 {
		t.Fatalf("endpoint radius %.6f != R %.6f", got, leg.Reach())
	}
	wantErr := d - leg.Reach()
	if math.Abs(p.EndError-wantErr) > 1e-6 {
		t.Fatalf("error %.6f, want D-R=%.6f", p.EndError, wantErr)
	}
}

// TestTooCloseClampsKneeLimit verifies the max-flexion distance floor.
func TestTooCloseClampsKneeLimit(t *testing.T) {
	leg := testLeg()
	hip := vec(0, 1)
	target := hip.add(vec(0.03, -0.16))
	p := leg.Solve(SolveArgs{
		Hip: hip, Target: target, Pole: vec(1, 0),
		PelvisAng: 0, FootAng: 0,
	})
	if p.Status != StatusTooClose {
		t.Fatalf("expected StatusTooClose, got %v", p.Status)
	}
	if math.Abs(p.KneeFlex-leg.KneeLimit.Max) > 1e-6 {
		t.Fatalf("knee %.4f != max limit %.4f", p.KneeFlex, leg.KneeLimit.Max)
	}
}

// TestNeverHyperextends runs the full gait plus the reach sweep and checks
// every joint against its limits, especially knee flexion >= 0.
func TestNeverHyperextends(t *testing.T) {
	cfg := DefaultConfig()
	for i := 0; i <= 2000; i++ {
		tm := float64(i) / 2000 * 5 * cfg.CycleT
		f := cfg.SampleFrame(tm)
		for _, fs := range f.Feet {
			p := fs.Pose
			if p.KneeFlex < -1e-9 {
				t.Fatalf("hyperextension at t=%.4f %s: %.6f rad", tm, fs.Name, p.KneeFlex)
			}
			if !cfg.Leg.KneeLimit.contains(p.KneeFlex) {
				t.Fatalf("knee out of limit: %.4f", radToDeg(p.KneeFlex))
			}
			if !cfg.Leg.HipLimit.contains(p.HipFlex) {
				t.Fatalf("hip out of limit: %.4f", radToDeg(p.HipFlex))
			}
			if !cfg.Leg.AnkleLimit.contains(p.AnkleDev) {
				t.Fatalf("ankle out of limit: %.4f", radToDeg(p.AnkleDev))
			}
			if p.Status == StatusOK && p.EndError > 1e-9 {
				t.Fatalf("walking IK error too large: %.3e", p.EndError)
			}
		}
	}
	for _, sp := range ReachSweep(cfg) {
		if sp.KneeDeg < -1e-9 {
			t.Fatalf("sweep hyperextension at d=%.3f: %.3f", sp.D, sp.KneeDeg)
		}
	}
}

// TestSoleLockFrameToFrame checks that consecutive stance samples with the
// same plant index produce bit-stable sole coordinates.
func TestSoleLockFrameToFrame(t *testing.T) {
	cfg := DefaultConfig()
	dt := cfg.CycleT / 240
	var prev [2]*FootSample
	maxDrift := 0.0
	for i := 0; i <= 480; i++ {
		f := cfg.SampleFrame(float64(i) * dt)
		for j := range f.Feet {
			fs := f.Feet[j]
			if prev[j] != nil && fs.Stance && prev[j].Stance && fs.PlantS == prev[j].PlantS {
				d := fs.Heel.sub(prev[j].Heel).len()
				maxDrift = math.Max(maxDrift, d)
				d2 := fs.Toe.sub(prev[j].Toe).len()
				maxDrift = math.Max(maxDrift, d2)
			}
			fsCopy := fs
			prev[j] = &fsCopy
		}
	}
	if maxDrift > 1e-12 {
		t.Fatalf("sole moved %.3e m during stance", maxDrift)
	}
}
