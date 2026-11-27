package main

import "math"

const (
	sphereW, sphereH = 1200, 680
	plateW, plateH   = 1200, 540
	ladderW, ladderH = 1600, 560
)

// renderPanel renders one sub-scene into the canvas at the given rectangle.
func renderPanel(c *Canvas, x0, y0, w, h int, scene *Scene, env *Env, cam *Camera, lut *EnergyLUT, cfg RenderConfig) {
	cam.Width, cam.Height = w, h
	rd := &Renderer{Scene: scene, Env: env, Cam: cam, LUT: lut, Cfg: cfg}
	pix := rd.Render()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			src := (y*w + x) * 3
			dst := ((y+y0)*c.W + (x + x0)) * 3
			c.Pixels[dst+0] = pix[src+0]
			c.Pixels[dst+1] = pix[src+1]
			c.Pixels[dst+2] = pix[src+2]
		}
	}
}

// buildSphereScene renders the 0 vs 90 degree tangent reference image.
func buildSphereScene(lut *EnergyLUT, cfg RenderConfig) *Canvas {
	c := newCanvas(sphereW, sphereH)
	c.rect(0, 0, c.W-1, c.H-1, colorDark)
	panelW, panelH := 540, 520
	panelY := 110
	leftX, rightX := 40, 620

	angles := []float64{0, math.Pi / 2}
	xOrigins := []int{leftX, rightX}
	sunDir := Vec3{-0.35, 0.55, 0.75}
	boxCenter := Vec3{0.1, 0.45, 0.88}
	boxU := Vec3{1, 0, 0.1}
	sphereCfg := cfg
	sphereCfg.SunSamples = 36
	sphereCfg.BoxSamples = 36
	sphereCfg.BlurPasses = 2

	for k, angle := range angles {
		mat := &Material{
			Params: Params{
				Roughness:  0.22,
				Anisotropy: 0.85,
				F0:         colorSteel,
			},
			AngleRad: angle,
		}
		scene := &Scene{Objects: []Object{
			&Sphere{Center: Vec3{0, 0, 0}, Radius: 1.6, Mat: mat},
		}}
		env := makeEnv(sunDir, boxCenter, boxU, 0.55, 0.12)
		cam := newCamera(Vec3{0, 0.1, 5.2}, Vec3{0, 0, 0}, 38*math.Pi/180, panelW, panelH)
		renderPanel(c, xOrigins[k], panelY, panelW, panelH, scene, env, cam, lut, sphereCfg)
	}

	c.frame(leftX, panelY, leftX+panelW-1, panelY+panelH-1, 2, colorFrame)
	c.frame(rightX, panelY, rightX+panelW-1, panelY+panelH-1, 2, colorFrame)
	c.drawText("0°", leftX+panelW/2, 30, 5, colorInk)
	c.drawText("90°", rightX+panelW/2, 30, 5, colorInk)
	return c
}

