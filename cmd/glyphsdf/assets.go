package main

import (
	"image"
	"path/filepath"

	"glyphsdf/internal/render"
)

type metricsFile struct {
	Grid      int              `json:"grid"`
	Distance  float64          `json:"distanceRangeTexels"`
	Zoom      float64          `json:"diagnosticZoom"`
	FontPath  string           `json:"fontPath"`
	Specimens []specimenMetric `json:"specimens"`
}

type specimenMetric struct {
	Key    string        `json:"key"`
	Label  string        `json:"label"`
	Report render.Report `json:"report"`
}

func writeBundleAssets(b bundle) error {
	paths := map[string]image.Image{
		b.spec.key + "-sdf-field.png":  b.field.SDFImage(),
		b.spec.key + "-msdf-field.png": b.field.MSDFImage(),
		b.spec.key + "-sdf-break.png":  b.sdfView,
		b.spec.key + "-msdf-sharp.png": b.msdfView,
	}
	if b.spec.key == "a" {
		paths["sdf-field.png"] = b.field.SDFImage()
		paths["msdf-field.png"] = b.field.MSDFImage()
		paths["coverage-sdf.png"] = b.sdfView
		paths["coverage-msdf.png"] = b.msdfView
	}
	for name, img := range paths {
		if err := writePNG(filepath.Join(outDir, name), img); err != nil {
			return err
		}
	}
	return nil
}

func buildMetrics(bundles []bundle) metricsFile {
	m := metricsFile{Grid: heroGrid, Distance: 4.0, Zoom: 12, FontPath: fontPath}
	for _, b := range bundles {
		m.Specimens = append(m.Specimens, specimenMetric{
			Key: b.spec.key, Label: b.spec.label, Report: b.report,
		})
	}
	return m
}
