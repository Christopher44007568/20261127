package main

import (
	"encoding/json"
	"image"
	"image/png"
	"log"
	"os"
	"path/filepath"
)

const outDir = "artifacts"

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	bundles := make([]bundle, 0, 4)
	for _, spec := range defaultSpecimens() {
		b := makeBundle(spec, 12)
		bundles = append(bundles, b)
		if err := writeBundleAssets(b); err != nil {
			return err
		}
	}
	montage, err := buildMontage(bundles)
	if err != nil {
		return err
	}
	if err := writePNG(filepath.Join(outDir, "sdf-msdf-montage.png"), montage); err != nil {
		return err
	}
	zoomSheet, err := buildZoomSheet()
	if err != nil {
		return err
	}
	if err := writePNG(filepath.Join(outDir, "zoom-levels.png"), zoomSheet); err != nil {
		return err
	}
	metrics := buildMetrics(bundles)
	data, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "metrics.json"), data, 0o644)
}

func writePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	return enc.Encode(f, img)
}
