package studio

import "math"

type point struct {
	X, Y float32
}

type polyline []point

// StrokeFont is a compact geometric display face. It avoids external font
// dependencies while still producing anti-aliased text with a consistent
// stroke weight.
type StrokeFont struct {
	Glyphs  map[rune][]polyline
	Advance float32
	Width   float32
	Height  float32
}

func NewStrokeFont() *StrokeFont {
	return &StrokeFont{
		Glyphs:  glyphs,
		Advance: 5.35,
		Width:   4,
		Height:  6,
	}
}

// TextWidth returns the advance width at the requested cap height.
func (f *StrokeFont) TextWidth(text string, height float32) float32 {
	if len(text) == 0 {
		return 0
	}
	scale := height / f.Height
	runes := []rune(text)
	return float32(len(runes)-1)*f.Advance*scale + f.Width*scale
}

// TextMask renders text into a full-frame single-channel coverage mask.
// The weight is measured in font units; 0.58 gives a medium display stroke.
func (f *StrokeFont) TextMask(w, h int, text string, x, top, height, weight float32) []float32 {
	mask := make([]float32, w*h)
	f.DrawMask(mask, w, h, text, x, top, height, weight)
	return mask
}

func (f *StrokeFont) DrawMask(mask []float32, w, h int, text string, x, top, height, weight float32) {
	if len(mask) != w*h {
		panic("studio: text mask size mismatch")
	}
	scale := height / f.Height
	radius := weight * scale * 0.5
	cursor := x
	for _, r := range text {
		if r == ' ' {
			cursor += f.Advance * scale
			continue
		}
		strokes, ok := f.Glyphs[r]
		if !ok {
			strokes = f.Glyphs['?']
		}
		for _, stroke := range strokes {
			for i := 1; i < len(stroke); i++ {
				a := point{cursor + stroke[i-1].X*scale, top + stroke[i-1].Y*scale}
				b := point{cursor + stroke[i].X*scale, top + stroke[i].Y*scale}
				drawCapsule(mask, w, h, a, b, radius)
			}
		}
		cursor += f.Advance * scale
	}
}

func drawCapsule(mask []float32, w, h int, a, b point, radius float32) {
	if radius <= 0 {
		return
	}
	minX := int(math.Floor(float64(minf(a.X, b.X) - radius - 1)))
	maxX := int(math.Ceil(float64(maxf(a.X, b.X) + radius + 1)))
	minY := int(math.Floor(float64(minf(a.Y, b.Y) - radius - 1)))
	maxY := int(math.Ceil(float64(maxf(a.Y, b.Y) + radius + 1)))
	if minX < 0 {
		minX = 0
	}
	if minY < 0 {
		minY = 0
	}
	if maxX >= w {
		maxX = w - 1
	}
	if maxY >= h {
		maxY = h - 1
	}

	abx, aby := b.X-a.X, b.Y-a.Y
	denom := abx*abx + aby*aby
	for y := minY; y <= maxY; y++ {
		py := float32(y) + 0.5
		for x := minX; x <= maxX; x++ {
			px := float32(x) + 0.5
			t := float32(0)
			if denom > 0 {
				t = ((px-a.X)*abx + (py-a.Y)*aby) / denom
				t = clamp01f(t)
			}
			dx := px - (a.X + abx*t)
			dy := py - (a.Y + aby*t)
			dist := sqrt32(dx*dx + dy*dy)
			cov := clamp01f(radius + 0.5 - dist)
			if cov > mask[y*w+x] {
				mask[y*w+x] = cov
			}
		}
	}
}

func minf(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// DrawLabel is a convenience for the diagnostic sheets.
func DrawLabel(im *FloatImage, font *StrokeFont, text string, x, y, height float32, c RGBA) {
	mask := font.TextMask(im.W, im.H, text, x, y, height, 0.7)
	p := Premultiply(c)
	for i, m := range mask {
		if m <= 0 {
			continue
		}
		j := i * 4
		im.Pix[j] = clamp01f(p.R*m + im.Pix[j]*(1-m))
		im.Pix[j+1] = clamp01f(p.G*m + im.Pix[j+1]*(1-m))
		im.Pix[j+2] = clamp01f(p.B*m + im.Pix[j+2]*(1-m))
		im.Pix[j+3] = clamp01f(p.A*m + im.Pix[j+3]*(1-m))
	}
}
