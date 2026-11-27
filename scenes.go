package main

import "math"

const (
	renderWidth  = 1600
	renderHeight = 1080
)

func defaultWorld() Rect {
	return Rect{Min: v2(-7, -4.725), Max: v2(7, 4.725)}
}

func RenderPrism(path string) error {
	world := defaultWorld()
	c := NewCanvas(renderWidth, renderHeight, world)
	paintBench(c)

	prism := NewPolygon([]Vec2{
		v2(-1.15, 2.15),
		v2(-2.75, -1.55),
		v2(0.45, -1.55),
	})
	PaintGlassBody(c, prism, 1.18, colCyan)

	screenX := 3.42
	DrawScreen(c, screenX, -4.42, 1.55)

	lambdas := WavelengthSamples(21)
	starts := horizontalStarts(-6.35, 0.52, 0.56, 31)
	DrawBundle(c, starts, v2(1, 0), lambdas, []Shape2{prism}, screenX, world, 0.24)
	c.ResolveBeams()
	PaintGlassRim(c, prism)
	AnnotatePrism(c)

	return c.SavePNG(path)
}

func RenderSphere(path string) error {
	world := defaultWorld()
	c := NewCanvas(renderWidth, renderHeight, world)
	paintBench(c)

	sphere := Circle2{C: v2(-1.35, 0.22), R: 1.42}
	PaintGlassBody(c, sphere, 1.2, colCyan)

	screenX := 1.18
	DrawScreen(c, screenX, -2.35, 2.85)

	lambdas := WavelengthSamples(21)
	starts := horizontalStarts(-6.35, 0.22, 1.78, 39)
	DrawBundle(c, starts, v2(1, 0), lambdas, []Shape2{sphere}, screenX, world, 0.20)
	c.ResolveBeams()
	focusSrc := Rect{Min: v2(-0.05, -0.95), Max: v2(1.30, 1.35)}
	focusDst := PixelRect{X0: 1288, Y0: 128, X1: 1540, Y1: 558}
	c.Connector(focusSrc, focusDst, colCyan)
	c.Magnify(focusSrc, focusDst)
	c.PixelRectBorder(focusDst, colCyan)
	PaintGlassRim(c, sphere)
	AnnotateSphere(c)

	return c.SavePNG(path)
}

func RenderPlate(path string) error {
	world := defaultWorld()
	c := NewCanvas(renderWidth, renderHeight, world)
	paintBench(c)

	plate := rotatedRect(v2(-1.25, 0.0), 1.18, 5.05, 38*math.Pi/180)
	PaintGlassBody(c, plate, 1.35, colCyan)

	screenX := 4.78
	DrawScreen(c, screenX, -2.65, 2.75)

	lambdas := WavelengthSamples(21)
	starts := horizontalStarts(-6.35, -0.10, 1.45, 7)
	DrawBundle(c, starts, v2(1, 0), lambdas, []Shape2{plate}, screenX, world, 0.20)
	c.ResolveBeams()
	edgeSrc := Rect{Min: v2(-1.45, -0.45), Max: v2(-0.10, 0.42)}
	edgeDst := PixelRect{X0: 1108, Y0: 150, X1: 1540, Y1: 429}
	c.Connector(edgeSrc, edgeDst, colCyan)
	c.Magnify(edgeSrc, edgeDst)
	c.PixelRectBorder(edgeDst, colCyan)
	PaintGlassRim(c, plate)
	AnnotatePlate(c)

	return c.SavePNG(path)
}
