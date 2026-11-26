package main

import (
	"image"
	"image/color"
	"math"
	"path/filepath"
)

func RenderTripleSet(outputDir string) error {
	bounds := image.Rect(0, 0, 900, 680)
	center := PointF{X: 450, Y: 330}
	circle := CircleMask(bounds, center, 280, 8)
	star := PolygonMask(bounds, StarVertices(center, 268, 118, 5, -math.Pi/2), 8)
	textWidth := MeasureVectorText("GO", 300)
	text := TextMask(bounds, "GO", (900-textWidth)/2, 176, 300, 8)

	overlay := NewCanvas(bounds)
	overlay.WithClip(circle, ClipReplace, func() {
		overlay.Fill(bounds, DiagonalGradient(
			color.NRGBA{R: 8, G: 145, B: 178, A: 255},
			color.NRGBA{R: 34, G: 211, B: 238, A: 255},
			color.NRGBA{R: 163, G: 230, B: 53, A: 255},
		))
	})
	overlay.WithClip(star, ClipReplace, func() {
		overlay.Fill(bounds, RadialPop(450, 320, 310,
			color.NRGBA{R: 254, G: 240, B: 138, A: 255},
			color.NRGBA{R: 251, G: 146, B: 60, A: 245},
			color.NRGBA{R: 219, G: 39, B: 119, A: 245},
		))
	})
	overlay.WithClip(text, ClipReplace, func() {
		overlay.Fill(bounds, DiagonalStripes(38,
			color.NRGBA{R: 255, G: 255, B: 255, A: 255},
			color.NRGBA{R: 167, G: 139, B: 250, A: 255},
			color.NRGBA{R: 34, G: 211, B: 238, A: 255},
		))
	})

	for _, item := range []struct {
		name string
		img  image.Image
	}{
		{"triple_overlay.png", overlay.Image()},
		{"triple_overlay_preview.png", CheckerPreview(overlay.Image(), 22)},
	} {
		if err := SavePNG(filepath.Join(outputDir, item.name), item.img); err != nil {
			return err
		}
	}

	stacked := NewCanvas(bounds)
	stacked.Push(circle, ClipIntersect)
	stacked.Push(star, ClipIntersect)
	stacked.Push(text, ClipIntersect)
	stacked.Fill(bounds, DiagonalGradient(
		color.NRGBA{R: 124, G: 58, B: 237, A: 255},
		color.NRGBA{R: 236, G: 72, B: 153, A: 255},
		color.NRGBA{R: 251, G: 191, B: 36, A: 255},
	))
	stacked.Pop()
	stacked.Pop()
	stacked.Pop()

	if err := SavePNG(filepath.Join(outputDir, "triple_clip_stack.png"), stacked.Image()); err != nil {
		return err
	}
	if err := SavePNG(filepath.Join(outputDir, "triple_clip_stack_preview.png"), CheckerPreview(stacked.Image(), 22)); err != nil {
		return err
	}

	combined := Intersect(Intersect(circle, star), text)
	if err := SavePNG(
		filepath.Join(outputDir, "triple_intersection_mask.png"),
		combined.Cropped(24).Visualize(color.NRGBA{R: 217, G: 70, B: 239, A: 255}),
	); err != nil {
		return err
	}
	return nil
}
