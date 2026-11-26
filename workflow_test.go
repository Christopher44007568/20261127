package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// TestRunOutputs generates every artifact into a temp directory and
// validates that PNGs decode and non-CSV reports exist.
func TestRunOutputs(t *testing.T) {
	dir := t.TempDir()
	if err := run(dir); err != nil {
		t.Fatalf("run: %v", err)
	}
	want := []string{
		"walk_frames_sheet.png",
		"target_end_error.png",
		"joint_angle_curves.png",
		"walk.gif",
		"metrics.csv",
		"reach_sweep.csv",
		"summary.txt",
		"index.html",
	}
	for _, name := range want {
		p := filepath.Join(dir, name)
		st, err := os.Stat(p)
		if err != nil {
			t.Fatalf("missing %s: %v", name, err)
		}
		if st.Size() == 0 {
			t.Fatalf("%s is empty", name)
		}
	}
	for _, name := range []string{"walk_frames_sheet.png", "target_end_error.png", "joint_angle_curves.png"} {
		f, err := os.Open(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := png.DecodeConfig(f)
		f.Close()
		if err != nil {
			t.Fatalf("decode config %s: %v", name, err)
		}
		if cfg.Width < 800 || cfg.Height < 600 {
			t.Fatalf("%s unexpectedly small: %dx%d", name, cfg.Width, cfg.Height)
		}
	}
}
