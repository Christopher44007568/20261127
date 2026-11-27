package material

import (
	"math"
	"testing"
)

func TestDefaultMaterialsPreserveIdentity(t *testing.T) {
	materials := DefaultMaterials()
	if len(materials) != 3 {
		t.Fatalf("expected 3 materials, got %d", len(materials))
	}

	for _, m := range materials {
		u := m.LowUniforms()
		if u.BaseColor != m.BaseColor {
			t.Fatalf("%s: low uniforms changed base color", m.ID)
		}
		if u.Metallic != m.Metallic || u.Roughness != m.Roughness {
			t.Fatalf("%s: low uniforms changed scalar identity parameters", m.ID)
		}
		if u.RoughnessMin != m.RoughnessMin || u.RoughnessMax != m.RoughnessMax {
			t.Fatalf("%s: low uniforms changed roughness interval", m.ID)
		}
		if m.Roughness < m.RoughnessMin || m.Roughness > m.RoughnessMax {
			t.Fatalf("%s: roughness %.3f outside [%.3f, %.3f]",
				m.ID, m.Roughness, m.RoughnessMin, m.RoughnessMax)
		}
		if delta := HueDelta(m.BaseColor.ToSRGB(), u.BaseColor.ToSRGB()); delta != 0 {
			t.Fatalf("%s: hue delta = %v", m.ID, delta)
		}
	}

	preservation := Preservation(materials)
	for _, p := range preservation {
		if !p.ColorPreserved || !p.RangePreserved {
			t.Fatalf("preservation failed: %+v", p)
		}
	}
}

func TestWeightsAtBoundaries(t *testing.T) {
	atHalf := WeightsAt(0.5, 0.1)
	if math.Abs(atHalf.Low-0.5) > 1e-12 || math.Abs(atHalf.Medium-0.5) > 1e-12 || atHalf.High != 0 {
		t.Fatalf("bad low/medium boundary weights: %+v", atHalf)
	}
	atOneAndHalf := WeightsAt(1.5, 0.1)
	if atOneAndHalf.Low != 0 || math.Abs(atOneAndHalf.Medium-0.5) > 1e-12 || math.Abs(atOneAndHalf.High-0.5) > 1e-12 {
		t.Fatalf("bad medium/high boundary weights: %+v", atOneAndHalf)
	}

	for lod := 0.0; lod <= 2.00001; lod += 0.01 {
		w := WeightsAt(lod, 0.1)
		sum := w.Low + w.Medium + w.High
		if math.Abs(sum-1) > 1e-9 {
			t.Fatalf("lod %.2f weights do not sum to 1: %+v", lod, w)
		}
		if w.Low < 0 || w.Medium < 0 || w.High < 0 || w.Low > 1 || w.Medium > 1 || w.High > 1 {
			t.Fatalf("lod %.2f has out-of-range weights: %+v", lod, w)
		}
	}
}

func TestViewSweepAndBoundaryContinuity(t *testing.T) {
	materials := DefaultMaterials()
	for _, m := range materials {
		for _, tier := range []Tier{TierLow, TierMedium, TierHigh} {
			samples := ViewSweep(m, tier)
			if len(samples) != 91 {
				t.Fatalf("%s/%s: expected 91 samples, got %d", m.ID, tier, len(samples))
			}
			for _, s := range samples {
				if math.IsNaN(s.DisplayLuma) || s.DisplayLuma < 0 || s.DisplayLuma > 1 {
					t.Fatalf("%s/%s: invalid display luma %+v", m.ID, tier, s)
				}
			}
		}
	}

	report := ScanContinuity(materials, 0.01, 0.10, 0.025)
	if len(report.Rows) != 201*len(materials) {
		t.Fatalf("expected %d rows, got %d", 201*len(materials), len(report.Rows))
	}
	if len(report.Boundaries) != 2 {
		t.Fatalf("expected 2 boundary summaries, got %d", len(report.Boundaries))
	}
	for _, b := range report.Boundaries {
		if !b.Passed {
			t.Fatalf("boundary %.1f did not pass: max step %.5f > %.3f",
				b.BoundaryLOD, b.ContinuousMaxStep, b.JumpThreshold)
		}
	}
}

func TestHexColorLinearizesKnownPreset(t *testing.T) {
	c, err := HexColor("#C87942")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(c.R-0.57758044) > 1e-6 || math.Abs(c.G-0.19120168) > 1e-6 || math.Abs(c.B-0.05448027) > 1e-6 {
		t.Fatalf("unexpected linear copper color: %+v", c)
	}
}
