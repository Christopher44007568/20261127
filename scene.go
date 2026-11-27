package main

import (
	"image"
	"image/color"
	"math"
)

type wallPanel struct {
	x0, y0, x1, y1 int
}

func (p wallPanel) width() int  { return p.x1 - p.x0 }
func (p wallPanel) height() int { return p.y1 - p.y0 }

func renderScene(tex *StoneTexture, scene Scene) (*image.RGBA, RenderStats) {
	img := image.NewRGBA(image.Rect(0, 0, imageWidth, imageHeight))
	panel := wallPanel{x0: 80, y0: 104, x1: 1120, y1: 728}
	drawBackdrop(img, scene.Kind)
	drawPanelFrame(img, panel)

	stats := RenderStats{Pixels: panel.width() * panel.height()}
	light := sceneLight(scene.Kind)
	layerTotal := 0.0
	for y := panel.y0; y < panel.y1; y++ {
		baseV := (float64(y-panel.y0) + 0.5) / float64(panel.height())
		for x := panel.x0; x < panel.x1; x++ {
			baseU := (float64(x-panel.x0) + 0.5) / float64(panel.width())
			u, v := insetWallUV(scene.Kind, baseU, baseV)
			view := viewForScene(scene.Kind, u, v)
			trace := traceParallax(tex, u, v, view, scene.Kind, scene.Budget, scene.MaxDistance)
			layerTotal += float64(trace.Layers)
			switch trace.Status {
			case TraceHit:
				stats.Hit++
			case TraceBoundary:
				stats.BoundaryFallback++
			case TraceBudget:
				stats.BudgetFallback++
			}
			c := shadeWall(tex, trace, view, light, scene.Kind)
			if trace.Status != TraceHit {
				c = shadeFallback(c, trace.Status, scene.Kind)
			}
			img.SetRGBA(x, y, toRGBA(c))
		}
	}
	stats.AverageLayers = layerTotal / float64(stats.Pixels)
	drawInterface(img, scene, stats, panel)
	return img, stats
}

func insetWallUV(kind ViewKind, u, v float64) (float64, float64) {
	guardU, guardV := 0.035, 0.035
	if kind == ViewGrazing {
		guardU, guardV = 0.285, 0.09
	} else if kind == ViewFallback {
		guardU, guardV = 0.0, 0.0
	}
	return guardU + u*(1-2*guardU), guardV + v*(1-2*guardV)
}

func shadeWall(tex *StoneTexture, trace TraceResult, view, light vec3, kind ViewKind) RGB {
	normalStrength := float32(0.0018)
	if kind == ViewGrazing || kind == ViewFallback {
		normalStrength = 0.0021
	}
	n := normalize(vec3{
		-trace.GradX * normalStrength,
		-trace.GradY * normalStrength,
		1,
	})
	ndotl := clampFloat32((n.dot(light)+0.055)/1.055, 0, 1)
	visibility := viewVisibility(tex, trace.U, trace.V, trace.Height, view)
	cavity := float32(0.68 + 0.32*float64(trace.Height))
	viewLight := 0.60 + 0.40*visibility
	diffuse := ndotl * viewLight

	half := normalize(light.add(view))
	spec := clampFloat32(n.dot(half), 0, 1)
	spec = float32(math.Pow(float64(spec), 30)) * 0.18 * visibility
	ambient := float32(0.33 + 0.08*float64(trace.Height))
	lightLevel := ambient*cavity + 0.74*diffuse
	lightLevel += spec
	grooveOcclusion := 1 - 0.82*(1-float64(visibility))*(1-float64(trace.Height))
	lightLevel *= float32(clamp01(grooveOcclusion))

	c := trace.Color.scale(lightLevel)
	warm := float32(0.020 * float64(diffuse))
	cool := float32(0.018 * (1 - float64(visibility)) * (1 - float64(trace.Height)))
	c.R += warm
	c.G += warm * 0.55
	c.B += cool
	c.R += spec * 0.32
	c.G += spec * 0.28
	c.B += spec * 0.18
	c.R = clampFloat32(c.R, 0, 1)
	c.G = clampFloat32(c.G, 0, 1)
	c.B = clampFloat32(c.B, 0, 1)
	return c
}

