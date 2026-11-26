package main

import (
	"image"

	"glyphsdf/internal/field"
	"glyphsdf/internal/render"
	"glyphsdf/internal/shapex"
)

func buildZoomSheet() (*image.RGBA, error) {
	shape := shapex.Fit(loadFontGlyph('A'), heroGrid, heroGrid, 4.2, true)
	f := field.Generate(shape, field.Options{
		W: heroGrid, H: heroGrid, Range: 4.0, CornerAngle: 0.43, ClampDistance: true,
	})
	field.CorrectSigns(f, 10, 8)

	scales := []int{3, 6, 12, 24}
	width := 1900
	height := 150
	for _, s := range scales {
		height += heroGrid*s + 42
	}
	img := render.NewCanvas(width, height, paperColor)
	render.DrawText(img, 44, 34, "ZOOM LEVELS: SDF ROUNDS / MSDF KEEPS TIPS", 5, inkColor)
	render.DrawText(img, 210, 96, "SDF + RED BREAK MARKS", 3, breakRedText())
	render.DrawText(img, 1030, 96, "MSDF MEDIAN + VECTOR CHECK", 3, inkColor)

	y := 140
	for _, scale := range scales {
		z := float64(scale)
		size := heroGrid * scale
		sdfCov := render.Coverage(f, render.MethodSDF, z, 2)
		msdfCov := render.Coverage(f, render.MethodMSDF, z, 2)
		sdfView := render.NewCanvas(size, size, paperColor)
		msdfView := render.NewCanvas(size, size, paperColor)
		render.DrawCoverageTinted(sdfView, sdfCov, 0, 0, blueInk, paperColor)
		render.DrawCoverageTinted(msdfView, msdfCov, 0, 0, blueInk, paperColor)
		rep := render.Analyze(f, z)
		render.PaintBreaks(sdfView, sdfCov, f, z, &rep, render.MethodSDF)
		render.PaintBreaks(msdfView, msdfCov, f, z, &rep, render.MethodMSDF)

		label := itoa(scale) + "X"
		render.DrawText(img, 54, y+24, label, 3, inkColor)
		summary := "S" + itoa(rep.SDFBrokenCount) + " / M" + itoa(rep.MSDFBrokenCount)
		render.DrawText(img, 44, y+64, summary, 2, mutedColor)
		render.DrawImage(img, sdfView, 210, y)
		render.DrawImage(img, msdfView, 1030, y)
		y += size + 42
	}
	_ = image.Point{}
	return img, nil
}
