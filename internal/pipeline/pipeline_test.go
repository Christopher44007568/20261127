package pipeline

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateWritesAllArtifacts(t *testing.T) {
	out := filepath.Join(t.TempDir(), "material-lod")
	art, err := Generate(Options{Out: out})
	if err != nil {
		t.Fatal(err)
	}

	required := []string{
		"spheres.png",
		"view_sweep.png",
		"brightness_diff.png",
		"boundary_continuity.png",
		"view_sweep.csv",
		"brightness_difference.csv",
		"boundary_errors.csv",
		"brightness_stats.json",
		"continuity_report.json",
		"materials.json",
		"report.md",
	}
	for _, name := range required {
		path := filepath.Join(out, name)
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
		if info.Size() == 0 {
			t.Fatalf("%s is empty", name)
		}
		if filepath.Ext(name) == ".png" {
			f, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := png.DecodeConfig(f); err != nil {
				f.Close()
				t.Fatalf("invalid png %s: %v", name, err)
			}
			if err := f.Close(); err != nil {
				t.Fatal(err)
			}
		}
	}

	if len(art.Materials) != 3 || len(art.Sweeps) != 3*3*91 {
		t.Fatalf("unexpected artifact counts: materials=%d sweeps=%d", len(art.Materials), len(art.Sweeps))
	}
	for _, boundary := range art.Continuity.Boundaries {
		if !boundary.Passed {
			t.Fatalf("boundary %s failed", boundary.Transition)
		}
	}
}
