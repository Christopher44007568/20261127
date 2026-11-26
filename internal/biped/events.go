package biped

import (
	"fmt"
	"sort"
	"strings"
)

type EventKind string

const (
	EventLeftHeel  EventKind = "left_heel_strike"
	EventRightHeel EventKind = "right_heel_strike"
	EventLeftToe   EventKind = "left_toe_off"
	EventRightToe  EventKind = "right_toe_off"
)

func (k EventKind) Label() string {
	switch k {
	case EventLeftHeel:
		return "左脚跟落地"
	case EventRightHeel:
		return "右脚跟落地"
	case EventLeftToe:
		return "左脚尖离地"
	case EventRightToe:
		return "右脚尖离地"
	default:
		return string(k)
	}
}

type EventFrame struct {
	Frame    int       `json:"frame"`
	EndFrame int       `json:"end_frame,omitempty"`
	Kind     EventKind `json:"kind"`
	Source   string    `json:"source"`
}

func (e EventFrame) Start() int {
	return e.Frame
}

func (e EventFrame) End() int {
	if e.EndFrame < e.Frame {
		return e.Frame
	}
	return e.EndFrame
}

type MergedEvent struct {
	ID           string       `json:"id"`
	Kind         EventKind    `json:"kind"`
	Frame        int          `json:"frame"`
	EndFrame     int          `json:"end_frame"`
	SourceIDs    []string     `json:"source_ids"`
	RawFrames    []int        `json:"raw_frames"`
	RawEvents    []EventFrame `json:"-"`
	Multiplicity int          `json:"multiplicity"`
}

func MergeEventFrames(events []EventFrame, proximityFrames int) []MergedEvent {
	if proximityFrames < 0 {
		proximityFrames = 0
	}
	ordered := append([]EventFrame(nil), events...)
	sort.SliceStable(ordered, func(i, j int) bool {
		if ordered[i].Kind != ordered[j].Kind {
			return ordered[i].Kind < ordered[j].Kind
		}
		if ordered[i].Start() != ordered[j].Start() {
			return ordered[i].Start() < ordered[j].Start()
		}
		return ordered[i].Source < ordered[j].Source
	})

	var groups []MergedEvent
	for _, event := range ordered {
		if len(groups) == 0 || groups[len(groups)-1].Kind != event.Kind ||
			event.Start() > groups[len(groups)-1].EndFrame+proximityFrames {
			groups = append(groups, MergedEvent{
				Kind:      event.Kind,
				Frame:     event.Start(),
				EndFrame:  event.End(),
				RawEvents: []EventFrame{event},
				RawFrames: []int{event.Frame},
			})
			continue
		}
		group := &groups[len(groups)-1]
		group.RawEvents = append(group.RawEvents, event)
		group.RawFrames = append(group.RawFrames, event.Frame)
		if event.Start() < group.Frame {
			group.Frame = event.Start()
		}
		if event.End() > group.EndFrame {
			group.EndFrame = event.End()
		}
	}

	for i := range groups {
		sourceSet := make(map[string]struct{})
		frames := make([]int, 0, len(groups[i].RawEvents))
		for _, raw := range groups[i].RawEvents {
			sourceSet[raw.Source] = struct{}{}
			frames = append(frames, raw.Frame)
		}
		sources := make([]string, 0, len(sourceSet))
		for source := range sourceSet {
			if source != "" {
				sources = append(sources, source)
			}
		}
		sort.Strings(sources)
		sort.Ints(frames)
		groups[i].SourceIDs = sources
		groups[i].RawFrames = uniqueSortedInts(frames)
		groups[i].Multiplicity = len(groups[i].RawEvents)
		groups[i].ID = fmt.Sprintf("%s@%06d", groups[i].Kind, groups[i].Frame)
	}

	sort.SliceStable(groups, func(i, j int) bool {
		if groups[i].Frame != groups[j].Frame {
			return groups[i].Frame < groups[j].Frame
		}
		return groups[i].Kind < groups[j].Kind
	})
	for i := range groups {
		groups[i].ID = fmt.Sprintf("%s@%06d", groups[i].Kind, groups[i].Frame)
	}
	return groups
}

func uniqueSortedInts(values []int) []int {
	if len(values) == 0 {
		return values
	}
	out := values[:1]
	for _, value := range values[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}

type EventGate struct {
	active map[EventKind]string
}

func NewEventGate() *EventGate {
	return &EventGate{active: make(map[EventKind]string)}
}

func (g *EventGate) Poll(activeEvents []MergedEvent) []MergedEvent {
	current := make(map[EventKind]MergedEvent)
	for _, event := range activeEvents {
		existing, ok := current[event.Kind]
		if !ok {
			current[event.Kind] = event
			continue
		}
		if event.Frame < existing.Frame {
			existing.Frame = event.Frame
		}
		if event.EndFrame > existing.EndFrame {
			existing.EndFrame = event.EndFrame
		}
		existing.RawEvents = append(existing.RawEvents, event.RawEvents...)
		existing.Multiplicity += event.Multiplicity
		current[event.Kind] = existing
	}

	var started []MergedEvent
	next := make(map[EventKind]string, len(current))
	for kind, event := range current {
		next[kind] = event.ID
		if _, active := g.active[kind]; !active {
			started = append(started, event)
		}
	}
	g.active = next
	sort.Slice(started, func(i, j int) bool {
		if started[i].Frame != started[j].Frame {
			return started[i].Frame < started[j].Frame
		}
		return started[i].Kind < started[j].Kind
	})
	return started
}

func (g *EventGate) ActiveKinds() []string {
	kinds := make([]string, 0, len(g.active))
	for kind := range g.active {
		kinds = append(kinds, string(kind))
	}
	sort.Strings(kinds)
	return kinds
}

func FormatEventList(events []MergedEvent) string {
	parts := make([]string, 0, len(events))
	for _, event := range events {
		parts = append(parts, fmt.Sprintf("%s(f%d,src=%s)", event.Kind, event.Frame, strings.Join(event.SourceIDs, "+")))
	}
	return strings.Join(parts, ", ")
}
