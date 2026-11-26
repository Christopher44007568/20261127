package main

import "fmt"

type BugMode uint8

const (
	BugNone BugMode = iota
	BugEarlyRelease
	BugLateReclaim
)

type frameRun struct {
	Image         *Surface
	FailedLayer   bool
	ReusedRetired bool
	Generation    uint64
}

type DiffMetrics struct {
	ChangedPixels int
	BlackPixels   int
	MagentaPixels int
}

type ScenarioResult struct {
	Name     string
	Policy   ReclaimPolicy
	Bug      BugMode
	Frames   []*Surface
	Diffs    []DiffMetrics
	Stats    LifecycleStats
	QueueEnd int
}

func renderManagedFrame(m *Manager, frame int, bug BugMode) frameRun {
	if err := m.BeginFrame(frame); err != nil {
		panic(err)
	}

	panel := NewSurface(panelSize, panelSize)
	paintBackdrop(panel)
	run := frameRun{}

	const targetID = "signal-layer"
	target, err := m.Create(targetID, panelSize, panelSize)
	if err != nil {
		if bug != BugLateReclaim {
			panic(fmt.Errorf("create layer: %w", err))
		}
		target, err = m.ReuseRetired(targetID)
		if err != nil {
			panic(fmt.Errorf("reuse retired layer: %w", err))
		}
		run.ReusedRetired = true
	}
	run.Generation = target.Generation

	producer, err := m.Acquire(targetID, "raster-pass")
	if err != nil {
		panic(fmt.Errorf("raster acquire: %w", err))
	}
	layer, err := m.Surface(producer)
	if err != nil {
		panic(fmt.Errorf("raster surface: %w", err))
	}
	if !run.ReusedRetired {
		layer.Clear()
	}
	drawSignalLayerOnto(layer, frame)

	var consumer *Lease
	if bug == BugEarlyRelease {
		// The faulty graph has no consumer reference. Releasing the producer
		// immediately destroys the texture before the composite pass runs.
		_ = m.Release(producer)
		consumer, err = m.Acquire(targetID, "composite-pass")
	} else {
		// Correct dependency order: the consumer pins the texture before the
		// producer is allowed to retire it.
		consumer, err = m.Acquire(targetID, "composite-pass")
		_ = m.Release(producer)
	}

	if err != nil {
		run.FailedLayer = true
		for _, b := range signalBars(frame) {
			panel.FillRect(b.X, b.Y, b.W, b.H, colBlack)
			panel.StrokeRect(b.X, b.Y, b.W, b.H, 2, colMagenta)
		}
	} else {
		layer.BlendOver(panel)
		_ = m.Release(consumer)
	}

	paintPanelChrome(panel, frame)
	if err := m.EndFrame(); err != nil {
		panic(err)
	}
	run.Image = panel
	return run
}

func renderScenario(name string, policy ReclaimPolicy, bug BugMode) *ScenarioResult {
	const frames = 2
	m := NewManager(policy)
	res := &ScenarioResult{Name: name, Policy: policy, Bug: bug}
	for frame := 0; frame < frames; frame++ {
		run := renderManagedFrame(m, frame, bug)
		ideal := renderIdealFrame(frame)
		res.Frames = append(res.Frames, run.Image)
		res.Diffs = append(res.Diffs, markDifferences(run.Image, ideal))
	}
	// Close the final boundary too, so the report reflects queue draining
	// rather than merely the last frame's in-flight state.
	if err := m.BeginFrame(frames); err != nil {
		panic(err)
	}
	res.Stats = m.Stats
	res.QueueEnd = m.QueueLen()
	return res
}

func renderFreshFrame(frame int) *Surface {
	m := NewManager(PolicyCorrect)
	return renderManagedFrame(m, frame, BugNone).Image
}

func markDifferences(actual, ideal *Surface) DiffMetrics {
	if actual.W != ideal.W || actual.H != ideal.H {
		panic("diff dimensions do not match")
	}
	metric := DiffMetrics{}
	diff := make([]bool, len(actual.Pix))
	var bx0, by0, bx1, by1 int
	bx0, by0 = actual.W, actual.H

	for y := 0; y < actual.H; y++ {
		for x := 0; x < actual.W; x++ {
			i := y*actual.W + x
			a, b := actual.Pix[i], ideal.Pix[i]
			if colorDistance(a, b) <= 6 {
				continue
			}
			diff[i] = true
			metric.ChangedPixels++
			if a.R == 0 && a.G == 0 && a.B == 0 {
				metric.BlackPixels++
			}
			bx0, by0 = minInt(bx0, x), minInt(by0, y)
			bx1, by1 = maxInt(bx1, x), maxInt(by1, y)
		}
	}

	for y := 0; y < actual.H; y++ {
		for x := 0; x < actual.W; x++ {
			i := y*actual.W + x
			if !diff[i] {
				continue
			}
			if (x+y)%11 == 0 || (x-y+actual.H)%37 == 0 {
				actual.Pix[i] = colMagenta
				metric.MagentaPixels++
			}
		}
	}
	if metric.ChangedPixels > 0 {
		drawErrorCorners(actual, bx0, by0, bx1, by1)
	}
	return metric
}

func drawErrorCorners(s *Surface, x0, y0, x1, y1 int) {
	const arm = 14
	for _, p := range [][2]int{
		{x0, y0}, {x1, y0}, {x0, y1}, {x1, y1},
	} {
		for i := 0; i < arm; i++ {
			s.Set(p[0]+i, p[1], colMagenta)
			s.Set(p[0]-i, p[1], colMagenta)
			s.Set(p[0], p[1]+i, colMagenta)
			s.Set(p[0], p[1]-i, colMagenta)
		}
	}
}

func colorDistance(a, b RGBA) int {
	return absInt(int(a.R)-int(b.R)) +
		absInt(int(a.G)-int(b.G)) +
		absInt(int(a.B)-int(b.B))
}
