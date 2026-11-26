package app_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"coverlab/app"
)

func TestRunGeneratesArtifactsAndChecks(t *testing.T) {
	out := t.TempDir()
	if err := app.Run(out); err != nil {
		t.Fatalf("Run: %v", err)
	}

	b, err := os.ReadFile(filepath.Join(out, "summary.json"))
	if err != nil {
		t.Fatal(err)
	}
	var sum app.Summary
	if err := json.Unmarshal(b, &sum); err != nil {
		t.Fatal(err)
	}
	for name, ok := range sum.Checks {
		if !ok {
			t.Errorf("check %s = false", name)
		}
	}
	if len(sum.Scenes) != 5 {
		t.Fatalf("got %d scenes, want 5", len(sum.Scenes))
	}

	required := []string{
		"coverage/diagonal__analytic.png",
		"coverage/diagonal__ss2.png",
		"coverage/diagonal__ss3.png",
		"closeup/cross__analytic.png",
		"heatmap/cross__ss2_error.png",
		"heatmap/cross__ss3_error.png",
		"sweep/continuity_chart.png",
		"sweep/continuity_strip.png",
		"sweep/sweep.csv",
		"stain/segregate__red_analytic.png",
		"stain/segregate__red_blur.png",
	}
	for _, rel := range required {
		info, err := os.Stat(filepath.Join(out, rel))
		if err != nil {
			t.Errorf("missing artifact %s: %v", rel, err)
			continue
		}
		if info.Size() == 0 {
			t.Errorf("empty artifact %s", rel)
		}
	}
}
