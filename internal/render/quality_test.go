package render

import (
	"testing"

	"glyphsdf/internal/field"
	"glyphsdf/internal/geom"
	"glyphsdf/internal/shapex"
)

func TestMSDFPreservesSharpSpecimens(t *testing.T) {
	specs := []struct {
		name    string
		shape   geom.Shape
		resolve bool
		minSDF  int
	}{
		{name: "overlap-star", shape: shapex.OverlapStar(), resolve: true, minSDF: 4},
		{name: "corner-comb", shape: shapex.CornerComb(), minSDF: 5},
	}
	for _, spec := range specs {
		t.Run(spec.name, func(t *testing.T) {
			shape := shapex.Fit(spec.shape, 32, 32, 4.2, true)
			if spec.resolve {
				shape = geom.ExtractBoundaryShape(shape, 0.035)
			}
			f := field.Generate(shape, field.Options{
				W: 32, H: 32, Range: 4, CornerAngle: 0.43,
			})
			field.CorrectSigns(f, 10, 8)
			rep := Analyze(f, 12)
			if rep.MSDFBrokenCount != 0 {
				t.Fatalf("MSDF has %d broken corners: %+v", rep.MSDFBrokenCount, rep.Corners)
			}
			if rep.SDFBrokenCount < spec.minSDF {
				t.Fatalf("SDF break count = %d, want at least %d", rep.SDFBrokenCount, spec.minSDF)
			}
		})
	}
}