func shadeFallback(c RGB, status TraceStatus, kind ViewKind) RGB {
	switch status {
	case TraceBoundary:
		c = c.scale(0.72)
		c.R += 0.012
		c.G += 0.018
		c.B += 0.026
	case TraceBudget:
		if kind == ViewFallback {
			c.R = lerpf(c.R, 0.78, 0.30)
			c.G = lerpf(c.G, 0.24, 0.34)
			c.B = lerpf(c.B, 0.08, 0.38)
		} else {
			c = c.scale(0.86)
			c.R += 0.028
		}
	}
	c.R = clampFloat32(c.R, 0, 1)
	c.G = clampFloat32(c.G, 0, 1)
	c.B = clampFloat32(c.B, 0, 1)
	return c
}

func drawBackdrop(img *image.RGBA, kind ViewKind) {
	centerX, centerY := 0.52, 0.42
	if kind == ViewGrazing {
		centerX = 0.62
	} else if kind == ViewFallback {
		centerX = 0.42
	}
	for y := 0; y < imageHeight; y++ {
		fy := float64(y) / float64(imageHeight-1)
		for x := 0; x < imageWidth; x++ {
			fx := float64(x) / float64(imageWidth-1)
			dx, dy := fx-centerX, fy-centerY
			radius := math.Sqrt(dx*dx*0.72 + dy*dy)
			glow := math.Exp(-radius * 4.2)
			noise := fbm(float64(x)*0.013, float64(y)*0.013, 0x7a11)
			shade := 0.045 + 0.055*(1-fy) + 0.035*glow
			r := shade*1.0 + 0.008*noise
			g := shade*1.03 + 0.010*noise
			b := shade*1.12 + 0.016*noise
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(clamp01(r) * 255),
				G: uint8(clamp01(g) * 255),
				B: uint8(clamp01(b) * 255),
				A: 255,
			})
		}
	}
}

func coverPixel(img *image.RGBA, x, y int, c RGB, alpha float32) {
	if x < 0 || y < 0 || x >= imageWidth || y >= imageHeight || alpha <= 0 {
		return
	}
	dst := img.RGBAAt(x, y)
	a := clamp01(float64(alpha))
	img.SetRGBA(x, y, color.RGBA{
		R: uint8((float64(dst.R)/255*(1-a) + float64(c.R)*a) * 255),
		G: uint8((float64(dst.G)/255*(1-a) + float64(c.G)*a) * 255),
		B: uint8((float64(dst.B)/255*(1-a) + float64(c.B)*a) * 255),
		A: 255,
	})
}

func drawPanelFrame(img *image.RGBA, p wallPanel) {
	for y := p.y0 - 22; y < p.y1+24; y++ {
		for x := p.x0 - 24; x < p.x1+24; x++ {
			if x >= p.x0 && x < p.x1 && y >= p.y0 && y < p.y1 {
				continue
			}
			dx := math.Max(float64(p.x0-x), float64(x-(p.x1-1)))
			dy := math.Max(float64(p.y0-y), float64(y-(p.y1-1)))
			d := math.Max(dx, dy)
			if d < 0 || d > 22 {
				continue
			}
			shadow := math.Exp(-d*0.20) * 0.72
			coverPixel(img, x, y, RGB{0.005, 0.006, 0.007}, float32(shadow))
		}
	}
	for i := 0; i < 4; i++ {
		edge := RGB{0.075, 0.078, 0.077}
		if i == 0 {
			edge = RGB{0.20, 0.20, 0.18}
		}
		drawRectOutline(img, p.x0-i-1, p.y0-i-1, p.x1+i, p.y1+i, edge, 1)
	}
}

func drawRectOutline(img *image.RGBA, x0, y0, x1, y1 int, c RGB, alpha float32) {
	for x := x0; x <= x1; x++ {
		coverPixel(img, x, y0, c, alpha)
		coverPixel(img, x, y1, c, alpha)
	}
	for y := y0; y <= y1; y++ {
		coverPixel(img, x0, y, c, alpha)
		coverPixel(img, x1, y, c, alpha)
	}
}

func toRGBA(c RGB) color.RGBA {
	return color.RGBA{
		R: uint8(clampFloat32(c.R, 0, 1) * 255),
		G: uint8(clampFloat32(c.G, 0, 1) * 255),
		B: uint8(clampFloat32(c.B, 0, 1) * 255),
		A: 255,
	}
}

func clampFloat32(v, lo, hi float32) float32 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