// buildPlateScene renders brushed plates with horizontal vs vertical fibers.
func buildPlateScene(lut *EnergyLUT, cfg RenderConfig) *Canvas {
	c := newCanvas(plateW, plateH)
	c.rect(0, 0, c.W-1, c.H-1, colorDark)
	panelW, panelH := 540, 400
	panelY := 100
	leftX, rightX := 25, 635

	angles := []float64{0, math.Pi / 2}
	xOrigins := []int{leftX, rightX}
	sunDir := Vec3{-0.45, 0.55, 0.7}
	boxCenter := Vec3{0, 0.45, 0.89}
	boxU := Vec3{1, 0, 0}
	plateCfg := cfg
	plateCfg.SunSamples = 36
	plateCfg.BoxSamples = 36
	plateCfg.BlurPasses = 2

	for k, angle := range angles {
		mat := &Material{
			Params: Params{
				Roughness:  0.42,
				Anisotropy: 0.9,
				F0:         colorAluminum,
			},
			BrushAmp:  0.12,
			BrushFreq: 48,
			BrushAxis: 1, // streaks vary across the fiber direction
			AngleRad:  angle,
		}
		scene := &Scene{Objects: []Object{
			&Plate{
				Center: Vec3{0, 0, 0},
				Half:   Vec3{2.1, 1.35, 0.1},
				Radius: 0.18,
				Mat:    mat,
			},
		}}
		env := makeEnv(sunDir, boxCenter, boxU, 0.95, 0.28)
		env.Sun.AngularRad = 0.07
		env.Sun.Intensity = 1.3
		env.Box.Intensity = 1.1
		cam := newCamera(Vec3{0, 0.35, 4.4}, Vec3{0, 0, 0}, 42*math.Pi/180, panelW, panelH)
		renderPanel(c, xOrigins[k], panelY, panelW, panelH, scene, env, cam, lut, plateCfg)
	}

	c.frame(leftX, panelY, leftX+panelW-1, panelY+panelH-1, 2, colorFrame)
	c.frame(rightX, panelY, rightX+panelW-1, panelY+panelH-1, 2, colorFrame)
	c.drawText("0°", leftX+panelW/2, 30, 5, colorInk)
	c.drawText("90°", rightX+panelW/2, 30, 5, colorInk)

	// Cyan fiber-direction arrows.
	c.arrow(leftX+40, panelY+45, leftX+130, panelY+45, 5, colorCyan)
	c.arrow(rightX+40, panelY+35, rightX+40, panelY+130, 5, colorCyan)
	return c
}

// buildLadderScene renders eight spheres of increasing roughness.
func buildLadderScene(lut *EnergyLUT, cfg RenderConfig) *Canvas {
	c := newCanvas(ladderW, ladderH)
	c.rect(0, 0, c.W-1, c.H-1, colorDark)
	roughs := []float64{0.05, 0.10, 0.17, 0.26, 0.38, 0.52, 0.68, 0.88}
	n := len(roughs)
	radius := 0.62
	spacing := 1.55
	objects := make([]Object, 0, n)
	for i, r := range roughs {
		x := (float64(i) - float64(n-1)/2) * spacing
		mat := &Material{
			Params: Params{
				Roughness:  r,
				Anisotropy: 0.55,
				F0:         colorSteel,
			},
			AngleRad: 0.35,
		}
		objects = append(objects, &Sphere{
			Center: Vec3{x, 0, 0},
			Radius: radius,
			Mat:    mat,
		})
	}
	scene := &Scene{Objects: objects}
	sunDir := Vec3{-0.45, 0.6, 0.7}
	env := makeEnv(sunDir, Vec3{0.05, 0.5, 0.86}, Vec3{1, 0, 0.05}, 0.5, 0.12)
	// Low-angle camera stresses grazing-angle stability.
	cam := newCamera(Vec3{0, 0.35, 7.4}, Vec3{0, 0, 0}, 34*math.Pi/180, ladderW, ladderH)
	ladderCfg := cfg
	ladderCfg.BlurPasses = 1
	rd := &Renderer{Scene: scene, Env: env, Cam: cam, LUT: lut, Cfg: ladderCfg}
	pix := rd.Render()
	copy(c.Pixels, pix)

	// Labels under each sphere.
	projW := cam.viewWidth * 7.4
	for i, r := range roughs {
		worldX := (float64(i) - float64(n-1)/2) * spacing
		screenX := int((worldX/projW + 0.5) * float64(ladderW))
		text := formatRough(r)
		c.drawText(text, screenX, ladderH-70, 3, colorInk)
	}
	return c
}

func formatRough(r float64) string {
	// All ladder values are two-digit decimals (0.05 .. 0.88).
	scaled := int(math.Round(r * 100))
	return string([]rune{
		'0', '.',
		'0' + rune(scaled/10),
		'0' + rune(scaled%10),
	})
}
