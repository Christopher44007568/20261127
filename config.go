package main

import "path/filepath"

// outputDir is set from the -out flag and anchors every artifact.
var outputDir = "outputs"

func outPath(name string) string {
	return filepath.Join(outputDir, name)
}
