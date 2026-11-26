package main

import (
	"image"
	"image/color"
	"math"
	"path/filepath"
)

type demoAssets struct {
	circle *Mask
	star   *Mask
	text   *Mask
}

func buildAssets() demoAssets {
	circleBounds := image.Rect(0, 0, 420, 420)
	starVertices := StarVertices(PointF{X: 210, Y: 210}, 190, 84, 5, -math.Pi/2)
	textBounds := image.Rect(0, 0, 400, 240)
	textWidth := MeasureVectorText("GO", 200)

	return demoAssets{
		circle: CircleMask(circleBounds, PointF{X: 210, Y: 210}, 184, 8),
		star:   PolygonMask(circleBounds, starVertices, 8),
		text:   TextMask(textBounds, "GO", (400-textWidth)/2, 20, 200, 8),
	}
}

func PlaceMask(local *Mask, bounds image.Rectangle, origin image.Point) *Mask {
	full := NewMask(bounds)
	localBounds := local.Bounds()
	for y := localBounds.Min.Y; y < localBounds.Max.Y; y++ {
		for x := localBounds.Min.X; x < localBounds.Max.X; x++ {
			destination := image.Pt(
				origin.X+x-localBounds.Min.X,
				origin.Y+y-localBounds.Min.Y,
			)
			full.setAlpha(destination.X, destination.Y, local.AlphaAt(x, y))
		}
	}
	return full
}

func (c *Canvas) DrawText(text string, left, top, em float64, textColor color.NRGBA, samples int) {
	mask := TextMask(c.Bounds(), text, left, top, em, samples)
	width := MeasureVectorText(text, em)
	area := image.Rect(
		int(math.Floor(left-em*0.1)),
		int(math.Floor(top)),
		int(math.Ceil(left+width+em*0.1)),
		int(math.Ceil(top+em)),
	).Intersect(c.Bounds())
	c.WithClip(mask, ClipReplace, func() {
		c.Fill(area, Solid(textColor))
	})
}

func (c *Canvas) DrawCenteredText(text string, centerX, top, em float64, textColor color.NRGBA) {
	width := MeasureVectorText(text, em)
	c.DrawText(text, float64(centerX)-width/2, top, em, textColor, 4)
}

func RenderDemo(outputDir string) error {
	assets := buildAssets()

	maskFiles := []struct {
		name string
		mask *Mask
		tint color.NRGBA
	}{
		{"circle_mask.png", assets.circle, color.NRGBA{R: 34, G: 211, B: 238, A: 255}},
		{"star_mask.png", assets.star, color.NRGBA{R: 251, G: 191, B: 36, A: 255}},
		{"text_mask.png", assets.text, color.NRGBA{R: 167, G: 139, B: 250, A: 255}},
	}
	for _, item := range maskFiles {
		pngMask := item.mask.Cropped(20)
		if err := SavePNG(filepath.Join(outputDir, item.name), pngMask.Visualize(item.tint)); err != nil {
			return err
		}
		alphaName := "alpha_" + item.name
		if err := SavePNG(filepath.Join(outputDir, alphaName), pngMask.GrayImage()); err != nil {
			return err
		}
	}

	bounds := image.Rect(0, 0, 1200, 940)
	canvas := NewCanvas(bounds)
	operations := renderComposite(canvas, assets)

	if err := SavePNG(filepath.Join(outputDir, "final_composite.png"), canvas.Image()); err != nil {
		return err
	}
	preview := CheckerPreview(canvas.Image(), 20)
	if err := SavePNG(filepath.Join(outputDir, "final_preview_checker.png"), preview); err != nil {
		return err
	}
	if err := RenderTripleSet(outputDir); err != nil {
		return err
	}

	extraMasks := []struct {
		name string
		mask *Mask
		tint color.NRGBA
	}{
		{"inverse_circle_mask.png", operations.inverse, color.NRGBA{R: 244, G: 114, B: 182, A: 255}},
		{"intersection_mask.png", operations.intersection, color.NRGBA{R: 16, G: 185, B: 129, A: 255}},
	}
	for _, item := range extraMasks {
		pngMask := item.mask.Cropped(18)
		if err := SavePNG(filepath.Join(outputDir, item.name), pngMask.Visualize(item.tint)); err != nil {
			return err
		}
		if err := SavePNG(filepath.Join(outputDir, "alpha_"+item.name), pngMask.GrayImage()); err != nil {
			return err
		}
	}
	return nil
}

type operationMasks struct {
	inverse      *Mask
	intersection *Mask
}

func renderComposite(canvas *Canvas, assets demoAssets) operationMasks {
	bounds := canvas.Bounds()
	ink := color.NRGBA{R: 15, G: 23, B: 42, A: 255}

	circleOrigin := image.Pt(10, 54)
	starOrigin := image.Pt(390, 54)
	textOrigin := image.Pt(800, 144)

	circleClip := PlaceMask(assets.circle, bounds, circleOrigin)
	starClip := PlaceMask(assets.star, bounds, starOrigin)
	textClip := PlaceMask(assets.text, bounds, textOrigin)

	canvas.WithClip(circleClip, ClipReplace, func() {
		canvas.Fill(image.Rect(10, 54, 430, 474), DiagonalGradient(
			color.NRGBA{R: 14, G: 165, B: 233, A: 255},
			color.NRGBA{R: 45, G: 212, B: 191, A: 255},
			color.NRGBA{R: 163, G: 230, B: 53, A: 255},
		))
	})

	canvas.WithClip(starClip, ClipReplace, func() {
		canvas.Fill(image.Rect(390, 54, 810, 474), RadialPop(600, 264, 230,
			color.NRGBA{R: 253, G: 224, B: 71, A: 255},
			color.NRGBA{R: 251, G: 146, B: 60, A: 255},
			color.NRGBA{R: 236, G: 72, B: 153, A: 255},
			color.NRGBA{R: 124, G: 58, B: 237, A: 255},
		))
	})

	canvas.WithClip(textClip, ClipReplace, func() {
		canvas.Fill(image.Rect(800, 144, 1200, 384), DiagonalStripes(28,
			color.NRGBA{R: 167, G: 139, B: 250, A: 255},
			color.NRGBA{R: 34, G: 211, B: 238, A: 255},
			color.NRGBA{R: 244, G: 114, B: 182, A: 255},
		))
	})

	canvas.DrawCenteredText("CIRCLE", 220, 488, 34, ink)
	canvas.DrawCenteredText("STAR", 600, 488, 34, ink)
	canvas.DrawCenteredText("TEXT", 980, 488, 34, ink)

	inverseMask := renderInverseDemo(canvas)
	intersectionMask := renderIntersectionDemo(canvas)
	renderNestedDemo(canvas)

	canvas.DrawCenteredText("INVERSE", 220, 866, 30, ink)
	canvas.DrawCenteredText("INTERSECT", 600, 866, 30, ink)
	canvas.DrawCenteredText("NESTED", 1000, 866, 30, ink)
	return operationMasks{inverse: inverseMask, intersection: intersectionMask}
}
