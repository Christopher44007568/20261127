package main

import (
	"flag"
	"fmt"
	"log"
	"path/filepath"
)

func main() {
	outDir := flag.String("out", "artifacts", "directory for the four rendered PNG files")
	width := flag.Int("width", 960, "output image width in pixels")
	height := flag.Int("height", 720, "output image height in pixels")
	aa := flag.Int("aa", 2, "supersampling factor")
	worldScale := flag.Float64("scale", DefaultWorldScale, "world-space texture repeat scale")
	hardness := flag.Float64("hardness", DefaultHardness, "normal-weight exponent / blend hardness")
	textureSize := flag.Int("texture-size", 512, "procedural texture resolution")
	flag.Parse()

	texture := NewMipmapTexture(NewRockTexture(*textureSize, 20261127))
	scenes := BuildScenes(texture, *worldScale, *hardness)
	opts := RenderOptions{Width: *width, Height: *height, AA: *aa}
	report := MetricsReport{
		WorldScale:          *worldScale,
		Hardness:            *hardness,
		GravityUVScale:      *worldScale,
		ProjectionWeighting: "pow(abs(N),hardness), component sum normalized to 1",
		Images:              make(map[string]ImageMetrics, len(scenes)),
	}
	report.WeightSumMaxError, report.WhiteSeamPeak = CoreInvariantMetrics(*worldScale, *hardness)

	for _, scene := range scenes {
		path := filepath.Join(*outDir, scene.Name+".png")
		frame := RenderFrame(scene, opts)
		metrics := MeasureFrame(frame)
		metrics.File = filepath.Base(path)
		report.Images[scene.Name] = metrics
		img := FrameToRGBA(frame, 1.34)
		if err := WritePNG(path, img); err != nil {
			log.Fatalf("write %s: %v", path, err)
		}
		fmt.Printf("wrote %s (%dx%d)\n", path, *width, *height)
	}

	metricsPath := filepath.Join(*outDir, "metrics.json")
	if err := WriteJSON(metricsPath, report); err != nil {
		log.Fatalf("write %s: %v", metricsPath, err)
	}
	fmt.Printf("wrote %s\n", metricsPath)
}
