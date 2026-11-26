package main

import (
	"math"
	"testing"
)

func newTestBlender(t *testing.T) (*Blender, *Mesh, []*Target) {
	t.Helper()
	base := BuildBaseFace()
	targets := BuildTargets(base)
	return &Blender{Base: base, Targets: targets}, base, targets
}

func approxEqual(a, b float64, eps float64) bool {
	return math.Abs(a-b) <= eps
}

func TestAllOffReturnsBasePerVertex(t *testing.T) {
	blender, base, targets := newTestBlender(t)
	result := blender.Blend(make([]float64, len(targets)))

	if result.Denominator != 0 {
		t.Fatalf("denominator = %v, want 0", result.Denominator)
	}
	posed := result.Mesh
	for i := range base.Positions {
		if posed.Positions[i] != base.Positions[i] {
			t.Fatalf("vertex %d position drifted with all targets off: %v vs %v", i, posed.Positions[i], base.Positions[i])
		}
		if posed.Normals[i] != base.Normals[i] {
			t.Fatalf("vertex %d normal drifted with all targets off: %v vs %v", i, posed.Normals[i], base.Normals[i])
		}
		if posed.Tangents[i] != base.Tangents[i] {
			t.Fatalf("vertex %d tangent drifted with all targets off: %v vs %v", i, posed.Tangents[i], base.Tangents[i])
		}
	}
	if len(posed.Tris) != len(base.Tris) {
		t.Fatalf("triangle count changed: %d vs %d", len(posed.Tris), len(base.Tris))
	}
}

func TestSingleTargetAppliesAllThreeFrames(t *testing.T) {
	blender, base, targets := newTestBlender(t)
	smile := targets[0]
	result := blender.Blend([]float64{0.75, 0, 0, 0})

	if result.Denominator != 0.75 || result.EffectiveWeights[0] != 0.75 {
		t.Fatalf("single target must keep its raw weight, got %+v", result.EffectiveWeights)
	}

	moved := false
	for i, basePos := range base.Positions {
		wantPos := basePos.Add(smile.PosD[i].Scale(0.75))
		if got := result.Mesh.Positions[i]; !approxEqual(got.Sub(wantPos).Length(), 0, 1e-12) {
			t.Fatalf("vertex %d position = %v, want %v", i, got, wantPos)
		}
		if result.Mesh.Normals[i].Sub(base.Normals[i]).LengthSquared() > 0 {
			moved = true
		}
		if !approxEqual(result.Mesh.Normals[i].Length(), 1, 1e-12) {
			t.Fatalf("vertex %d blended normal is not unit length", i)
		}
	}
	if !moved {
		t.Fatal("blended normals are identical to base; normal deltas were ignored")
	}
}

func TestCombinedTargetsNormalizeWeights(t *testing.T) {
	blender, _, targets := newTestBlender(t)
	raw := []float64{0.8, 0.8, 0.6, 0}
	clamped, effective, denom := blender.ResolveWeights(raw)

	wantSum := 2.2
	if !approxEqual(denom, wantSum, 1e-12) {
		t.Fatalf("denominator = %v, want %v", denom, wantSum)
	}
	gotSum := 0.0
	for i := range targets {
		if clamped[i] != raw[i] {
			t.Fatalf("in-range weight %d was altered by clamping", i)
		}
		gotSum += effective[i]
	}
	if !approxEqual(gotSum, 1, 1e-12) {
		t.Fatalf("normalized weights sum to %v, want 1", gotSum)
	}

	// A combination whose clamped sum is below 1 must not be renormalized.
	_, lowEff, lowDenom := blender.ResolveWeights([]float64{0.3, 0.2, 0, 0})
	if lowDenom != 0.5 || lowEff[0] != 0.3 || lowEff[1] != 0.2 {
		t.Fatalf("sub-unit combination was renormalized: %+v", lowEff)
	}
}

func TestExtremeWeightsAreClamped(t *testing.T) {
	blender, _, targets := newTestBlender(t)
	clamped, effective, denom := blender.ResolveWeights([]float64{9, -4, 0.5, 2.5})

	want := []float64{1, 0, 0.5, 1}
	for i := range targets {
		if clamped[i] != want[i] {
			t.Fatalf("clamped[%d] = %v, want %v", i, clamped[i], want[i])
		}
	}
	if !approxEqual(denom, 2.5, 1e-12) {
		t.Fatalf("denominator = %v, want 2.5", denom)
	}
	sum := 0.0
	for _, weight := range effective {
		sum += weight
	}
	if !approxEqual(sum, 1, 1e-12) {
		t.Fatalf("clamped+normalized weights sum to %v, want 1", sum)
	}
}

func TestNaNWeightTreatedAsZero(t *testing.T) {
	blender, _, targets := newTestBlender(t)
	clamped, _, denom := blender.ResolveWeights([]float64{math.NaN(), 0, 0, 0})

	if clamped[0] != 0 || denom != 0 {
		t.Fatalf("NaN must collapse to zero: clamped=%v denom=%v", clamped, denom)
	}
	if len(targets) != 4 {
		t.Fatalf("target count = %d, want 4", len(targets))
	}
}

func TestTangentFrameStaysOrthonormal(t *testing.T) {
	blender, _, _ := newTestBlender(t)
	result := blender.Blend([]float64{1, 1, 1, 1})

	for i := range result.Mesh.Positions {
		n := result.Mesh.Normals[i]
		tan := result.Mesh.Tangents[i]
		if !approxEqual(n.Length(), 1, 1e-10) || !approxEqual(tan.Length(), 1, 1e-10) {
			t.Fatalf("vertex %d frame is not unit length", i)
		}
		if math.Abs(n.Dot(tan)) > 1e-10 {
			t.Fatalf("vertex %d tangent/normal dot = %v, want ~0", i, n.Dot(tan))
		}
	}
}

func TestSmileLineNormalChangesUnderBlend(t *testing.T) {
	// Regression guard for the original defect: blending only positions kept
	// base normals on the nasolabial fold, so smile-line shading was wrong.
	blender, base, targets := newTestBlender(t)
	smile := targets[0]

	// Find the vertex with the largest smile normal delta on the crease.
	worst, worstDelta := -1, 0.0
	for i, delta := range smile.NormalD {
		if length := delta.Length(); length > worstDelta {
			worst, worstDelta = i, length
		}
	}
	if worst < 0 || worstDelta < 1e-6 {
		t.Fatalf("smile target has no meaningful normal deltas (max %v)", worstDelta)
	}

	full := blender.Blend([]float64{1, 0, 0, 0})
	if full.Mesh.Normals[worst] == base.Normals[worst] {
		t.Fatal("crease normal unchanged: normal blending is not wired in")
	}

	// A position-only blend would leave this normal pointing along the base
	// normal. Verify the posed one follows the target normal instead.
	baseN := base.Normals[worst]
	targetN := baseN.Add(smile.NormalD[worst]).Normalized()
	posedN := full.Mesh.Normals[worst]
	if posedN.Dot(targetN) <= baseN.Dot(targetN) {
		t.Fatal("posed crease normal does not move toward the target normal")
	}
}
