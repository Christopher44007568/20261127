package render

import (
	"fmt"
	"math"
	"strings"
)

// Render runs the supplied compiled program. A program whose Mode differs
// from settings.ShadowMode is intentionally allowed: that is exactly how the
// stale legacy-cache hit manifests as an incorrectly shaded frame.
func Render(settings Settings, program Program) (*Frame, error) {
	if err := settings.Validate(); err != nil {
		return nil, err
	}
	if program.Mode != ShadowHard && program.Mode != ShadowSoft {
		return nil, fmt.Errorf("invalid program shadow mode %d", program.Mode)
	}

	frame := NewFrame(settings.Width, settings.Height)
	w := float64(settings.Width)
	h := float64(settings.Height)
	aspect := w / h

	for y := 0; y < settings.Height; y++ {
		py := 1 - (float64(y)+0.5)/h*2
		for x := 0; x < settings.Width; x++ {
			px := ((float64(x)+0.5)/w*2 - 1) * aspect
			r, g, b := shadePixel(px, py, program)
			frame.Set(x, y, toByte(r), toByte(g), toByte(b))
		}
	}
	drawModeBadge(frame, program.Mode)
	return frame, nil
}

func RenderDirect(settings Settings) (*Frame, Program, error) {
	program := CompileShader(settings.ShadowMode)
	frame, err := Render(settings, program)
	return frame, program, err
}

func shadePixel(px, py float64, program Program) (float64, float64, float64) {
	deepNavy := [3]float64{0.015, 0.025, 0.060}
	steelBlue := [3]float64{0.105, 0.170, 0.300}
	shadowTint := [3]float64{0.006, 0.010, 0.030}

	t := clamp((py+1)*0.5, 0, 1)
	col := lerpColor(deepNavy, steelBlue, t)

	glowX := px + 0.72
	glowY := py - 0.72
	glow := 0.11 * math.Exp(-(glowX*glowX+glowY*glowY)/0.32)
	col = addColor(col, scaleColor([3]float64{0.45, 0.30, 0.13}, glow))

	shadowDist := shadowEllipseDistance(px, py)
	shadowAlpha := 1 - smoothstep(-program.Penumbra, program.Penumbra, shadowDist)
	col = lerpColor(col, shadowTint, shadowAlpha*0.82)

	sphere := shadeSphere(px, py)
	col = lerpColor(col, sphere.color, sphere.coverage)

	return col[0], col[1], col[2]
}

func shadowEllipseDistance(px, py float64) float64 {
	const (
		cx     = 0.31
		cy     = -0.43
		rx     = 0.66
		ry     = 0.23
		angle  = -0.30
		minRad = 0.23
	)
	dx := px - cx
	dy := py - cy
	c := math.Cos(angle)
	s := math.Sin(angle)
	lx := c*dx + s*dy
	ly := -s*dx + c*dy
	qx := lx / rx
	qy := ly / ry
	q := math.Hypot(qx, qy)
	return (q - 1) * minRad
}

type sphereShade struct {
	color    [3]float64
	coverage float64
}

func shadeSphere(px, py float64) sphereShade {
	const (
		cx = -0.39
		cy = 0.10
		r  = 0.43
	)
	dx := px - cx
	dy := py - cy
	d := length2(dx, dy) - r
	aa := 0.008
	coverage := 1 - smoothstep(-aa, aa, d)
	if coverage <= 0 {
		return sphereShade{coverage: 0}
	}

	nx := dx / r
	ny := dy / r
	nz := math.Sqrt(math.Max(0, 1-nx*nx-ny*ny))
	light := [3]float64{-0.52, 0.74, 0.42}
	invLen := 1 / math.Sqrt(light[0]*light[0]+light[1]*light[1]+light[2]*light[2])
	lx, ly, lz := light[0]*invLen, light[1]*invLen, light[2]*invLen

	diffuse := math.Max(0, nx*lx+ny*ly+nz*lz)
	ambient := 0.16
	base := [3]float64{0.90, 0.42, 0.20}
	col := mulColor(base, ambient+0.84*diffuse)

	hx, hy, hz := lx, ly, lz+1
	hinv := 1 / math.Sqrt(hx*hx+hy*hy+hz*hz)
	nh := math.Max(0, nx*hx*hinv+ny*hy*hinv+nz*hz*hinv)
	spec := math.Pow(nh, 54) * 0.85
	col = addColor(col, [3]float64{spec, spec * 0.88, spec * 0.68})

	rim := 1 - smoothstep(0.0, 0.040, -d)
	col = lerpColor(col, [3]float64{0.10, 0.035, 0.018}, rim*0.62)
	return sphereShade{color: col, coverage: coverage}
}

func scaleColor(c [3]float64, s float64) [3]float64 {
	return [3]float64{c[0] * s, c[1] * s, c[2] * s}
}

var font5x7 = map[byte][7]string{
	' ': {"00000", "00000", "00000", "00000", "00000", "00000", "00000"},
	'A': {"01110", "10001", "10001", "11111", "10001", "10001", "10001"},
	'D': {"11110", "10001", "10001", "10001", "10001", "10001", "11110"},
	'F': {"11111", "10000", "10000", "11110", "10000", "10000", "10000"},
	'H': {"10001", "10001", "10001", "11111", "10001", "10001", "10001"},
	'O': {"01110", "10001", "10001", "10001", "10001", "10001", "01110"},
	'R': {"11110", "10001", "10001", "11110", "10100", "10010", "10001"},
	'S': {"01111", "10000", "10000", "01110", "00001", "00001", "11110"},
	'T': {"11111", "00100", "00100", "00100", "00100", "00100", "00100"},
	'W': {"10001", "10001", "10001", "10101", "10101", "10101", "01010"},
}

func drawModeBadge(frame *Frame, mode ShadowMode) {
	text := "SHADOW " + strings.ToUpper(mode.String())
	const scale = 2
	textWidth := len(text) * 6 * scale
	x := 10
	y := frame.Height - 28
	fillRect(frame, x-4, y-5, textWidth+8, 7*scale+10, 0.02, 0.03, 0.06)
	drawText(frame, x, y, scale, text, 0.92, 0.94, 0.98)
}

func fillRect(frame *Frame, x, y, w, h int, r, g, b float64) {
	for yy := y; yy < y+h; yy++ {
		for xx := x; xx < x+w; xx++ {
			frame.Set(xx, yy, toByte(r), toByte(g), toByte(b))
		}
	}
}

func drawText(frame *Frame, x, y, scale int, text string, r, g, b float64) {
	cursor := x
	for i := 0; i < len(text); i++ {
		rows, ok := font5x7[text[i]]
		if !ok {
			cursor += 6 * scale
			continue
		}
		for ry, row := range rows {
			for rx := 0; rx < 5; rx++ {
				if row[rx] != '1' {
					continue
				}
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						frame.Set(cursor+rx*scale+dx, y+ry*scale+dy, toByte(r), toByte(g), toByte(b))
					}
				}
			}
		}
		cursor += 6 * scale
	}
}
