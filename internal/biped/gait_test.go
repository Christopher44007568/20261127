package biped

import "testing"

func TestGaitSegmentsAreMonotonic(t *testing.T) {
	for _, clip := range []Clip{WalkClip(), RunClip()} {
		params := ClipMotionParams(clip)
		previous := GaitFootTarget(0, Left, params)
		for i := 1; i <= 100; i++ {
			q := params.StanceFraction * float64(i) / 100
			current := GaitFootTarget(q, Left, params)
			if current.RelX > previous.RelX+1e-12 {
				t.Fatalf("%s stance reversed at %.3f: %.4f -> %.4f", clip.ID, q, previous.RelX, current.RelX)
			}
			previous = current
		}

		var swingPrevious float64
		for i := 0; i <= 100; i++ {
			u := float64(i) / 100
			q := params.StanceFraction + u*(1-params.StanceFraction)
			current := GaitFootTarget(q, Left, params)
			if i > 0 && current.RelX < swingPrevious-1e-12 {
				t.Fatalf("%s swing reversed at %.3f: %.4f -> %.4f", clip.ID, q, swingPrevious, current.RelX)
			}
			swingPrevious = current.RelX
		}
	}
}

func TestTriggeredEventIDsAreUnique(t *testing.T) {
	result := RunScenario(Scenario{Duration: 5})
	seen := make(map[string]int)
	for _, event := range result.TriggeredEvents {
		seen[event.ID]++
		if seen[event.ID] > 1 {
			t.Fatalf("event %s triggered %d times", event.ID, seen[event.ID])
		}
	}
	if len(seen) != len(result.Events) {
		t.Fatalf("unique triggers %d, merged events %d", len(seen), len(result.Events))
	}
}
