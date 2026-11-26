package report

import (
	"bytes"
	"math"
	"strings"
	"testing"

	"orientation-interp/interp"
	"orientation-interp/rotmath"
)

func TestBuildSeriesSlerpMeetsArcBudget(t *testing.T) {
	keys := interp.Scenario()
	all := make([]Series, 0, 3)
	for _, m := range []interp.Method{interp.MethodEuler, interp.MethodNlerp, interp.MethodSlerp} {
		all = append(all, BuildSeries(m, keys, interp.ScenarioStep))
	}
	budget := ArcBudget(keys)
	slerp := all[2]
	if math.Abs(slerp.SweptAngle-budget) > 0.02 {
		t.Fatalf("slerp swept %v, budget %v", slerp.SweptAngle, budget)
	}
	if slerp.Degenerate != 0 {
		t.Fatal("canonical slerp must never be degenerate")
	}
	euler, nlerp := all[0], all[1]
	if euler.SweptAngle <= slerp.SweptAngle {
		t.Fatal("euler lerp should overshoot the shortest arc")
	}
	if nlerp.SweptAngle <= slerp.SweptAngle {
		t.Fatal("raw nlerp should overshoot the shortest arc on flipped signs")
	}
}

func TestAntipodalNlerpIsFlaggedAndSlerpStaysStill(t *testing.T) {
	e := rotmath.Euler{Yaw: 0.7}
	keys := []interp.Keyframe{
		{T: 0, Euler: e, Quat: rotmath.FromEuler(e)},
		{T: 1, Euler: e, Quat: rotmath.FromEuler(e).Neg()},
	}
	nlerp := BuildSeries(interp.MethodNlerp, keys, 0.02)
	if nlerp.Degenerate == 0 {
		t.Fatal("antipodal nlerp midpoint must be flagged degenerate")
	}
	minNorm := math.Inf(1)
	for _, v := range nlerp.BlendNorm {
		minNorm = math.Min(minNorm, v)
	}
	if minNorm > 0.01 {
		t.Fatalf("nlerp blend norm should collapse, got min %v", minNorm)
	}

	slerp := BuildSeries(interp.MethodSlerp, keys, 0.02)
	if slerp.SweptAngle > 1e-6 || slerp.MaxSpeed > 1e-6 {
		t.Fatalf("slerp on a repeated pose must stay still, swept=%v maxSpeed=%v",
			slerp.SweptAngle, slerp.MaxSpeed)
	}
}

func TestAllReportsAreRendered(t *testing.T) {
	keys := interp.Scenario()
	all := []Series{BuildSeries(interp.MethodSlerp, keys, interp.ScenarioStep)}
	var buf bytes.Buffer
	checks := []func() error{
		func() error { buf.Reset(); return WritePoseStrip(&buf, all, KeyframeTimes(keys)) },
		func() error { buf.Reset(); return WriteSpeedChart(&buf, all, KeyframeTimes(keys)) },
		func() error { buf.Reset(); return WriteOrthoChart(&buf, all, KeyframeTimes(keys)) },
		func() error { buf.Reset(); return WriteBlendNormChart(&buf, all, KeyframeTimes(keys)) },
		func() error { buf.Reset(); return WriteCSV(&buf, all) },
	}
	for _, fn := range checks {
		if err := fn(); err != nil {
			t.Fatalf("render report: %v", err)
		}
		if buf.Len() == 0 {
			t.Fatal("report produced empty output")
		}
	}
}

func TestSVGsAreWellFormed(t *testing.T) {
	keys := interp.Scenario()
	all := []Series{BuildSeries(interp.MethodSlerp, keys, interp.ScenarioStep)}
	for _, fn := range []func(*bytes.Buffer) error{
		func(b *bytes.Buffer) error { return WritePoseStrip(b, all, KeyframeTimes(keys)) },
		func(b *bytes.Buffer) error { return WriteSpeedChart(b, all, KeyframeTimes(keys)) },
	} {
		var b bytes.Buffer
		if err := fn(&b); err != nil {
			t.Fatal(err)
		}
		s := b.String()
		if !strings.HasPrefix(s, "<svg") || !strings.HasSuffix(strings.TrimSpace(s), "</svg>") {
			t.Fatal("svg missing wrapper tags")
		}
	}
}
