package main

import "testing"

func testScene() *Scene {
	return &Scene{
		Texture: BuildRoughnessTexture(512),
		LUT:     BuildEnergyLUT(12, 9, 48),
		Lights:  DefaultLights(),
		F0:      V(0.74, 0.80, 0.88),
	}
}

func TestOriginalHasPointSparklesAndCorrectionRemovesThem(t *testing.T) {
	scene := testScene()
	raw := AnalyzeMode(scene, ModeOriginal, 208)
	naive := AnalyzeMode(scene, ModeNaiveMIP, 208)
	fixed := AnalyzeMode(scene, ModeFootprint, 208)

	if raw.SparklePixels < 120 {
		t.Fatalf("original mode should contain point sparkles, got %d", raw.SparklePixels)
	}
	if fixed.SparklePixels > raw.SparklePixels/20 {
		t.Fatalf("correction retained too many sparkles: fixed=%d raw=%d", fixed.SparklePixels, raw.SparklePixels)
	}
	if raw.HighFrequencyError < fixed.HighFrequencyError*4 {
		t.Fatalf("raw HF %.4f should greatly exceed corrected HF %.4f",
			raw.HighFrequencyError, fixed.HighFrequencyError)
	}
	if fixed.MeanRoughness <= naive.MeanRoughness {
		t.Fatalf("footprint RMS roughness %.3f should exceed naive mean %.3f",
			fixed.MeanRoughness, naive.MeanRoughness)
	}
}

func TestCorrectedBrightnessIsContinuousOverDistance(t *testing.T) {
	scene := testScene()
	stats, maxJump := AnalyzeContinuity(scene)
	if len(stats) < 5 {
		t.Fatalf("too few continuity samples: %d", len(stats))
	}
	if maxJump > 0.05 {
		t.Fatalf("luminance jump %.4f exceeds continuity budget", maxJump)
	}
}

func TestEnergyLUTIsBoundedAndCompensationFinite(t *testing.T) {
	lut := BuildEnergyLUT(12, 9, 48)
	for _, rough := range []float64{MinRoughness, 0.25, 0.6, 1} {
		for _, mu := range []float64{0.05, 0.3, 0.7, 1} {
			e := lut.Energy(rough, mu)
			if e < 0 || e > 1 {
				t.Fatalf("energy out of bounds r=%.2f mu=%.2f E=%.3f", rough, mu, e)
			}
			ms := lut.MultiScatterBRDF(V(1, 1, 1), rough, mu, mu)
			if ms.X < 0 || ms.X > 2 || ms.Y < 0 || ms.Z < 0 {
				t.Fatalf("invalid compensation %+v", ms)
			}
		}
	}
}
