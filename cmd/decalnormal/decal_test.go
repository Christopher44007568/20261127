package main

import (
	"math"
	"testing"
)

func TestReorientKeepsUnitLength(t *testing.T) {
	bases := []Vec3{
		v3(0, 0, 1),
		v3(0.3, -0.2, 1).Normalized(),
		v3(-0.7, 0.5, 0.4).Normalized(),
	}
	details := []Vec3{
		v3(0, 0, 1),
		v3(0.6, -0.4, 1).Normalized(),
		v3(-0.9, 0.2, 0.3).Normalized(),
	}
	for _, b := range bases {
		for _, d := range details {
			out := reorient(b, d)
			if dev := math.Abs(out.Length() - 1); dev > 1e-15 {
				t.Fatalf("reorient left unit sphere: base=%v detail=%v dev=%v", b, d, dev)
			}
		}
	}
}

// TestHighlightCollapse demonstrates the failure mode called out in the
// brief: a linear average pulls the perturbed normal back toward the base
// normal, shrinking its height (Z) component and flattening the highlight,
// while reorientation preserves the full detail deflection.
func TestHighlightCollapse(t *testing.T) {
	base := v3(0, 0, 1)
	detail := v3(0.8, 0, 1).Normalized()

	reoriented := reorient(base, detail)
	linear := base.Add(reoriented).Scale(0.5).Normalized()

	if math.Abs(reoriented.Z-detail.Z) > 1e-12 {
		t.Fatalf("reorientation must preserve detail Z, got %.4f want %.4f", reoriented.Z, detail.Z)
	}
	if linear.Z <= detail.Z {
		t.Fatalf("linear average should weaken highlight: linearZ=%.4f detailZ=%.4f", linear.Z, detail.Z)
	}
	flatten := 100 * (linear.Z - detail.Z) / (1 - detail.Z)
	t.Logf("detail Z=%.4f  reoriented Z=%.4f  linear-average Z=%.4f (%.1f%% of deflection erased)",
		detail.Z, reoriented.Z, linear.Z, flatten)
}

// TestOverlapStaysUnit stacks several strong details through the same
// blending path the renderer uses for overlapping decals.
func TestOverlapStaysUnit(t *testing.T) {
	current := v3(0.2, -0.3, 1).Normalized()
	details := []Vec3{
		v3(0.9, 0.1, 0.4).Normalized(),
		v3(-0.8, 0.5, 0.3).Normalized(),
		v3(0.4, -0.9, 0.2).Normalized(),
	}
	for _, tn := range details {
		oriented := reorient(current, tn)
		current = Lerp(current, oriented, 0.9).Normalized()
	}
	if dev := math.Abs(current.Length() - 1); dev > 1e-15 {
		t.Fatalf("overlapped blend left unit sphere: dev=%v", dev)
	}
}

// TestProjectorFootprintIsCurvatureIndependent confirms the projector is
// defined in world space: equal radius on a flat and on a sharp bump yield
// equal-size footprints measured along the surface tangents.
func TestProjectorFootprintIsCurvatureIndependent(t *testing.T) {
	flat := newDecal("flat", 0, 0, 1.0, patRings, 1, v3(1, 1, 1))
	bump := newDecal("bump", -2.7, 1.3, 1.0, patRings, 1, v3(1, 1, 1))
	dimple := newDecal("dimple", 2.8, -1.4, 1.0, patBump, 1, v3(1, 1, 1))

	measure := func(d Decal) (float64, float64) {
		pr := d.project(d.Frame.Origin.Add(d.Frame.T1.Scale(d.Radius)))
		px := pr.U
		pr = d.project(d.Frame.Origin.Add(d.Frame.T2.Scale(d.Radius)))
		return math.Abs(px), math.Abs(pr.V)
	}
	for _, d := range []Decal{flat, bump, dimple} {
		u, v := measure(d)
		if math.Abs(u-1) > 1e-12 || math.Abs(v-1) > 1e-12 {
			t.Fatalf("%s footprint drifted: u=%.4f v=%.4f", d.Name, u, v)
		}
	}
}
