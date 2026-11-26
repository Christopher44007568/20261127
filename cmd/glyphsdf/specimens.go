package main

import (
	"image"
	"image/color"

	"glyphsdf/internal/field"
	"glyphsdf/internal/fontx"
	"glyphsdf/internal/geom"
	"glyphsdf/internal/render"
	"glyphsdf/internal/shapex"
)

type specimen struct {
	key     string
	label   string
	shape   geom.Shape
	resolve bool
}

type bundle struct {
	spec     specimen
	field    *field.Field
	zoom     float64
	sdfCov   *image.Gray
	msdfCov  *image.Gray
	sdfView  *image.RGBA
	msdfView *image.RGBA
	report   render.Report
}

const heroGrid = 32

func defaultSpecimens() []specimen {
	a := loadFontGlyph('A')
	r := loadFontGlyph('R')
	return []specimen{
		{key: "a", label: "A: SHARP TIP + HOLE", shape: a},
		{key: "r", label: "R: COUNTER + LEG", shape: r},
		{key: "overlap", label: "OVERLAP + CROSSING", shape: shapex.OverlapStar(), resolve: true},
		{key: "comb", label: "DENSE SHARP CORNERS", shape: shapex.CornerComb()},
	}
}

func loadFontGlyph(r rune) geom.Shape {
	s, path, err := fontx.GlyphShapeFromSystem(r)
	if err == nil {
		fontPath = path
		return s
	}
	if r == 'A' {
		return shapex.BuiltinA()
	}
	return shapex.CornerComb()
}

var fontPath string

func makeBundle(spec specimen, zoom float64) bundle {
	shape := shapex.Fit(spec.shape, heroGrid, heroGrid, 4.2, true)
	if spec.resolve {
		shape = geom.ExtractBoundaryShape(shape, 0.035)
	}
	f := field.Generate(shape, field.Options{
		W: heroGrid, H: heroGrid, Range: 4.0, CornerAngle: 0.43, ClampDistance: true,
	})
	field.CorrectSigns(f, 10, 8)

	sdfCov := render.Coverage(f, render.MethodSDF, zoom, 3)
	msdfCov := render.Coverage(f, render.MethodMSDF, zoom, 3)
	paper := color.RGBA{R: 248, G: 248, B: 245, A: 255}
	ink := color.RGBA{R: 18, G: 28, B: 92, A: 255}
	sdfView := render.NewCanvas(sdfCov.Bounds().Dx(), sdfCov.Bounds().Dy(), paper)
	msdfView := render.NewCanvas(msdfCov.Bounds().Dx(), msdfCov.Bounds().Dy(), paper)
	render.DrawCoverageTinted(sdfView, sdfCov, 0, 0, ink, paper)
	render.DrawCoverageTinted(msdfView, msdfCov, 0, 0, ink, paper)

	rep := render.Analyze(f, zoom)
	render.PaintBreaks(sdfView, sdfCov, f, zoom, &rep, render.MethodSDF)
	render.PaintBreaks(msdfView, msdfCov, f, zoom, &rep, render.MethodMSDF)
	return bundle{
		spec: spec, field: f, zoom: zoom,
		sdfCov: sdfCov, msdfCov: msdfCov,
		sdfView: sdfView, msdfView: msdfView, report: rep,
	}
}
