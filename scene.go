package main

import (
	"image/color"
	"math"
	"strconv"
)

const maxTrail = 42

func (r *Renderer) drawScene(fb *FrameBuffer, cam *Camera, centerX int,
	pose Pose, f int, isSource bool) {
	trails := &r.TgtTrails
	if isSource {
		trails = &r.SrcTrails
	}
	for side := Left; side <= Right; side++ {
		idx := int(side)
		trails[idx] = append(trails[idx], pose.Sole[idx])
		if len(trails[idx]) > maxTrail {
			trails[idx] = trails[idx][len(trails[idx])-maxTrail:]
		}
	}

	drawGroundGrid(fb, cam, centerX)
	drawTrails(fb, cam, centerX, *trails)
	drawContactPins(fb, cam, centerX, pose)

	boneCol, jointCol := colTgt, colTgtJoint
	if isSource {
		boneCol, jointCol = colSrc, colSrcJoint
	}
	for _, pair := range bonePairs {
		a, b := pose.Pos[pair[0]], pose.Pos[pair[1]]
		ax, ay := cam.Screen(a, centerX)
		bx, by := cam.Screen(b, centerX)
		fb.Line(ax, ay, bx, by, boneCol)
	}
	for j := 0; j < JointCount; j++ {
		x, y := cam.Screen(pose.Pos[j], centerX)
		fb.Circle(x, y, 2, jointCol)
	}

	// Centre of pressure: a hollow ring sitting on the ground between feet.
	cx, cy := cam.Screen(pose.Cop, centerX)
	fb.Ring(cx, cy, 6, colCop)
	fb.Line(cx-3, cy, cx+3, cy, colCop)

	// Pelvis heading tick makes fast turns readable.
	hip := pose.Pos[JPelvis]
	fwd := v3(math.Cos(pose.Heading), math.Sin(pose.Heading), 0)
	hx, hy := cam.Screen(hip.Add(fwd.Scale(0.22)), centerX)
	px, py := cam.Screen(hip, centerX)
	fb.Line(px, py, hx, hy, colTurn)

	drawSceneLegend(fb, centerX, isSource, pose)
}

func drawGroundGrid(fb *FrameBuffer, cam *Camera, centerX int) {
	// World-fixed grid: lines do not move with the skeleton, so a planted foot
	// and its pin stay aligned while the character travels through the world.
	for gx := -20; gx <= 40; gx += 2 {
		ax, ay := cam.Screen(v3(float64(gx)*0.5, -3, 0), centerX)
		bx, by := cam.Screen(v3(float64(gx)*0.5, 3, 0), centerX)
		fb.Line(ax, ay, bx, by, colGridFar)
	}
	for gy := -8; gy <= 8; gy += 2 {
		ax, ay := cam.Screen(v3(-4, float64(gy)*0.5, 0), centerX)
		bx, by := cam.Screen(v3(8, float64(gy)*0.5, 0), centerX)
		fb.Line(ax, ay, bx, by, colGrid)
	}
}

func drawTrails(fb *FrameBuffer, cam *Camera, centerX int, trails [2][]Vec3) {
	for side := Left; side <= Right; side++ {
		base := colSoleL
		if side == Right {
			base = colSoleR
		}
		points := trails[int(side)]
		for i := 1; i < len(points); i++ {
			alpha := uint8(40 + 200*float64(i)/float64(len(points)))
			c := withAlpha(base, alpha)
			ax, ay := cam.Screen(points[i-1], centerX)
			bx, by := cam.Screen(points[i], centerX)
			fb.Line(ax, ay, bx, by, c)
		}
		if len(points) > 0 {
			px, py := cam.Screen(points[len(points)-1], centerX)
			fb.Circle(px, py, 3, base)
		}
	}
}

func drawContactPins(fb *FrameBuffer, cam *Camera, centerX int, pose Pose) {
	for side := Left; side <= Right; side++ {
		idx := int(side)
		if !pose.Contact[idx] {
			continue
		}
		c := colSoleL
		if side == Right {
			c = colSoleR
		}
		px, py := cam.Screen(pose.Sole[idx], centerX)
		fb.Ring(px, py, 4, c)
	}
}

func drawSceneLegend(fb *FrameBuffer, centerX int, isSource bool, pose Pose) {
	x0 := centerX + 60
	y0 := PanelY0 + 46
	fb.FillRect(x0-8, y0-8, x0+220, y0+44, withAlpha(colBg, 220))
	fb.Circle(x0, y0, 4, colSoleL)
	fb.DrawText(x0+10, y0-5, 1, "L SOLE TRAIL", colTextDim)
	fb.Circle(x0, y0+18, 4, colSoleR)
	fb.DrawText(x0+10, y0+13, 1, "R SOLE TRAIL", colTextDim)
	fb.Ring(x0+120, y0, 6, colCop)
	fb.DrawText(x0+132, y0-5, 1, "COP", colTextDim)
	_ = isSource
	_ = pose
}

func drawHeader(fb *FrameBuffer, f int, turnRate float64) {
	fb.DrawText(20, 12, 2, "SKELETON RETARGET: JOINT MAP + SCALE + FOOT LOCK", colText)
	t := float64(f) * FrameDT
	msg := formatFrameLabel(t, turnRate)
	fb.DrawText(CanvasW-textWidth(msg, 1)-20, 16, 1, msg, colTextDim)
}

func formatFrameLabel(t, turn float64) string {
	return "T=" + floatFixed(t, 2) + "S  TURN=" + floatFixed(turn, 2) + " RAD/F"
}

func floatFixed(v float64, prec int) string {
	return strconv.FormatFloat(v, 'f', prec, 64)
}

var _ = color.Black
