package ppm

import (
	"os"
	"path/filepath"
	"testing"

	"shadowcache/internal/render"
)

func TestWriteBinaryPPM(t *testing.T) {
	frame := render.NewFrame(3, 2)
	frame.Set(0, 0, 255, 0, 0)
	frame.Set(2, 1, 0, 0, 255)
	path := filepath.Join(t.TempDir(), "tiny.ppm")
	if err := WriteBinary(path, frame); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	header := "P6\n3 2\n255\n"
	if len(data) != len(header)+len(frame.Pix) {
		t.Fatalf("file length = %d, want %d", len(data), len(header)+len(frame.Pix))
	}
	if string(data[:len(header)]) != header {
		t.Fatalf("header = %q, want %q", data[:len(header)], header)
	}
}
