package main

import (
	"math"
	"testing"
)

func fixedSamples() [numMaterials]MaterialSample {
	return [numMaterials]MaterialSample{
		{Albedo: Vec3{0.18, 0.10, 0.05}, Height: 0.38, Rough: 0.82, Specular: 0.10},
		{Albedo: Vec3{0.08, 0.24, 0.04}, Height: 0.55, Rough: 0.93, Specular: 0.045},
		{Albedo: Vec3{0.26, 0.31, 0.34}, Height: 0.72, Rough: 0.34, Specular: 0.58},
	}
}

func TestWeightsAreNormalized(t *testing.T) {
	cov := [numMaterials]float64{0.41, 0.33, 0.26}
	naive := naiveWeights(cov)
	height := heightWeights(cov, fixedSamples())
	for name, w := range map[string][numMaterials]float64{"naive": naive, "height": height} {
		sum := 0.0
		for _, v := range w {
			if v < 0 || v > 1 {
				t.Fatalf("%s weight out of range: %+v", name, w)
			}
			sum += v
		}
		if math.Abs(sum-1) > 1e-12 {
			t.Fatalf("%s weights sum to %v, want 1", name, sum)
		}
	}
}

func TestHeightBlendReducesBoundaryMixing(t *testing.T) {
	samples := fixedSamples()
	naiveMix, heightMix := 0.0, 0.0
	steps := 101
	for i := 0; i < steps; i++ {
		tv := float64(i) / float64(steps-1)
		cov := [numMaterials]float64{
			(1 - tv) * 0.78,
			(1 - math.Abs(2*tv-1)) * 0.34,
			tv * 0.78,
		}
		nw := naiveWeights(cov)
		hw := heightWeights(cov, samples)
		naiveMix += 1 - maxWeight(nw)
		heightMix += 1 - maxWeight(hw)
	}
	if heightMix >= naiveMix*0.85 {
		t.Fatalf("height blend does not sharpen enough: naive mixing %.4f, height mixing %.4f", naiveMix, heightMix)
	}
}

func maxWeight(w [numMaterials]float64) float64 {
	m := w[0]
	for _, v := range w[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func TestTriplanarWeightsAreNormalized(t *testing.T) {
	for _, n := range []Vec3{
		{0, 1, 0},
		{0.6, 0.7, 0.2},
		{-0.8, 0.1, 0.4},
	} {
		w := triplanarWeights(n, 5)
		sum := w[0] + w[1] + w[2]
		if math.Abs(sum-1) > 1e-12 {
			t.Fatalf("triplanar weights sum to %v for normal %+v", sum, n)
		}
	}
}

func TestWetStoneBrightensNearWaterline(t *testing.T) {
	dry := wetnessAt(-0.4, 0.2, waterLevel+0.30)
	wet := wetnessAt(-0.4, 0.2, waterLevel+0.005)
	if dry > 0.05 {
		t.Fatalf("dry sample unexpectedly wet: %v", dry)
	}
	if wet < 0.65 {
		t.Fatalf("waterline sample not wet enough: %v", wet)
	}
	s := fixedSamples()[WetStone]
	dryAlbedo := luminance(materialAlbedo(WetStone, dry, s))
	wetAlbedo := luminance(materialAlbedo(WetStone, wet, s))
	if wetAlbedo <= dryAlbedo*1.20 {
		t.Fatalf("wet stone not brighter: dry %.4f, wet %.4f", dryAlbedo, wetAlbedo)
	}
}
