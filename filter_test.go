package main

import (
	"math"
	"testing"
)

func TestFootprintSVDAxes(t *testing.T) {
	fp := makeFootprint(Vec2{3, 0}, Vec2{0, 1})
	if !fp.Valid {
		t.Fatal("footprint should be valid")
	}
	if math.Abs(fp.Major-3) > 1e-9 || math.Abs(fp.Minor-1) > 1e-9 {
		t.Fatalf("unexpected axes: major=%g minor=%g", fp.Major, fp.Minor)
	}
	if math.Abs(math.Abs(fp.AxisMajor.X)+math.Abs(fp.AxisMajor.Y)-1) > 1e-9 {
		t.Fatalf("major axis is not axis aligned: %+v", fp.AxisMajor)
	}
}

func TestFilterBudgetIsHardLimit(t *testing.T) {
	pattern := NewStripePattern()
	tex := NewTexture(pattern, 64)
	fp := makeFootprint(Vec2{12, 2}, Vec2{-1, 1})
	for _, budget := range []int{4, 8, 16, 32, 64, 128} {
		_, stats := anisotropic(tex, Vec2{31.5, 33.5}, fp, budget)
		if stats.Taps > budget {
			t.Fatalf("budget %d exceeded: taps=%d", budget, stats.Taps)
		}
		if stats.ActualProbes > 0 && stats.Taps == 0 {
			t.Fatalf("budget %d produced no taps", budget)
		}
	}
}

func TestAnisotropicUsesMoreProbesThanFixed(t *testing.T) {
	pattern := NewStripePattern()
	tex := NewTexture(pattern, 64)
	fp := makeFootprint(Vec2{12, 3}, Vec2{1, 2})
	_, fixed := fixedFour(tex, Vec2{10, 10})
	_, aniso := anisotropic(tex, Vec2{10, 10}, fp, 128)
	if fixed.Taps != 4 {
		t.Fatalf("fixed four should read four texels, got %d", fixed.Taps)
	}
	if aniso.ActualProbes <= fixed.Probes {
		t.Fatalf("anisotropic probe count %d not above fixed %d", aniso.ActualProbes, fixed.Probes)
	}
}

func TestReferenceAttenuatesFineDetailAtGrazingFootprint(t *testing.T) {
	pattern := NewStripePattern()
	uv := Vec2{17.25, 41.75}
	sharp := pattern.analyticFiltered(uv, Footprint{
		Valid:     true,
		Major:     1e-6,
		Minor:     1e-6,
		AxisMajor: Vec2{1, 0},
		AxisMinor: Vec2{0, 1},
	}).Luma()
	blurred := pattern.analyticFiltered(uv, Footprint{
		Valid:     true,
		Major:     16,
		Minor:     2,
		AxisMajor: Vec2{1, 0},
		AxisMinor: Vec2{0, 1},
	}).Luma()
	if math.Abs(blurred-sharp) < 1e-4 {
		t.Fatalf("wide footprint failed to attenuate stripes: sharp=%g blurred=%g", sharp, blurred)
	}
}
