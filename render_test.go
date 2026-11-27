package main

import (
	"math"
	"testing"
)

func flatTestTexture(size int, height float32) *StoneTexture {
	tex := &StoneTexture{
		Width:       size,
		Height:      size,
		heightField: make([]float32, size*size),
		albedo:      make([]RGB, size*size),
		gradX:       make([]float32, size*size),
		gradY:       make([]float32, size*size),
	}
	for i := range tex.heightField {
		tex.heightField[i] = height
		tex.albedo[i] = RGB{0.4, 0.4, 0.4}
	}
	return tex
}

func TestBudgetRetainsLastValidLayer(t *testing.T) {
	tex := flatTestTexture(32, 0.2)
	const budget = 4
	result := traceParallax(tex, 0.5, 0.5, vec3{0.9, 0, 0.435}, ViewGrazing, budget, 2)
	if result.Status != TraceBudget {
		t.Fatalf("status = %v, want TraceBudget", result.Status)
	}
	if result.Layers != budget {
		t.Fatalf("layers = %d, want %d", result.Layers, budget)
	}
	if result.U >= 0.5 || result.U < 0 {
		t.Fatalf("fallback UV u = %.5f, want last valid layer inside [0, 0.5)", result.U)
	}
	if math.Abs(result.U-0.5) < 1e-6 {
		t.Fatal("budget fallback returned the base layer instead of the last stepped layer")
	}
}

func TestBoundaryFallbackDoesNotWrap(t *testing.T) {
	tex := flatTestTexture(32, 0.2)
	result := traceParallax(tex, 0.004, 0.5, vec3{0.99, 0, 0.141}, ViewFallback, 256, 2)
	if result.Status != TraceBoundary {
		t.Fatalf("status = %v, want TraceBoundary", result.Status)
	}
	if result.U < 0 || result.U > 1 || result.V < 0 || result.V > 1 {
		t.Fatalf("fallback UV = (%.5f, %.5f), want guarded in-range sample", result.U, result.V)
	}
	if result.U > 0.02 {
		t.Fatalf("fallback UV u = %.5f, expected the last sample before the left edge", result.U)
	}
}

func TestGradientAtIsConsistent(t *testing.T) {
	tex := &StoneTexture{
		Width:       4,
		Height:      1,
		heightField: []float32{0.0, 0.2, 0.6, 0.8},
		albedo:      make([]RGB, 4),
		gradX:       make([]float32, 4),
		gradY:       make([]float32, 4),
	}
	computeGradients(tex)
	gx, gy := tex.GradientAt(0.5, 0)
	if gx <= 0 {
		t.Fatalf("gradient x = %f, want positive height slope", gx)
	}
	if gy != 0 {
		t.Fatalf("gradient y = %f, want zero on a single-row height field", gy)
	}
}
