package main

import (
	"fmt"
	"os"
)

// RGBA is an 8-bit color. A is only used while compositing offscreen layers.
type RGBA struct {
	R, G, B, A uint8
}

var (
	colTransparent = RGBA{}
	colBlack       = RGBA{0, 0, 0, 255}
	colMagenta     = RGBA{255, 0, 255, 255}
)

func rgb(r, g, b uint8) RGBA {
	return RGBA{r, g, b, 255}
}

// Surface is a small software raster target. It deliberately uses plain slices
// so resource generations can be compared without caches or external libraries.
type Surface struct {
	W, H int
	Pix  []RGBA
}

func NewSurface(w, h int) *Surface {
	if w <= 0 || h <= 0 {
		panic("surface dimensions must be positive")
	}
	return &Surface{W: w, H: h, Pix: make([]RGBA, w*h)}
}

func (s *Surface) Clone() *Surface {
	out := &Surface{W: s.W, H: s.H, Pix: make([]RGBA, len(s.Pix))}
	copy(out.Pix, s.Pix)
	return out
}

func (s *Surface) InBounds(x, y int) bool {
	return x >= 0 && y >= 0 && x < s.W && y < s.H
}

func (s *Surface) At(x, y int) RGBA {
	if !s.InBounds(x, y) {
		return colTransparent
	}
	return s.Pix[y*s.W+x]
}

func (s *Surface) Set(x, y int, c RGBA) {
	if s.InBounds(x, y) {
		s.Pix[y*s.W+x] = c
	}
}

func (s *Surface) Fill(c RGBA) {
	for i := range s.Pix {
		s.Pix[i] = c
	}
}

func (s *Surface) Clear() {
	clear(s.Pix)
}

func blendPixel(dst, src RGBA) RGBA {
	if src.A == 0 {
		return dst
	}
	if src.A == 255 {
		return src
	}
	sa := int(src.A)
	da := int(dst.A)
	daFactor := da * (255 - sa) / 255
	oa := sa + daFactor
	if oa == 0 {
		return colTransparent
	}
	return RGBA{
		R: uint8((int(src.R)*sa + int(dst.R)*daFactor) / oa),
		G: uint8((int(src.G)*sa + int(dst.G)*daFactor) / oa),
		B: uint8((int(src.B)*sa + int(dst.B)*daFactor) / oa),
		A: uint8(oa),
	}
}

func (s *Surface) BlendOver(dst *Surface) {
	for y := 0; y < s.H; y++ {
		for x := 0; x < s.W; x++ {
			dst.Set(x, y, blendPixel(dst.At(x, y), s.At(x, y)))
		}
	}
}

func (s *Surface) DrawSurface(src *Surface, x, y int) {
	for sy := 0; sy < src.H; sy++ {
		for sx := 0; sx < src.W; sx++ {
			s.Set(x+sx, y+sy, src.Pix[sy*src.W+sx])
		}
	}
}

func (s *Surface) FillRect(x, y, w, h int, c RGBA) {
	if w <= 0 || h <= 0 {
		return
	}
	x0, y0 := maxInt(x, 0), maxInt(y, 0)
	x1, y1 := minInt(x+w, s.W), minInt(y+h, s.H)
	for py := y0; py < y1; py++ {
		row := s.Pix[py*s.W:]
		for px := x0; px < x1; px++ {
			row[px] = blendPixel(row[px], c)
		}
	}
}

func (s *Surface) StrokeRect(x, y, w, h, thickness int, c RGBA) {
	if w <= 0 || h <= 0 || thickness <= 0 {
		return
	}
	s.FillRect(x, y, w, thickness, c)
	s.FillRect(x, y+h-thickness, w, thickness, c)
	s.FillRect(x, y+thickness, thickness, h-2*thickness, c)
	s.FillRect(x+w-thickness, y+thickness, thickness, h-2*thickness, c)
}

func (s *Surface) DrawLine(x0, y0, x1, y1 int, c RGBA) {
	dx, dy := absInt(x1-x0), -absInt(y1-y0)
	sx, sy := -1, -1
	if x0 < x1 {
		sx = 1
	}
	if y0 < y1 {
		sy = 1
	}
	err := dx + dy
	for {
		s.Set(x0, y0, blendPixel(s.At(x0, y0), c))
		if x0 == x1 && y0 == y1 {
			break
		}
		e2 := 2 * err
		if e2 >= dy {
			err += dy
			x0 += sx
		}
		if e2 <= dx {
			err += dx
			y0 += sy
		}
	}
}

func (s *Surface) WritePPM(path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}

	if _, err := fmt.Fprintf(f, "P6\n%d %d\n255\n", s.W, s.H); err != nil {
		f.Close()
		return err
	}
	buf := make([]byte, 0, s.W*3)
	for y := 0; y < s.H; y++ {
		buf = buf[:0]
		for x := 0; x < s.W; x++ {
			c := s.Pix[y*s.W+x]
			buf = append(buf, c.R, c.G, c.B)
		}
		if _, err := f.Write(buf); err != nil {
			f.Close()
			return err
		}
	}
	return f.Close()
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}
