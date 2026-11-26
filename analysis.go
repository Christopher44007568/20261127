package main

import (
	"image/color"
	"math"
)

var (
	bgColor      = color.RGBA{14, 16, 24, 255}
	ghostColor   = color.RGBA{235, 70, 70, 255}  // reprojection lands outside history
	disocclColor = color.RGBA{245, 200, 60, 255} // newly uncovered character pixel
	okTint       = color.RGBA{70, 150, 230, 255}
)

// HistoryView reprojects the previous frame's color buffer through the
// current per-pixel screen velocity. Nearest-neighbor lookup keeps any
// ghost from skinned motion easy to see.
func Reproject(f *Frame, prevColor []color.RGBA, prevCover []bool) []color.RGBA {
	out := make([]color.RGBA, len(f.Color))
	for y := 0; y < f.H; y++ {
		for x := 0; x < f.W; x++ {
			i := f.idx(x, y)
			if f.Hud[i] {
				out[i] = bgColor
				continue
			}
			if !f.Cover[i] {
				// Trailing edge: keep the previous character color so
				// the reprojection panel visibly shows the ghost.
				if prevCover[i] {
					out[i] = prevColor[i]
				} else {
					out[i] = bgColor
				}
				continue
			}
			px := int(float64(x) - f.VelX[i] + 0.5)
			py := int(float64(y) - f.VelY[i] + 0.5)
			if px < 0 || py < 0 || px >= f.W || py >= f.H {
				out[i] = bgColor
				continue
			}
			out[i] = prevColor[py*f.W+px]
		}
	}
	return out
}

// GhostMask classifies non-HUD pixels:
//
//	red    - trailing edge where history still had character color
//	         (the ghost/trail left behind by fast skin motion)
//	yellow - leading edge with no history coverage (disocclusion)
//	blue   - valid history correspondence.
func GhostMask(f *Frame, prevCover []bool) []color.RGBA {
	out := make([]color.RGBA, len(f.Color))
	for i := range out {
		out[i] = bgColor
		if f.Hud[i] {
			continue
		}
		if !f.Cover[i] {
			if prevCover[i] {
				out[i] = ghostColor
			}
			continue
		}
		px := int(float64(i%f.W) - f.VelX[i] + 0.5)
		py := int(float64(i/f.W) - f.VelY[i] + 0.5)
		switch {
		case px < 0 || py < 0 || px >= f.W || py >= f.H:
			out[i] = ghostColor
		case !prevCover[py*f.W+px]:
			out[i] = disocclColor
		default:
			out[i] = okTint
		}
	}
	return out
}

// VelocityImage encodes the screen velocity buffer with the standard
// motion-vector convention: zero speed is neutral gray, R/G hold signed
// X/Y velocity, B carries magnitude. Background is darker than neutral
// and HUD pixels are pure black to prove they hold no velocity.
func VelocityImage(f *Frame) []color.RGBA {
	const peak = 48 // pixels/frame that maps to full saturation
	out := make([]color.RGBA, len(f.Color))
	for i := range out {
		switch {
		case f.Hud[i]:
			out[i] = color.RGBA{0, 0, 0, 255}
		case !f.Cover[i]:
			out[i] = color.RGBA{8, 8, 12, 255}
		default:
			mag := Vec3{f.VelX[i], f.VelY[i], 0}.Len()
			out[i] = color.RGBA{
				R: signedByte(f.VelX[i], peak),
				G: signedByte(f.VelY[i], peak),
				B: clamp8(mag / peak),
				A: 255,
			}
		}
	}
	return out
}

func signedByte(v, peak float64) uint8 {
	return clamp8(0.5 + 0.5*clampSigned(v/peak))
}

// countColor counts pixels equal to c, used for mask statistics.
func countColor(buf []color.RGBA, c color.RGBA) int {
	n := 0
	for _, p := range buf {
		if p == c {
			n++
		}
	}
	return n
}

// RegionSpeed returns the mean velocity magnitude inside a screen-space
// rectangle, over covered non-HUD pixels.
func RegionSpeed(f *Frame, x0, y0, x1, y1 int) float64 {
	var sum float64
	n := 0
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			i := f.idx(x, y)
			if f.Hud[i] || !f.Cover[i] {
				continue
			}
			sum += math.Hypot(f.VelX[i], f.VelY[i])
			n++
		}
	}
	if n == 0 {
		return 0
	}
	return sum / float64(n)
}

func clampSigned(v float64) float64 {
	if v < -1 {
		return -1
	}
	if v > 1 {
		return 1
	}
	return v
}
