package demo

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunWritesRequestedEvidence(t *testing.T) {
	outDir := t.TempDir()
	report, err := Run(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if !report.AllChecksPass {
		for _, c := range report.Checks {
			if !c.OK {
				t.Errorf("check failed: %s: %s", c.Name, c.Detail)
			}
		}
		t.Fatal("demo checks did not all pass")
	}

	correct, err := os.ReadFile(filepath.Join(outDir, "correct.ppm"))
	if err != nil {
		t.Fatal(err)
	}
	wrong, err := os.ReadFile(filepath.Join(outDir, "wrong_key.ppm"))
	if err != nil {
		t.Fatal(err)
	}
	rebuilt, err := os.ReadFile(filepath.Join(outDir, "rebuilt.ppm"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(correct, rebuilt) {
		t.Fatal("rebuilt.ppm is not byte-identical to correct.ppm")
	}
	if bytes.Equal(correct, wrong) {
		t.Fatal("wrong_key.ppm unexpectedly equals correct.ppm")
	}

	reportData, err := os.ReadFile(filepath.Join(outDir, "report.json"))
	if err != nil {
		t.Fatal(err)
	}
	var decoded Report
	if err := json.Unmarshal(reportData, &decoded); err != nil {
		t.Fatal(err)
	}
	if !decoded.AllChecksPass {
		t.Fatal("persisted report does not report all checks passing")
	}
	if decoded.FixedCache.Stats.Hits < 1 {
		t.Fatal("persisted report has no fixed-cache hit")
	}
	if decoded.LegacyCache.Stats.Hits < 1 {
		t.Fatal("persisted report has no legacy-cache hit")
	}
}
