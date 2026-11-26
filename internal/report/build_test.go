package report

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildAllScenes(t *testing.T) {
	outDir := t.TempDir()
	page, err := Build(outDir)
	if err != nil {
		t.Fatal(err)
	}
	if !page.Summary.AllPass {
		t.Fatalf("validation failed: min IoU = %.9f", page.Summary.MinIOU)
	}
	if page.Summary.Operations != 9 || page.Summary.TotalOBJFiles != 9 {
		t.Fatalf("operations=%d objFiles=%d, want 9/9",
			page.Summary.Operations, page.Summary.TotalOBJFiles)
	}
	if page.Summary.MinIOU != 1 {
		t.Fatalf("min silhouette IoU = %.12f, want 1", page.Summary.MinIOU)
	}
	if _, err := os.Stat(filepath.Join(outDir, "index.html")); err != nil {
		t.Fatal(err)
	}
}
