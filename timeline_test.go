package main

import (
	"math"
	"testing"
)

func TestFrameCountAndTimes(t *testing.T) {
	tl := Timeline{Duration: 2, FPS: 4, Shutter: 0.5, ShutterSteps: 3}
	if got := tl.FrameCount(); got != 8 {
		t.Fatalf("FrameCount = %d, want 8", got)
	}
	if got := tl.FrameTime(0); math.Abs(got-0.125) > 1e-9 {
		t.Fatalf("FrameTime(0) = %v, want 0.125", got)
	}
	if got := tl.FrameTime(7); math.Abs(got-1.875) > 1e-9 {
		t.Fatalf("FrameTime(7) = %v, want 1.875", got)
	}
}

func TestShutterIntervalAndTimes(t *testing.T) {
	tl := Timeline{Duration: 2, FPS: 4, Shutter: 0.5, ShutterSteps: 4}
	open, close := tl.ShutterInterval(0)
	// Frame center 0.125, half exposure = (0.5/4)/2 = 0.0625, so the first
	// exposure interval is [0.0625, 0.1875]; shutter times never go negative.
	if math.Abs(open-0.0625) > 1e-12 {
		t.Fatalf("open = %v, want 0.0625", open)
	}
	if math.Abs(close-0.1875) > 1e-9 {
		t.Fatalf("close = %v, want 0.1875", close)
	}
	if open < 0 {
		t.Fatal("shutter open must never be negative")
	}
	times := tl.ShutterTimes(2)
	if len(times) != 4 {
		t.Fatalf("got %d shutter times, want 4", len(times))
	}
	wantOpen, wantClose := tl.ShutterInterval(2)
	if math.Abs(times[0]-wantOpen) > 1e-12 || math.Abs(times[3]-wantClose) > 1e-12 {
		t.Fatalf("shutter times must include endpoints, got %v..%v", times[0], times[3])
	}
	for i := 1; i < len(times); i++ {
		if times[i] <= times[i-1] {
			t.Fatal("shutter times must be strictly increasing")
		}
	}
}
