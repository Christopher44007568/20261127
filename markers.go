package main

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

type screenMarker struct {
	m Marker
	p Projected
}

func drawMarkers(img *image.RGBA, vp Rect, cam Camera, r *SceneResult, depth []float64) {
	var visible []screenMarker
	for _, m := range r.Markers {
		pr, ok := cam.Project(m.Pos)
		if !ok || !vp.Contains(int(pr.X), int(pr.Y)) {
			continue
		}
		px, py := int(pr.X), int(pr.Y)
		di := (py-vp.Y)*vp.W + (px - vp.X)
		if di < 0 || di >= len(depth) {
			continue
		}
		if depth[di] == 0 || depth[di] > pr.W+0.012 {
			continue
		}
		drawCross(img, px, py)
		visible = append(visible, screenMarker{m: m, p: pr})
	}

	const minSep = 150.0
	var labeled []screenMarker
	for _, sm := range visible {
		far := true
		for _, old := range labeled {
			if math.Hypot(sm.p.X-old.p.X, sm.p.Y-old.p.Y) < minSep {
				far = false
				break
			}
		}
		if !far {
			continue
		}
		labeled = append(labeled, sm)
		drawMarkerLabel(img, vp, sm)
		if len(labeled) >= 4 {
			break
		}
	}
}

func drawCross(img *image.RGBA, x, y int) {
	drawCrossColor(img, x, y, colYellow)
}

func drawCrossColor(img *image.RGBA, x, y int, c color.RGBA) {
	for d := -8; d <= 8; d++ {
		setPixelSafe(img, x+d, y, colInk)
		setPixelSafe(img, x, y+d, colInk)
		if absInt(d) <= 5 {
			setPixelSafe(img, x+d, y, c)
			setPixelSafe(img, x, y+d, c)
		}
	}
	for _, d := range [][2]int{{1, 1}, {1, -1}, {-1, 1}, {-1, -1}} {
		setPixelSafe(img, x+d[0]*6, y+d[1]*6, c)
	}
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func drawMarkerLabel(img *image.RGBA, vp Rect, sm screenMarker) {
	px, py := int(sm.p.X), int(sm.p.Y)
	line1 := fmt.Sprintf("E=%.6f", sm.m.Err)
	line2 := fmt.Sprintf("D%d H=%.4f", sm.m.Depth, sm.m.H)
	w1 := textWidth(line1, 1)
	w2 := textWidth(line2, 1)
	boxW := int(math.Max(float64(w1), float64(w2))) + 18
	boxH := 31
	lx := px + 12
	ly := py - boxH - 8
	if lx+boxW > vp.X+vp.W-6 {
		lx = px - boxW - 12
	}
	if ly < vp.Y+6 {
		ly = py + 12
	}
	if ly+boxH > vp.Y+vp.H-6 {
		ly = py - boxH - 12
	}
	rect := Rect{X: lx, Y: ly, W: boxW, H: boxH}
	fillRect(img, rect, colInk)
	strokeRect(img, rect, colYellow)
	drawText(img, lx+8, ly+6, line1, 1, colWhite)
	drawText(img, lx+8, ly+18, line2, 1, colYellow)
}
