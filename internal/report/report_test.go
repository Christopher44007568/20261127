package report

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"characterbounds/hierbox"
)

func TestGenerateWritesAllArtifacts(t *testing.T) {
	r, err := hierbox.NewRunner()
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	bundle, err := Generate(r, dir)
	if err != nil {
		t.Fatal(err)
	}
	if !bundle.Poses[0].Coverage.Covered {
		t.Fatal("first pose not covered")
	}

	required := []string{
		"index.html", "summary.json", "result.json", "culling.json",
		"culling.svg", "volume_layers.svg", "volume_layers.csv", "update_trace.csv",
		"overlay_squat.svg", "overlay_push_off.svg", "overlay_tail_swing.svg",
	}
	for _, name := range required {
		path := filepath.Join(dir, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", name)
		}
	}

	data, err := os.ReadFile(filepath.Join(dir, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var summary Summary
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatal(err)
	}
	if !summary.AllVerticesCovered {
		t.Fatalf("summary does not report coverage: %s", data)
	}
	if summary.Incremental.MinBoundsUpdates <= 0 || summary.Incremental.MaxBoundsUpdates > summary.Incremental.NodeCount {
		t.Fatalf("bad incremental summary: %+v", summary.Incremental)
	}

	svg, err := os.ReadFile(filepath.Join(dir, "overlay_tail_swing.svg"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(svg), "Tail swing") {
		t.Fatal("overlay SVG missing pose title")
	}
}
