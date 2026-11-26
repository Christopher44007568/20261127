package pipeline

import (
	"bytes"
	"crypto/sha256"
	"reflect"
	"strings"
	"testing"
)

func TestAllSchedulesAreLegal(t *testing.T) {
	schedules := Schedules()
	if len(schedules) != 3 {
		t.Fatalf("got %d schedules, want 3", len(schedules))
	}
	for _, schedule := range schedules {
		if err := schedule.Validate(); err != nil {
			t.Errorf("%s: %v", schedule.Name, err)
		}
	}
}

func TestScheduleRejectsShadowSceneParallel(t *testing.T) {
	schedule := Schedule{
		Name:  "illegal_parallel_shadow_scene",
		Waves: []Wave{{Shadow, Scene}, {Tonemap}, {UI}},
	}
	err := schedule.Validate()
	if err == nil {
		t.Fatal("expected shadow/scene parallel schedule to fail")
	}
	if !strings.Contains(err.Error(), "before dependency") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestScheduleRejectsMissingAndDuplicatePasses(t *testing.T) {
	missing := Schedule{Name: "missing_ui", Waves: []Wave{{Shadow}, {Scene}, {Tonemap}}}
	if err := missing.Validate(); err == nil {
		t.Fatal("expected missing UI pass to fail")
	}
	duplicate := Schedule{Name: "duplicate_ui", Waves: []Wave{{Shadow}, {Scene}, {Tonemap}, {UI}, {UI}}}
	if err := duplicate.Validate(); err == nil {
		t.Fatal("expected duplicate UI pass to fail")
	}
}

func TestDataEdgesMatchRequirementsExactly(t *testing.T) {
	edges := DataEdges()
	if len(edges) != 2 {
		t.Fatalf("got %d data edges, want 2", len(edges))
	}

	edgeSet := make(map[[2]PassID]bool, len(edges))
	for _, edge := range edges {
		if edge.From == UI || edge.To == UI {
			t.Fatalf("UI must have no data edge, got %s -> %s", edge.From, edge.To)
		}
		edgeSet[[2]PassID{edge.From, edge.To}] = true
	}
	for id, spec := range specs {
		for _, dependency := range spec.Requires {
			key := [2]PassID{dependency, id}
			if !edgeSet[key] {
				t.Errorf("missing data edge %s -> %s", dependency, id)
			}
		}
	}
}

func TestSchedulesProduceIdenticalFramesAndBuffers(t *testing.T) {
	var reference *Outputs
	var referencePPM []byte

	for _, schedule := range Schedules() {
		outputs, err := Execute(schedule)
		if err != nil {
			t.Fatalf("%s: %v", schedule.Name, err)
		}
		frame, err := Composite(outputs)
		if err != nil {
			t.Fatalf("%s: %v", schedule.Name, err)
		}
		var ppm bytes.Buffer
		if err := WritePPM(&ppm, frame); err != nil {
			t.Fatalf("%s: %v", schedule.Name, err)
		}

		if reference == nil {
			reference = outputs
			referencePPM = ppm.Bytes()
			continue
		}
		if !bytes.Equal(referencePPM, ppm.Bytes()) {
			t.Fatalf("%s produced a different PPM (sha256 %x vs %x)", schedule.Name, sha256.Sum256(referencePPM), sha256.Sum256(ppm.Bytes()))
		}
		if !reflect.DeepEqual(reference.Shadow, outputs.Shadow) {
			t.Fatalf("%s produced a different shadow buffer", schedule.Name)
		}
		if !reflect.DeepEqual(reference.Scene, outputs.Scene) {
			t.Fatalf("%s produced a different scene buffer", schedule.Name)
		}
		if !reflect.DeepEqual(reference.Tonemaped, outputs.Tonemaped) {
			t.Fatalf("%s produced a different tonemapped buffer", schedule.Name)
		}
		if !reflect.DeepEqual(reference.UI, outputs.UI) {
			t.Fatalf("%s produced a different UI buffer", schedule.Name)
		}
	}
}

func TestPPMHeaderAndPayloadLength(t *testing.T) {
	outputs, err := Execute(Schedules()[0])
	if err != nil {
		t.Fatal(err)
	}
	frame, err := Composite(outputs)
	if err != nil {
		t.Fatal(err)
	}
	var ppm bytes.Buffer
	if err := WritePPM(&ppm, frame); err != nil {
		t.Fatal(err)
	}
	header := "P6\n480 270\n255\n"
	if !bytes.HasPrefix(ppm.Bytes(), []byte(header)) {
		t.Fatalf("bad PPM header: %q", ppm.Bytes()[:len(header)])
	}
	if got, want := ppm.Len(), len(header)+Width*Height*3; got != want {
		t.Fatalf("PPM length %d, want %d", got, want)
	}
}

func TestSVGOnlyDrawsRealDataEdges(t *testing.T) {
	schedule := Schedules()[2]
	var svg bytes.Buffer
	if err := WriteScheduleSVG(&svg, schedule, "test-sha"); err != nil {
		t.Fatal(err)
	}
	markup := svg.String()
	if count := strings.Count(markup, `class="data-edge"`); count != len(DataEdges()) {
		t.Fatalf("SVG has %d data-edge groups, want %d", count, len(DataEdges()))
	}
	if !strings.Contains(markup, `id="node-ui"`) {
		t.Fatal("SVG is missing the isolated UI node")
	}
	if strings.Contains(markup, `data-ui`) {
		t.Fatal("SVG invents a UI data edge")
	}
}
