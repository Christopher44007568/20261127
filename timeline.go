package main

import "math"

// Timeline is the single source of truth shared by the spline camera and by
// character sampling. Both are evaluated at exactly the same seconds value,
// which guarantees they stay on one shared timeline during offscreen export.
type Timeline struct {
	Duration     float64 // total animation length in seconds
	FPS          float64 // export frame rate
	Shutter      float64 // exposure interval as a fraction (0,1] of a frame
	ShutterSteps int     // sub-samples evaluated inside each shutter interval
}

// FrameCount is the number of whole frames that fit in the duration.
func (t Timeline) FrameCount() int {
	return int(math.Floor(t.Duration*t.FPS + 1e-9))
}

// FrameTime is the timestamp at the center of frame i.
func (t Timeline) FrameTime(i int) float64 {
	return (float64(i) + 0.5) / t.FPS
}

// ShutterInterval is the closed [open, close] exposure interval of frame i.
// The interval is centered on the frame time and clamped to the animation.
func (t Timeline) ShutterInterval(i int) (open, close float64) {
	center := t.FrameTime(i)
	half := (t.Shutter / t.FPS) / 2
	open = math.Max(0, center-half)
	close = math.Min(t.Duration, center+half)
	return open, close
}

// ShutterTimes returns the sub-sample timestamps of frame i, always including
// both interval endpoints. A single-step shutter evaluates at the center only.
func (t Timeline) ShutterTimes(i int) []float64 {
	open, close := t.ShutterInterval(i)
	if t.ShutterSteps <= 1 {
		return []float64{(open + close) / 2}
	}
	times := make([]float64, t.ShutterSteps)
	for k := 0; k < t.ShutterSteps; k++ {
		u := float64(k) / float64(t.ShutterSteps-1)
		times[k] = open + (close-open)*u
	}
	return times
}
