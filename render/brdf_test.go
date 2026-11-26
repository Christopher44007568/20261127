package render

import (
	"math"
	"testing"
)

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-12
}

func testMaterial(weight, cap float64) Material {
	return Material{
		Albedo:           V(0.58, 0.155, 0.042),
		ClearcoatWeight:  weight,
		CoatingRoughness: 0.08,
		CoatingF0:        DefaultCoatingF0,
		EnergyCap:        cap,
	}
}

func TestZeroClearcoatReturnsUnderlayerExactly(t *testing.T) {
	m := testMaterial(0, 0.85)
	light := V(0.71, 0.54, 0.39)
	env := V(0.93, 0.82, 0.71)
	expected := m.Albedo.MulVec(light)

	for i := 0; i <= 100; i++ {
		nDotV := float64(i) / 100
		sample := m.Shade(nDotV, light, env)
		if sample.Clearcoat != (Vec3{}) {
			t.Fatalf("coat at nDotV %.2f = %+v, want zero", nDotV, sample.Clearcoat)
		}
		if sample.Split != (EnergySplit{Clearcoat: 0, Underlayer: 1}) {
			t.Fatalf("split at nDotV %.2f = %+v", nDotV, sample.Split)
		}
		if sample.Diffuse != expected {
			t.Fatalf("base at nDotV %.2f = %+v, want %+v", nDotV, sample.Diffuse, expected)
		}
		if sample.Composite() != sample.Diffuse {
			t.Fatalf("composite differs from base at nDotV %.2f", nDotV)
		}
	}
}

func TestFresnelSplitConservesAndCapsEnergy(t *testing.T) {
	const cap = 0.72
	m := testMaterial(1, cap)
	grazing := m.SplitEnergy(0)

	for i := 0; i <= 1000; i++ {
		nDotV := float64(i) / 1000
		split := m.SplitEnergy(nDotV)
		if split.Clearcoat < 0 || split.Clearcoat > cap+1e-15 {
			t.Fatalf("coat energy at nDotV %.3f = %.17g, cap %.2f", nDotV, split.Clearcoat, cap)
		}
		if !almostEqual(split.Clearcoat+split.Underlayer, 1) {
			t.Fatalf("split at nDotV %.3f sums to %.17g", nDotV, split.Clearcoat+split.Underlayer)
		}
	}
	if !almostEqual(grazing.Clearcoat, cap) {
		t.Fatalf("grazing coat = %.17g, want capped value %.17g", grazing.Clearcoat, cap)
	}
	if !(grazing.Clearcoat >= m.SplitEnergy(1).Clearcoat) {
		t.Fatal("grazing reflectance should be at least normal-incidence reflectance")
	}
}

func TestGrazingHighlightAndCompositeStayWithinBudget(t *testing.T) {
	m := testMaterial(1, DefaultEnergyCap)
	one := V(1, 1, 1)

	for i := 0; i <= 1000; i++ {
		nDotV := float64(i) / 1000
		sample := m.Shade(nDotV, one, one)
		if sample.Clearcoat.X > DefaultEnergyCap+1e-15 ||
			sample.Clearcoat.Y > DefaultEnergyCap+1e-15 ||
			sample.Clearcoat.Z > DefaultEnergyCap+1e-15 {
			t.Fatalf("highlight at nDotV %.3f exceeds energy cap: %+v", nDotV, sample.Clearcoat)
		}
		composite := sample.Composite()
		if composite.X > 1+1e-12 || composite.Y > 1+1e-12 || composite.Z > 1+1e-12 {
			t.Fatalf("composite at nDotV %.3f exceeds unit energy: %+v", nDotV, composite)
		}
	}
}

func TestRoughnessSpreadIsBoundedAndMonotonic(t *testing.T) {
	last := 0.0
	for i := 0; i <= 100; i++ {
		r := float64(i) / 100
		spread := RoughnessToSpread(r)
		if spread < last {
			t.Fatalf("spread decreased at roughness %.2f: %.6f < %.6f", r, spread, last)
		}
		if spread <= 0 || spread > math.Pi/4 {
			t.Fatalf("spread at roughness %.2f out of range: %.6f", r, spread)
		}
		last = spread
	}
}

func TestStudioEnvironmentIsBounded(t *testing.T) {
	directions := []Vec3{
		keyDirection,
		V(0, 1, 0),
		V(0, -1, 0),
		V(1, 0, 0),
		V(0.3, 0.5, -0.8),
	}
	for _, dir := range directions {
		c := StudioEnvironment(dir)
		if c.X < 0 || c.X > 1 || c.Y < 0 || c.Y > 1 || c.Z < 0 || c.Z > 1 {
			t.Fatalf("environment out of bounds for %+v: %+v", dir, c)
		}
	}
}
