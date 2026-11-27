package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

func drawZoomInset(img *image.RGBA, panel Rect, r *SceneResult, mesh *Mesh, baseline bool) {
	zr := Rect{X: panel.X + panel.W - 238, Y: panel.Y + 58, W: 222, H: 214}
	fillRect(img, zr, color.RGBA{8, 12, 19, 248})
	strokeRect(img, zr, colCyan)
	title := "NORMAL-QEF ZOOM"
	titleCol := colGreen
	if baseline {
		title = "BASELINE ZOOM"
		titleCol = colRed
	}
	drawText(img, zr.X+10, zr.Y+9, title, 1, titleCol)

	vr := Rect{X: zr.X + 3, Y: zr.Y + 25, W: zr.W - 6, H: 154}
	target := zoomTarget(r.Scene)
	dist := 0.095
	lookAt := target
	if r.Scene.Kind == KindShell {
		inner := shellCorner(r.Scene.Half.Sub(V(r.Scene.Thickness, r.Scene.Thickness, r.Scene.Thickness)))
		lookAt = target.Add(inner).Mul(0.5)
		dist = 0.28
	}
	cam := newZoomCamera(vr, lookAt, dist)
	depth := make([]float64, vr.W*vr.H)
	renderMesh(img, vr, cam, mesh, depth, baseline)
	drawReferenceFeatures(img, vr, cam, r, depth, true)
	if r.Scene.Kind == KindShell {
		inner := shellCorner(r.Scene.Half.Sub(V(r.Scene.Thickness, r.Scene.Thickness, r.Scene.Thickness)))
		drawLine3D(img, vr, cam, depth, target, inner, color.RGBA{255, 91, 214, 255}, false, true)
	}

	pt, ok := cam.Project(target)
	if ok {
		px, py := int(pt.X), int(pt.Y)
		if vr.Contains(px, py) {
			drawCross(img, px, py)
		}
	}
	if id := nearestVertex(r, target); id >= 0 {
		vp, ok := cam.Project(mesh.Pos[id])
		if ok && vr.Contains(int(vp.X), int(vp.Y)) {
			c := colGreen
			if baseline {
				c = colRed
			}
			drawCrossColor(img, int(vp.X), int(vp.Y), c)
		}
	}

	h := r.Oct.HFinest
	barLen := int(math.Round(cam.Focal * h / dist))
	barX := zr.X + 14
	barY := zr.Y + zr.H - 24
	for x := barX - 1; x <= barX+barLen+1; x++ {
		setPixelSafe(img, x, barY-1, colInk)
		setPixelSafe(img, x, barY+1, colInk)
	}
	for x := barX; x <= barX+barLen; x++ {
		setPixelSafe(img, x, barY, colWhite)
	}
	label := fmt.Sprintf("1 VOXEL H=%.4f", h)
	drawText(img, barX+barLen+10, barY-6, label, 1, colYellow)
	caption := "YELLOW=CREASE CYAN=INNER"
	if r.Scene.Kind == KindShell {
		caption = "MAGENTA=WALL T=0.060"
	} else if r.Scene.Kind == KindSphere {
		caption = "SMOOTH CURVATURE CONTROL"
	}
	drawText(img, zr.X+10, zr.Y+zr.H-12, caption, 1, colMuted)
}

func shellCorner(b Vec3) Vec3 {
	return V(b.X, b.Y, -b.Z)
}

func nearestVertex(r *SceneResult, p Vec3) int {
	best := -1
	bestD := 1e100
	for _, id := range r.UsedIDs {
		d := r.Oct.Leaves[id].Sharp.Sub(p).Norm2()
		if d < bestD {
			bestD = d
			best = id
		}
	}
	return best
}

func zoomTarget(s Scene) Vec3 {
	switch s.Kind {
	case KindSphere:
		return V(0, s.Radius, 0)
	case KindShell:
		b := s.Half
		return V(b.X, b.Y, -b.Z)
	default:
		b := s.Half
		return V(b.X, b.Y, -b.Z)
	}
}

func newZoomCamera(r Rect, target Vec3, dist float64) Camera {
	base := NewCamera(Rect{X: 0, Y: 0, W: 100, H: 100})
	eye := target.Sub(base.F.Mul(dist))
	fovy := 35 * math.Pi / 180
	focal := float64(r.H) / (2 * math.Tan(fovy/2))
	return Camera{
		Eye: eye, Target: target, F: base.F, R: base.R, U: base.U,
		Focal: focal,
		CX:    float64(r.X) + float64(r.W)/2,
		CY:    float64(r.Y) + float64(r.H)/2,
	}
}
