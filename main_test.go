package main

import (
	"os"
	"path/filepath"
	"testing"

	"fillmask/demo"
)

// Root-level smoke test: the module entry package can drive the whole run and
// every acceptance check in the report passes.
func TestRootRunSmoke(t *testing.T) {
	dir := t.TempDir()
	rep, err := demo.Run(filepath.Join(dir, "artifacts"))
	if err != nil {
		t.Fatalf("demo.Run: %v", err)
	}
	if !rep.OK {
		t.Fatalf("report has failing checks: %+v", rep.Checks)
	}
	if _, err := os.Stat(filepath.Join(dir, "artifacts", "composite.png")); err != nil {
		t.Fatalf("composite missing: %v", err)
	}
}
