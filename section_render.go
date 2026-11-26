package holefill

import (
	"fmt"
	"math"
)

type sectionView struct {
	xMin, xMax float64
	zMin, zMax float64
	ml, mr     int
	mt, mb     int
	w, h       int
}

func RenderSectionPPM(shell *Mesh, layout SceneLayout, width, height int) *Canvas {
	cv := NewCanvas(width, height, RGB{11, 18, 32})
	segs := BuildSection(shell, layout.SectionY)
	view := sectionView{
		xMin: layout.XMin, xMax: layout.XMax,
		zMin: -0.86, zMax: 0.58,
		ml: 72, mr: 34, mt: 82, mb: 70,
		w: width, h: height,
	}
	drawSectionGrid(cv, view)

	for _, s := range segs {
		col, thick := sectionStyle(s.Tag)
		x1, y1 := view.mapPoint(s.X1, s.Z1)
		x2, y2 := view.mapPoint(s.X2, s.Z2)
		cv.Line(x1, y1, x2, y2, thick+2, RGB{2, 6, 23})
		cv.Line(x1, y1, x2, y2, thick, col)
	}

	drawThicknessMarker(cv, view, segs, -4.55, layout.Thickness)
	DrawText(cv, 28, 24, "THICKNESS SECTION", 3, RGB{248, 250, 252})
	DrawText(cv, 29, 56, fmt.Sprintf("PLANE Y=%.2f  Z EXAG 2.7X", layout.SectionY), 2, RGB{148, 163, 184})

	legendX := width - 360
	DrawLegendRow(cv, legendX, 24, ClassSlit.Color(), "SLIT PATCH", 2)
	DrawLegendRow(cv, legendX, 48, ClassLNotch.Color(), "L NOTCH PATCH", 2)
	DrawLegendRow(cv, legendX, 72, ClassClosePair.Color(), "CLOSE HOLE PATCH", 2)
	DrawLegendRow(cv, 150, height-34, RGB{148, 163, 184}, "SCAN SURFACE", 2)
	DrawLegendRow(cv, 360, height-34, RGB{100, 116, 139}, "NORMAL OFFSET UNDERSIDE", 2)
	return cv
}

func (v sectionView) mapPoint(x, z float64) (int, int) {
	px := float64(v.ml) + (x-v.xMin)/(v.xMax-v.xMin)*float64(v.w-v.ml-v.mr)
	pz := float64(v.mt) + (v.zMax-z)/(v.zMax-v.zMin)*float64(v.h-v.mt-v.mb)
	return int(math.Round(px)), int(math.Round(pz))
}

func drawSectionGrid(cv *Canvas, v sectionView) {
	grid := RGB{30, 41, 59}
	axis := RGB{71, 85, 105}
	for x := int(math.Ceil(v.xMin)); x <= int(math.Floor(v.xMax)); x++ {
		x0, y0 := v.mapPoint(float64(x), v.zMin)
		x1, y1 := v.mapPoint(float64(x), v.zMax)
		col := grid
		if x == 0 {
			col = axis
		}
		cv.Line(x0, y0, x1, y1, 1, col)
		if x%2 == 0 {
			DrawText(cv, x0-8, v.h-50, fmt.Sprintf("%d", x), 1, RGB{100, 116, 139})
		}
	}
	for step := -7; step <= 5; step++ {
		z := float64(step) * 0.1
		x0, y := v.mapPoint(v.xMin, z)
		x1, _ := v.mapPoint(v.xMax, z)
		col := grid
		if step == 0 {
			col = axis
		}
		cv.Line(x0, y, x1, y, 1, col)
	}
}

func sectionStyle(tag FaceTag) (RGB, int) {
	switch tag.Kind {
	case KindPatch:
		return tag.Class.Color(), 4
	case KindBottom:
		return RGB{100, 116, 139}, 2
	case KindWall:
		return RGB{214, 166, 105}, 3
	default:
		return RGB{185, 199, 216}, 3
	}
}

func drawThicknessMarker(cv *Canvas, v sectionView, segs []SectionSegment, x, thickness float64) {
	topZ := sectionZAtX(segs, x, false)
	bottomZ := sectionZAtX(segs, x, true)
	if math.IsNaN(topZ) || math.IsNaN(bottomZ) {
		return
	}
	xt, yTop := v.mapPoint(x, topZ)
	xb, yBottom := v.mapPoint(x, bottomZ)
	for y := yTop; y <= yBottom; y += 6 {
		cv.Line(xt, y, xb, minInt(y+3, yBottom), 2, RGB{254, 240, 138})
	}
	cv.Line(xt-9, yTop, xt+9, yTop, 2, RGB{254, 240, 138})
	cv.Line(xb-9, yBottom, xb+9, yBottom, 2, RGB{254, 240, 138})
	label := fmt.Sprintf("T=%.2f", thickness)
	DrawText(cv, xt+16, (yTop+yBottom)/2-8, label, 2, RGB{254, 240, 138})
	DrawText(cv, xt+16, (yTop+yBottom)/2+14, "NORMAL OFFSET", 1, RGB{253, 224, 71})
}

func sectionZAtX(segs []SectionSegment, x float64, bottom bool) float64 {
	best := math.NaN()
	bestDist := math.Inf(1)
	for _, s := range segs {
		isBottom := s.Tag.Kind == KindBottom
		if isBottom != bottom {
			continue
		}
		if (x < math.Min(s.X1, s.X2)-0.02 || x > math.Max(s.X1, s.X2)+0.02) &&
			math.Abs(s.X1-s.X2) > 1e-6 {
			continue
		}
		z := s.Z1
		if math.Abs(s.X2-s.X1) > 1e-6 {
			t := (x - s.X1) / (s.X2 - s.X1)
			t = math.Max(0, math.Min(1, t))
			z = s.Z1 + (s.Z2-s.Z1)*t
		}
		d := math.Abs((s.X1+s.X2)*0.5 - x)
		if d < bestDist {
			bestDist = d
			best = z
		}
	}
	return best
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
