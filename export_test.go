package main

import (
	"image/gif"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEndToEndExport(t *testing.T) {
	dir := t.TempDir()
	scene := demoScene()
	tl := Timeline{Duration: 12, FPS: 4, Shutter: 0.75, ShutterSteps: 6}
	report, err := Export(scene, tl, dir)
	if err != nil {
		t.Fatalf("Export returned error: %v", err)
	}
	if report.Frames != 48 || len(report.FrameReports) != 48 {
		t.Fatalf("frame count mismatch: %d reports", len(report.FrameReports))
	}

	for _, name := range []string{
		"thumbnails/frame_0000.png", "thumbnails/frame_0047.png",
		"offscreen.gif", "path_map.png", "report.json",
	} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Fatalf("expected artifact %s: %v", name, err)
		}
	}

	f, err := os.Open(filepath.Join(dir, "offscreen.gif"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	anim, err := gif.DecodeAll(f)
	if err != nil {
		t.Fatalf("invalid gif: %v", err)
	}
	if len(anim.Image) != 48 {
		t.Fatalf("gif frames = %d, want 48", len(anim.Image))
	}

	if len(report.Collisions) == 0 {
		t.Error("demo scene must report at least one obstacle collision")
	}
	if len(report.OutOfBounds) == 0 {
		t.Error("demo scene must report an out-of-bounds segment")
	}
	if pauses := report.CharacterPauses["hero"]; len(pauses) != 2 {
		t.Errorf("hero pauses = %v, want 2 intervals", pauses)
	}

	// Every frame records the same shared-timeline timestamp for camera and
	// character sampling; shutter ghosts must exist for the exposure window.
	for i, fr := range report.FrameReports {
		if fr.ShutterClose <= fr.ShutterOpen {
			t.Fatalf("frame %d has empty shutter interval", i)
		}
		if len(fr.CharacterPos) != len(scene.Characters) {
			t.Fatalf("frame %d missing character samples", i)
		}
	}
}

func TestNoThirdPartyDependencies(t *testing.T) {
	b, err := os.ReadFile("go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "require") {
		t.Fatalf("go.mod must declare no third-party dependencies:\n%s", b)
	}
}
