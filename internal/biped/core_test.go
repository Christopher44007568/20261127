package biped

import (
	"math"
	"testing"
)

func TestTargetWeightsAreNormalized(t *testing.T) {
	space := NewDefaultBlendSpace()
	for _, speed := range []float64{0, 0.4, 1.5, 2.2, 4.2, 8} {
		weights := space.TargetWeights(speed)
		if got := WeightSum(weights); math.Abs(got-1) > 1e-12 {
			t.Fatalf("speed %.1f weight sum = %.12f, want 1", speed, got)
		}
		for _, weight := range weights {
			if weight.Value < 0 || weight.Value > 1 {
				t.Fatalf("speed %.1f invalid weight %+v", speed, weight)
			}
		}
	}

	if w := WeightValue(space.TargetWeights(0), "idle"); w != 1 {
		t.Fatalf("idle weight at zero = %.3f, want 1", w)
	}
	if w := WeightValue(space.TargetWeights(1.5), "walk"); w != 1 {
		t.Fatalf("walk weight at knot = %.3f, want 1", w)
	}
	if w := WeightValue(space.TargetWeights(4.2), "run"); w != 1 {
		t.Fatalf("run weight at knot = %.3f, want 1", w)
	}
}

func TestNormalizeWeightsRepairsExternalBlend(t *testing.T) {
	weights := NormalizeWeights([]Weight{
		{ID: "idle", Value: 0.4},
		{ID: "walk", Value: 1.2},
		{ID: "run", Value: 0.4},
	})
	if sum := WeightSum(weights); math.Abs(sum-1) > 1e-12 {
		t.Fatalf("normalized sum = %.12f", sum)
	}
	if math.Abs(WeightValue(weights, "walk")-0.6) > 1e-12 {
		t.Fatalf("walk = %.3f, want 0.6", WeightValue(weights, "walk"))
	}

	fallback := NormalizeWeights([]Weight{{ID: "walk", Value: 0}})
	if got := WeightValue(fallback, "walk"); got != 1 {
		t.Fatalf("zero blend fallback = %.3f, want 1", got)
	}
}

func TestMergeOverlappingEventFramesOnce(t *testing.T) {
	raw := []EventFrame{
		{Frame: 10, Kind: EventLeftHeel, Source: "walk"},
		{Frame: 10, EndFrame: 12, Kind: EventLeftHeel, Source: "run"},
		{Frame: 11, Kind: EventLeftHeel, Source: "run-quantized"},
		{Frame: 34, Kind: EventLeftHeel, Source: "walk"},
		{Frame: 35, Kind: EventLeftHeel, Source: "run"},
	}
	merged := MergeEventFrames(raw, 1)
	if len(merged) != 2 {
		t.Fatalf("merged count = %d, want 2: %+v", len(merged), merged)
	}
	if merged[0].Frame != 10 || merged[0].EndFrame != 12 {
		t.Fatalf("first span = %d..%d, want 10..12", merged[0].Frame, merged[0].EndFrame)
	}
	if merged[0].Multiplicity != 3 {
		t.Fatalf("first multiplicity = %d, want 3", merged[0].Multiplicity)
	}
	if len(merged[0].SourceIDs) != 3 {
		t.Fatalf("sources = %v, want three unique sources", merged[0].SourceIDs)
	}
	if merged[1].Frame != 34 {
		t.Fatalf("second event frame = %d, want 34", merged[1].Frame)
	}
}

func TestEventGateEdgeTriggersOverlap(t *testing.T) {
	gate := NewEventGate()
	event := MergedEvent{ID: "left_heel_strike@000010", Kind: EventLeftHeel, Frame: 10, EndFrame: 12}

	first := gate.Poll([]MergedEvent{event})
	second := gate.Poll([]MergedEvent{event})
	if len(first) != 1 {
		t.Fatalf("first poll triggers %d events, want 1", len(first))
	}
	if len(second) != 0 {
		t.Fatalf("overlap poll triggers %d events, want 0", len(second))
	}

	gate.Poll(nil)
	next := event
	next.ID = "left_heel_strike@000034"
	next.Frame = 34
	third := gate.Poll([]MergedEvent{next})
	if len(third) != 1 {
		t.Fatalf("new step triggers %d events, want 1", len(third))
	}
}

func TestScenarioIsSmoothAndFootLocked(t *testing.T) {
	result := RunScenario(Scenario{Duration: 4.0, FPS: 60, Schedule: func(t float64) float64 {
		if t < 0.5 {
			return 0
		}
		return 4.2
	}})
	if len(result.Frames) != 240 {
		t.Fatalf("frames = %d, want 240", len(result.Frames))
	}
	for _, frame := range result.Frames {
		if math.Abs(frame.WeightSum-1) > 1e-12 {
			t.Fatalf("frame %d weight sum = %.12f", frame.Index, frame.WeightSum)
		}
		if frame.LeftSlip.LockedSpeed > 1e-10 || frame.RightSlip.LockedSpeed > 1e-10 {
			t.Fatalf("frame %d locked slip L=%.6f R=%.6f", frame.Index, frame.LeftSlip.LockedSpeed, frame.RightSlip.LockedSpeed)
		}
		if frame.Index > 1 {
			previous := result.Frames[frame.Index-2]
			if jump := math.Abs(frame.Speed - previous.Speed); jump > 0.4 {
				t.Fatalf("frame %d speed jump = %.3f", frame.Index, jump)
			}
		}
	}
	if result.Summary.MaxNaiveSlipSpeed < 0.5 {
		t.Fatalf("naive max slip = %.3f, expected visible phase-conflict slide", result.Summary.MaxNaiveSlipSpeed)
	}
	if len(result.Events) == 0 || len(result.TriggeredEvents) != len(result.Events) {
		t.Fatalf("events merged=%d triggered=%d, want equal non-zero unique counts", len(result.Events), len(result.TriggeredEvents))
	}
}
