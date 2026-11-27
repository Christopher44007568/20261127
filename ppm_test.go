package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPPMRoundTrip(t *testing.T) {
	fb := NewFramebuffer(7, 5)
	for y := 0; y < fb.H; y++ {
		for x := 0; x < fb.W; x++ {
			fb.SetPixel(x, y, rgb(x*30, y*50, x*10+y*20))
		}
	}
	p := filepath.Join(t.TempDir(), "case.ppm")
	if err := WritePPM(p, fb); err != nil {
		t.Fatal(err)
	}

	head, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	if string(head[:3]) != "P6\n" {
		t.Fatalf("missing P6 magic, header starts %q", head[:8])
	}

	got, err := ReadPPM(p)
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(fb) {
		t.Fatal("PPM round trip changed pixels")
	}
}

func TestReadPPMRejectsUnknownMagic(t *testing.T) {
	p := filepath.Join(t.TempDir(), "x.ppm")
	if err := os.WriteFile(p, []byte("P5\n1 1\n255\n\x00"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := ReadPPM(p); err == nil {
		t.Fatal("expected error for unsupported P5 magic")
	}
}
