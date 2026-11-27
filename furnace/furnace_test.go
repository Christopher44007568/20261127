package furnace

import (
	"math"
	"os"
	"strings"
	"testing"
)

func TestLambertDirectionalIntegral(t *testing.T) {
	albedo := V(0.71, 0.19, 0.43)
	for _, mu := range []float64{1, 0.73, 0.31, 0.08} {
		got := IntegrateHemisphere(func(v, l Vec3) Vec3 {
			return LambertBRDF(albedo)
		}, mu, 32768)
		for c, name := range []string{"r", "g", "b"} {
			want := []float64{albedo.X, albedo.Y, albedo.Z}[c]
			val := []float64{got.X, got.Y, got.Z}[c]
			if math.Abs(val-want) > 0.012 {
				t.Fatalf("mu=%.2f channel=%s got %.4f want %.4f", mu, name, val, want)
			}
		}
	}
}

func TestGGXWhiteEnergyStaysBounded(t *testing.T) {
	for _, rough := range []float64{0.16, 0.42, 0.72, 0.95} {
		alpha := clamp(rough*rough, 0.0015, 1)
		for _, mu := range []float64{1, 0.5, 0.12, 0.03} {
			e := integrateGGXEnergy(mu, alpha, V(1, 1, 1), 16384).X
			if e < 0.18 || e > 1.025 || math.IsNaN(e) {
				t.Fatalf("bad GGX energy rough=%.2f mu=%.2f E=%.4f", rough, mu, e)
			}
		}
	}
}

func TestWhiteFurnacePresets(t *testing.T) {
	for _, m := range Presets() {
		r := RunWhiteFurnace(m, 72, 8192, 16384, 0.03)
		for _, p := range r.Points {
			if !p.Pass {
				t.Errorf("%s mu=%.3f corrected=%.5f err=%.5f", m.Kind, p.NdotV, p.CorrectedEnergy, p.AbsError)
			}
		}
		if !r.Sphere.Pass {
			t.Errorf("%s sphere failed: worst mu=%.4f err=%.5f rim=%.5f",
				m.Kind, r.Sphere.WorstNdotV, r.Sphere.WorstError, r.Sphere.RimError)
		}
		if r.WorstError > 0.03 {
			t.Fatalf("%s worst error %.5f > tolerance", m.Kind, r.WorstError)
		}
		t.Logf("%s worst=%.5f at mu=%.3f sphere_checks=%d rim=%.5f",
			m.Kind, r.WorstError, r.WorstNdotV, r.Sphere.DirectionChecks, r.Sphere.RimError)
	}
}

func TestCorrectedColoredEnergyNeverExceedsOne(t *testing.T) {
	for _, m := range Presets() {
		table := BuildEnergyTable(m, 64, 8192)
		for _, mu := range []float64{1, 0.8, 0.5, 0.25, 0.1, 0.04, 0.02} {
			e := CorrectedDirectionalAlbedo(m, table, mu, 16384)
			if e.X > 1.025 || e.Y > 1.025 || e.Z > 1.025 {
				t.Fatalf("%s mu=%.2f energy=%+v exceeds unit incoming energy", m.Kind, mu, e)
			}
			if e.MinChannel() < -1e-6 {
				t.Fatalf("%s negative energy %+v", m.Kind, e)
			}
		}
	}
}

func TestNormalizedBRDFFiniteAndNonNegative(t *testing.T) {
	for _, m := range Presets() {
		table := DefaultEnergyTable(m)
		for i := 1; i <= 40; i++ {
			mu := float64(i) / 41
			v := outgoingDirection(mu)
			for j := 0; j < 24; j++ {
				u1, u2 := Hammersley2D(j, 24)
				l := CosineHemisphere(u1, u2)
				raw := RawBRDF(m, v, l)
				got := NormalizedBRDF(m, table, v, l)
				if !finiteVec(got) || got.MinChannel() < -1e-8 {
					t.Fatalf("%s invalid normalized BRDF raw=%+v got=%+v", m.Kind, raw, got)
				}
			}
		}
	}
}

func TestMISWhiteFurnaceOfActualBRDF(t *testing.T) {
	for _, m := range Presets() {
		wm := WhiteFurnaceMaterial(m)
		table := BuildEnergyTable(wm, 72, 8192)
		for _, mu := range []float64{1, 0.55, 0.18, 0.06, 0.025} {
			e := IntegrateNormalizedBRDFMIS(wm, table, mu, 16384).X
			if math.Abs(e-1) > 0.035 {
				t.Fatalf("%s mu=%.3f MIS E=%.5f", m.Kind, mu, e)
			}
		}
	}
}

func TestSmallRendersAreFinite(t *testing.T) {
	for _, corrected := range []bool{false, true} {
		for _, m := range Presets() {
			cfg := RenderConfig{Width: 72, Height: 72, Corrected: corrected, AA: 1, Exposure: 1.1}
			if corrected {
				cfg.Table = BuildEnergyTable(m, 24, 1024)
			}
			c := RenderSpecimen(m, cfg)
			for _, p := range c.Pix {
				if !finiteVec(p) || p.MinChannel() < -1e-6 {
					t.Fatalf("invalid render pixel for %s corrected=%v: %+v", m.Kind, corrected, p)
				}
			}
		}
	}

	sheet := RenderMaterialSheet(true, 180, RunAllWhiteFurnace())
	if sheet.Bounds().Dx() != 540 || sheet.Bounds().Dy() <= 224 {
		t.Fatalf("bad sheet bounds %v", sheet.Bounds())
	}
	heat := RenderEnergyHeatmap(nil)
	if heat.Bounds().Dx() != 1500 {
		t.Fatalf("bad heatmap bounds %v", heat.Bounds())
	}
	furnace := RenderWhiteFurnaceSheet(nil)
	if furnace.Bounds().Dx() != 920 {
		t.Fatalf("bad furnace bounds %v", furnace.Bounds())
	}
}

func TestGoModHasNoExternalRequires(t *testing.T) {
	data, err := os.ReadFile("../go.mod")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "require ") {
		t.Fatalf("go.mod contains external requirements:\n%s", data)
	}
}
