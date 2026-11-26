package skinsss

import (
	"math"
	"testing"
)

func approx(a, b, tol float64) bool { return math.Abs(a-b) <= tol }

func vecApprox(a, b Vec3, tol float64) bool {
	return approx(a.X, b.X, tol) && approx(a.Y, b.Y, tol) && approx(a.Z, b.Z, tol)
}

// Beer-Lambert transmittance must be in [0,1], monotone decreasing in depth.
func TestLayerTransmittance(t *testing.T) {
	near := layerTransmittance(SigmaDeep, 0.1)
	far := layerTransmittance(SigmaDeep, 1.5)
	pairs := [][2]float64{
		{near.X, far.X}, {near.Y, far.Y}, {near.Z, far.Z},
	}
	for _, v := range pairs {
		if v[0] <= 0 || v[0] > 1 || v[1] <= 0 || v[1] > 1 {
			t.Fatalf("transmittance out of range: %v %v", near, far)
		}
		if !(v[1] < v[0]) {
			t.Fatalf("transmittance not decreasing: near=%v far=%v", near, far)
		}
	}
}

// A thin chord must transmit more light (and stay brighter) than a thick
// chord under identical geometry and lighting.
func TestThinBrighterThanThick(t *testing.T) {
	base := ShadeInput{
		Normal:     Vec3{0, 0, 1}, // normal toward viewer
		View:       Vec3{0, 0, 1},
		Light:      Vec3{0, 0, -1}, // light behind the subject
		Curvature:  1.0,
		SSSEnabled: true,
	}
	thin := base
	thin.Thickness = 0.25
	thick := base
	thick.Thickness = 1.5

	thinCol := Shade(thin)
	thickCol := Shade(thick)
	if !(thinCol.X > thickCol.X && thinCol.Y > thickCol.Y) {
		t.Fatalf("thin edge not brighter: thin=%v thick=%v", thinCol, thickCol)
	}
}

// All shading outputs stay inside [0,1]: no additive emissive blowout.
func TestShadeBounded(t *testing.T) {
	lights := []Vec3{
		{0.35, 0.45, 1},
		{1, 0.15, 0.25},
		{0, 0, -1},
	}
	for _, l := range lights {
		for _, sss := range []bool{true, false} {
			in := ShadeInput{
				Normal: Vec3{0.3, -0.2, 1}.Normalize(),
				View:   Vec3{0, 0, 1}, Light: l.Normalize(),
				Thickness: 1.0, Curvature: 1.2, SSSEnabled: sss,
			}
			c := Shade(in)
			if c != c.Clamp01() {
				t.Fatalf("unbounded shade output for light %v sss=%v: %v", l, sss, c)
			}
		}
	}
}

// With SSS disabled, a back-facing surface receives only ambient light:
// there must be no transmission or wrapped diffuse leaking in.
func TestDiffuseFallback(t *testing.T) {
	in := ShadeInput{
		Normal: Vec3{0, 0, 1}, View: Vec3{0, 0, 1},
		Light: Vec3{0, 0, -1}, Thickness: 0.3, Curvature: 1.5,
	}
	in.SSSEnabled = false
	got := Shade(in)
	want := SkinAlbedo.Scale(Ambient)
	if !vecApprox(got, want, 1e-12) {
		t.Fatalf("back-facing diffuse = %v, want ambient %v", got, want)
	}
}

// Wrapped diffuse only exists with SSS enabled; the fallback is hard Lambert.
func TestLambertNoWrap(t *testing.T) {
	n := Vec3{0, 0, 1}
	l := Vec3{0.98, 0, 0.2}.Normalize() // grazing, still front-facing
	in := ShadeInput{Normal: n, Light: l, Thickness: 1, Curvature: 1}
	in.SSSEnabled = false
	hard := diffuseIrradiance(in)
	in.SSSEnabled = true
	wrapped := diffuseIrradiance(in)
	if wrapped <= hard {
		t.Fatal("SSS wrap should lift grazing irradiance over hard Lambert")
	}
	// And for a fully back-facing light, hard Lambert stays exactly zero.
	back := ShadeInput{Normal: n, Light: Vec3{0, 0, -1}}
	back.SSSEnabled = false
	if diffuseIrradiance(back) != 0 {
		t.Fatal("hard Lambert must be zero for a back-facing light")
	}
}

func TestHeatmapRange(t *testing.T) {
	for _, x := range []float64{-1, 0, 0.5, 1, 2} {
		c := HeatmapColor(x)
		if c != c.Clamp01() {
			t.Fatalf("heatmap color out of range for %v: %v", x, c)
		}
	}
	if !vecApprox(HeatmapColor(0), heatStops[0].col, 1e-12) {
		t.Fatal("heatmap thin endpoint mismatch")
	}
	if !vecApprox(HeatmapColor(1), heatStops[len(heatStops)-1].col, 1e-12) {
		t.Fatal("heatmap thick endpoint mismatch")
	}
}
