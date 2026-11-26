package postfx

import "math"

func clamp(value, low, high float64) float64 {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func clampInt(value, low, high int) int {
	if value < low {
		return low
	}
	if value > high {
		return high
	}
	return value
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func max3(a, b, c float64) float64 {
	return maxFloat(maxFloat(a, b), c)
}

func mix(a, b, t float64) float64 {
	return a*(1-t) + b*t
}

func aces(value float64) float64 {
	value = math.Max(0, value)
	return clamp((value*(2.51*value+0.03))/(value*(2.43*value+0.59)+0.14), 0, 1)
}

func drawRect(img *Image, x, y, width, height int, r, g, b float64) {
	for py := y; py < y+height; py++ {
		for px := x; px < x+width; px++ {
			if px < 0 || px >= img.Width || py < 0 || py >= img.Height {
				continue
			}
			img.Set(px, py, r, g, b)
		}
	}
}

func drawCircle(img *Image, cx, cy, radius int, r, g, b float64) {
	radius2 := radius * radius
	for py := cy - radius; py <= cy+radius; py++ {
		for px := cx - radius; px <= cx+radius; px++ {
			dx, dy := px-cx, py-cy
			if dx*dx+dy*dy > radius2 {
				continue
			}
			if px < 0 || px >= img.Width || py < 0 || py >= img.Height {
				continue
			}
			img.Set(px, py, r, g, b)
		}
	}
}
