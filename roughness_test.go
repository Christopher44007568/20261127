package main

import (
	"math"
	"testing"
)

func checkerTexture(size int) *RoughnessTexture {
	values := make([]float64, size*size)
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if (x+y)%2 == 0 {
				values[y*size+x] = 0.10
			} else {
				values[y*size+x] = 0.90
			}
		}
	}
	return NewRoughnessFromValues(values, size)
}

func TestFootprintUsesSecondMoment(t *testing.T) {
	tex := checkerTexture(4)
	fp := Footprint{LOD: 2, LongTexels: 4, AreaTexels: 16}
	wrong := tex.Resolve(ModeNaiveMIP, 0.5, 0.5, fp)
	correct := tex.Resolve(ModeFootprint, 0.5, 0.5, fp)

	const mean = 0.5
	const meanAlpha = (0.01 + 0.81) * 0.5
	if math.Abs(wrong.Roughness-mean) > 1e-9 {
		t.Fatalf("naive mip roughness = %v, want %v", wrong.Roughness, mean)
	}
	if math.Abs(correct.RMSRoughness-math.Sqrt(meanAlpha)) > 1e-9 {
		t.Fatalf("RMS roughness = %v, want %v", correct.RMSRoughness, math.Sqrt(meanAlpha))
	}
	if correct.Roughness <= wrong.Roughness {
		t.Fatalf("footprint roughness %v must exceed naive mean %v", correct.Roughness, wrong.Roughness)
	}
	if correct.Variance <= 0.15 {
		t.Fatalf("checker variance too small: %v", correct.Variance)
	}
}

func TestFootprintRoughnessIsContinuousAcrossMips(t *testing.T) {
	tex := BuildRoughnessTexture(128)
	u, v := 0.371, 0.623
	var prev float64
	for i := 0; i <= 80; i++ {
		lod := -0.5 + float64(i)*5.5/80
		fp := Footprint{LOD: lod, LongTexels: math.Exp2(math.Max(lod, 0)), AreaTexels: math.Exp2(2 * math.Max(lod, 0))}
		got := tex.Resolve(ModeFootprint, u, v, fp).Roughness
		if i > 0 && math.Abs(got-prev) > 0.035 {
			t.Fatalf("roughness jump at i=%d: %v -> %v", i, prev, got)
		}
		if got < MinRoughness || got > 1 {
			t.Fatalf("roughness out of range: %v", got)
		}
		prev = got
	}
}

func TestDistantFootprintIsGenerallyRougherThanClose(t *testing.T) {
	tex := BuildRoughnessTexture(128)
	var closeSum, farSum float64
	const n = 64
	for i := 0; i < n; i++ {
		u := 0.25 + 0.5*float64(i)/float64(n-1)
		v := 0.18 + 0.64*float64((i*37)%n)/float64(n-1)
		closeFP := FootprintFromDerivatives(0.001, 0, 0, 0.002, float64(tex.Size))
		farFP := FootprintFromDerivatives(0.018, 0, 0, 0.036, float64(tex.Size))
		closeSum += tex.Resolve(ModeFootprint, u, v, closeFP).Roughness
		farSum += tex.Resolve(ModeFootprint, u, v, farFP).Roughness
	}
	if farSum/n <= closeSum/n+0.01 {
		t.Fatalf("far mean roughness %.3f should be clearly above close %.3f", farSum/n, closeSum/n)
	}
}
