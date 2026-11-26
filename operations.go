package main

import (
	"image"
	"image/color"
	"math"
)

func strokeCircle(bounds image.Rectangle, center PointF, radius, width float64) *Mask {
	points := ellipseArc(center.X, center.Y, radius, radius, 0, 360, 80)
	return StrokeMask(bounds, []StrokePath{{Points: points, Width: width, Closed: true}}, 4)
}

func strokePolygon(bounds image.Rectangle, points []PointF, width float64) *Mask {
	return StrokeMask(bounds, []StrokePath{{Points: points, Width: width, Closed: true}}, 4)
}

func fillMasked(canvas *Canvas, mask *Mask, mode ClipMode, area image.Rectangle, paint PaintFunc) {
	canvas.WithClip(mask, mode, func() {
		canvas.Fill(area, paint)
	})
}

func renderInverseDemo(canvas *Canvas) *Mask {
	bounds := canvas.Bounds()
	center := PointF{X: 220, Y: 700}
	outer := CircleMask(bounds, center, 142, 8)
	hole := CircleMask(bounds, center, 56, 8)
	area := image.Rect(70, 550, 370, 850)

	canvas.Push(outer, ClipReplace)
	canvas.Fill(area, RadialPop(220, 700, 180,
		color.NRGBA{R: 49, G: 46, B: 129, A: 255},
		color.NRGBA{R: 124, G: 58, B: 237, A: 255},
	))
	// Destination-out proves the inner circle is absent from RGBA, not white.
	canvas.Cut(hole)
	canvas.Push(hole, ClipInverse)
	canvas.Fill(area, DiagonalStripes(24,
		color.NRGBA{R: 250, G: 204, B: 21, A: 255},
		color.NRGBA{R: 244, G: 114, B: 182, A: 255},
	))
	canvas.Pop()
	canvas.Pop()

	outline := strokeCircle(bounds, center, 142, 4)
	fillMasked(canvas, outline, ClipReplace, area, Solid(color.NRGBA{R: 15, G: 23, B: 42, A: 255}))
	return Intersect(outer, hole.Inverse())
}

func renderIntersectionDemo(canvas *Canvas) *Mask {
	bounds := canvas.Bounds()
	center := PointF{X: 600, Y: 700}
	circle := CircleMask(bounds, center, 128, 8)
	starVertices := StarVertices(center, 162, 72, 5, -math.Pi/2)
	star := PolygonMask(bounds, starVertices, 8)
	region := Intersect(circle, star)
	area := image.Rect(430, 530, 770, 870)

	fillMasked(canvas, region, ClipReplace, area, RadialPop(600, 700, 170,
		color.NRGBA{R: 16, G: 185, B: 129, A: 255},
		color.NRGBA{R: 14, G: 116, B: 144, A: 255},
		color.NRGBA{R: 30, G: 64, B: 175, A: 255},
	))

	circleOutline := strokeCircle(bounds, center, 128, 3)
	starOutline := strokePolygon(bounds, starVertices, 3)
	ink := color.NRGBA{R: 15, G: 23, B: 42, A: 220}
	fillMasked(canvas, circleOutline, ClipReplace, area, Solid(ink))
	fillMasked(canvas, starOutline, ClipReplace, area, Solid(ink))
	return region
}

func renderNestedDemo(canvas *Canvas) {
	bounds := canvas.Bounds()
	center := PointF{X: 1000, Y: 700}
	outer := CircleMask(bounds, center, 142, 8)
	innerCenter := PointF{X: 1000, Y: 675}
	innerVertices := StarVertices(innerCenter, 92, 40, 5, -math.Pi/2)
	inner := PolygonMask(bounds, innerVertices, 8)
	area := image.Rect(850, 550, 1150, 850)

	canvas.Push(outer, ClipReplace)
	canvas.Fill(area, Solid(color.NRGBA{R: 186, G: 230, B: 253, A: 255}))

	canvas.Push(inner, ClipIntersect)
	canvas.Fill(image.Rect(900, 580, 1100, 770), DiagonalStripes(22,
		color.NRGBA{R: 244, G: 63, B: 94, A: 255},
		color.NRGBA{R: 251, G: 191, B: 36, A: 255},
	))
	canvas.Pop()

	// These sibling bars are drawn after the inner clip has been popped.
	// Full rectangular bars must remain visible even outside the inner star.
	barColor := color.NRGBA{R: 4, G: 120, B: 87, A: 255}
	for _, y := range []int{730, 760, 790} {
		canvas.Fill(image.Rect(884, y, 1116, y+18), Solid(barColor))
	}
	canvas.Pop()

	outline := strokeCircle(bounds, center, 142, 4)
	fillMasked(canvas, outline, ClipReplace, area, Solid(color.NRGBA{R: 15, G: 23, B: 42, A: 255}))
}
