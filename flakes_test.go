package flakepaint

import (
	"math"
	"testing"
)

func TestFlakeFieldDeterministic(t *testing.T) {
	field := DefaultFlakeField(0x20261127)
	frame := Frame{N: V(0, 0, 1), T: V(1, 0, 0), B: V(0, 1, 0)}
	points := []Vec2{{0, 0}, {3.25, -7.75}, {48.5, 91.25}, {-120, 44}}
	for _, p := range points {
		a := field.Sample(p, frame, 0.17)
		b := field.Sample(p, frame, 0.17)
		if a != b {
			t.Fatalf("sample at %+v is not deterministic", p)
		}
	}
}

func TestResolvedFlakeCouplesNormalRoughnessAndSpecular(t *testing.T) {
	field := DefaultFlakeField(0x20261127)
	var flake FlakeInstance
	found := false
	field.EnumerateFlakes(V2(-40, -40), V2(40, 40), func(fi FlakeInstance) {
		if !found {
			flake = fi
			found = true
		}
	})
	if !found {
		t.Fatal("expected at least one flake")
	}

	frame := Frame{N: V(0, 0, 1), T: V(1, 0, 0), B: V(0, 1, 0)}
	sample := field.Sample(flake.Center, frame, 0.03)
	if sample.Coverage < 0.5 {
		t.Fatalf("expected resolved coverage, got %.3f", sample.Coverage)
	}
	if sample.Detail < 0.5 {
		t.Fatalf("expected detail LOD, got %.3f", sample.Detail)
	}
	if angle := math.Acos(clamp(sample.N.Dot(frame.N), -1, 1)); angle < 0.004 {
		t.Fatalf("flake normal did not perturb base normal, angle=%g", angle)
	}
	if math.Abs(sample.Roughness-0.05) > 0.09 {
		t.Fatalf("flake roughness outside expected layered range: %.3f", sample.Roughness)
	}

	view := V(0, 0.05, 1).Normalize()
	light := V(-0.17, 0.20, 0.97).Normalize()
	baseSpec := directSpecular(frame.N, view, light, 0.02, V(1, 1, 1))
	flakeSpec := directSpecular(sample.N, view, light, 0.02, V(1, 1, 1))
	if baseSpec == flakeSpec {
		t.Fatal("flake normal did not alter specular direction")
	}
	baseReflect := reflectDir(view.Mul(-1), frame.N)
	flakeReflect := reflectDir(view.Mul(-1), sample.N)
	if baseReflect == flakeReflect {
		t.Fatal("flake normal did not alter reflection vector")
	}
}

func TestFarLODAggregatesWithoutResolvedSparkle(t *testing.T) {
	field := DefaultFlakeField(0x20261127)
	frame := Frame{N: V(0, 0, 1), T: V(1, 0, 0), B: V(0, 1, 0)}
	base := field.Sample(V2(23.5, 17.25), frame, 8.0)
	if base.Detail > 1e-9 {
		t.Fatalf("far sample should have no resolved detail, got %.6g", base.Detail)
	}
	if base.Coverage > 1e-9 {
		t.Fatalf("far sample should have no resolved coverage, got %.6g", base.Coverage)
	}
	if base.N.Sub(frame.N).Length() > 1e-9 {
		t.Fatal("far unresolved normal should equal the geometric normal")
	}
	if base.Unresolved < 0.2 {
		t.Fatalf("expected aggregate flake coverage, got %.3f", base.Unresolved)
	}

	for _, offset := range []Vec2{{0.03, 0}, {0, 0.03}, {0.07, 0.07}} {
		other := field.Sample(V2(23.5+offset.X, 17.25+offset.Y), frame, 8.0)
		if math.Abs(other.Unresolved-base.Unresolved) > 1e-4 {
			t.Fatalf("far aggregate coverage changed across tiny offsets: %g", other.Unresolved-base.Unresolved)
		}
		if math.Abs(other.Roughness-base.Roughness) > 1e-4 {
			t.Fatalf("far aggregate roughness changed across tiny offsets: %g", other.Roughness-base.Roughness)
		}
	}
}

func TestDensityGridIsFiniteAndNonUniform(t *testing.T) {
	field := DefaultFlakeField(7)
	g := buildDensityGrid(field, V2(-180, -110), V2(180, 110), 90, 55)
	if g.FlakeCount < 1000 {
		t.Fatalf("expected many active flakes, got %d", g.FlakeCount)
	}
	var minV, maxV, sum float64
	minV = math.Inf(1)
	maxV = math.Inf(-1)
	for _, v := range g.Vals {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatal("density grid contains non-finite value")
		}
		if v < 0 {
			t.Fatal("density grid contains negative value")
		}
		minV = math.Min(minV, v)
		maxV = math.Max(maxV, v)
		sum += v
	}
	if maxV <= minV {
		t.Fatal("density map is uniform")
	}
	if mean := sum / float64(len(g.Vals)); mean <= 0.03 {
		t.Fatalf("density mean unexpectedly low: %.4f", mean)
	}
}
