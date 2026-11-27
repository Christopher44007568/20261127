package main

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestRenderScenesToPNG(t *testing.T) {
	dir := t.TempDir()
	scenes := []struct {
		name string
		fn   func(string) error
	}{
		{"prism.png", RenderPrism},
		{"sphere.png", RenderSphere},
		{"plane.png", RenderPlate},
	}

	for _, scene := range scenes {
		path := filepath.Join(dir, scene.name)
		if err := scene.fn(path); err != nil {
			t.Fatalf("%s: %v", scene.name, err)
		}
		f, err := os.Open(path)
		if err != nil {
			t.Fatal(err)
		}
		cfg, err := png.DecodeConfig(f)
		f.Close()
		if err != nil {
			t.Fatalf("decode config %s: %v", scene.name, err)
		}
		if cfg.Width != renderWidth || cfg.Height != renderHeight {
			t.Fatalf("%s dimensions=%dx%d", scene.name, cfg.Width, cfg.Height)
		}
		info, err := os.Stat(path)
		if err != nil || info.Size() < 80_000 {
			t.Fatalf("%s has unexpected file size: %v %v", scene.name, info.Size(), err)
		}
	}
}
