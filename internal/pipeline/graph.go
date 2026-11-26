package pipeline

import (
	"fmt"
	"strings"
)

const (
	Width  = 480
	Height = 270
)

// PassID identifies one scheduled GPU-style render pass.
type PassID string

const (
	Shadow  PassID = "shadow"
	Scene   PassID = "scene"
	Tonemap PassID = "tonemap"
	UI      PassID = "ui"
)

// PassSpec describes the immutable inputs and output of a pass. Only Requires
// creates scheduling edges; output ownership is used by the diagram labels.
type PassSpec struct {
	ID       PassID
	Requires []PassID
	Reads    string
	Writes   string
}

var passOrder = []PassID{Shadow, Scene, Tonemap, UI}

var specs = map[PassID]PassSpec{
	Shadow:  {ID: Shadow, Writes: "shadow visibility"},
	Scene:   {ID: Scene, Requires: []PassID{Shadow}, Reads: "shadow", Writes: "scene HDR"},
	Tonemap: {ID: Tonemap, Requires: []PassID{Scene}, Reads: "scene HDR", Writes: "scene LDR"},
	UI:      {ID: UI, Writes: "UI RGBA"},
}

// DataEdge is a real producer-consumer edge, not merely a display ordering.
type DataEdge struct {
	From     PassID
	To       PassID
	Resource string
}

var dataEdges = []DataEdge{
	{From: Shadow, To: Scene, Resource: "shadow visibility"},
	{From: Scene, To: Tonemap, Resource: "scene HDR"},
}

func Spec(id PassID) PassSpec {
	return specs[id]
}

func DataEdges() []DataEdge {
	return append([]DataEdge(nil), dataEdges...)
}

func AllPasses() []PassID {
	return append([]PassID(nil), passOrder...)
}

// Wave is a set of passes that may be executed concurrently.
type Wave []PassID

// Schedule is one legal execution order. Waves are barriers only for the
// passes placed in them; the validator rejects missing or premature passes.
type Schedule struct {
	Name        string
	Description string
	Waves       []Wave
}

func Schedules() []Schedule {
	return []Schedule{
		{
			Name:        "ui_before_tonemap",
			Description: "UI runs before tonemap; both consume distinct buffers.",
			Waves:       []Wave{{Shadow}, {Scene}, {UI}, {Tonemap}},
		},
		{
			Name:        "tonemap_before_ui",
			Description: "The independent UI and tonemap passes are swapped.",
			Waves:       []Wave{{Shadow}, {Scene}, {Tonemap}, {UI}},
		},
		{
			Name:        "ui_parallel_tonemap",
			Description: "UI and tonemap, which have no edge between them, run together.",
			Waves:       []Wave{{Shadow}, {Scene}, {UI, Tonemap}},
		},
	}
}

// Validate checks completeness and real data dependencies. A pass in the same
// wave as a producer is rejected, which prevents shadow/scene from overlapping.
func (s Schedule) Validate() error {
	if strings.TrimSpace(s.Name) == "" {
		return fmt.Errorf("schedule has no name")
	}

	completed := make(map[PassID]bool, len(specs))
	seen := make(map[PassID]bool, len(specs))
	for waveIndex, wave := range s.Waves {
		if len(wave) == 0 {
			return fmt.Errorf("%s: wave %d is empty", s.Name, waveIndex)
		}
		inWave := make(map[PassID]bool, len(wave))
		for _, id := range wave {
			spec, ok := specs[id]
			if !ok {
				return fmt.Errorf("%s: unknown pass %q", s.Name, id)
			}
			if seen[id] {
				return fmt.Errorf("%s: pass %q scheduled more than once", s.Name, id)
			}
			if inWave[id] {
				return fmt.Errorf("%s: pass %q repeated in wave %d", s.Name, id, waveIndex)
			}
			inWave[id] = true
			seen[id] = true
			for _, dep := range spec.Requires {
				if !completed[dep] {
					return fmt.Errorf("%s: %q starts before dependency %q completes", s.Name, id, dep)
				}
			}
		}
		for id := range inWave {
			completed[id] = true
		}
	}

	for _, id := range passOrder {
		if !seen[id] {
			return fmt.Errorf("%s: pass %q is missing", s.Name, id)
		}
	}
	return nil
}
