package report

import "math"

type plot struct {
	x, y, w, h float64
	minA, maxA float64
	minB, maxB float64
	scale      float64
	offA       float64
	offB       float64
}

func newPlot(x, y, w, h, minA, maxA, minB, maxB float64) plot {
	rangeA := maxA - minA
	rangeB := maxB - minB
	scale := math.Min(w/rangeA, h/rangeB)
	drawW := rangeA * scale
	drawH := rangeB * scale
	return plot{
		x: x + (w-drawW)/2, y: y + (h-drawH)/2, w: drawW, h: drawH,
		minA: minA, maxA: maxA, minB: minB, maxB: maxB, scale: scale,
	}
}

func (p plot) mapPoint(a, b float64) (float64, float64) {
	x := p.x + (a-p.minA)*p.scale
	y := p.y + p.h - (b-p.minB)*p.scale
	return x, y
}

func (p plot) rect(a0, b0, a1, b1 float64) (x, y, w, h float64) {
	x0, y0 := p.mapPoint(a0, b0)
	x1, y1 := p.mapPoint(a1, b1)
	x = math.Min(x0, x1)
	y = math.Min(y0, y1)
	return x, y, math.Abs(x1 - x0), math.Abs(y1 - y0)
}

func depthColor(depth int) string {
	colors := []string{"#ea580c", "#d97706", "#16a34a", "#2563eb", "#7c3aed", "#0891b2"}
	if depth < 0 {
		return colors[0]
	}
	if depth >= len(colors) {
		return colors[len(colors)-1]
	}
	return colors[depth]
}

func poseTitle(name string) string {
	switch name {
	case "squat":
		return "深蹲"
	case "push_off":
		return "蹬地"
	case "tail_swing":
		return "摆尾"
	default:
		return name
	}
}

func englishPoseTitle(name string) string {
	switch name {
	case "squat":
		return "Squat"
	case "push_off":
		return "Push-off"
	case "tail_swing":
		return "Tail swing"
	default:
		return name
	}
}
