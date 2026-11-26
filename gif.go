package main

import (
	"bytes"
	"image"
	"image/color"
	"image/color/palette"
	"image/gif"
)

// quantizer builds a once-per-clip nearest-colour lookup table over the
// limited set of canvas colours. Indexing the LUT per pixel is far cheaper
// than scanning the 256-entry palette for every pixel of every frame.
type quantizer struct {
	pal color.Palette
	lut [64 * 64 * 64]uint8
}

func newQuantizer() *quantizer {
	q := &quantizer{pal: color.Palette(palette.Plan9)}
	for r := 0; r < 64; r++ {
		for g := 0; g < 64; g++ {
			for b := 0; b < 64; b++ {
				q.lut[(r<<12)|(g<<6)|b] = uint8(q.pal.Index(color.RGBA{
					R: uint8(r * 255 / 63),
					G: uint8(g * 255 / 63),
					B: uint8(b * 255 / 63),
					A: 0xff,
				}))
			}
		}
	}
	return q
}

func (q *quantizer) index(r, g, b uint8) uint8 {
	return q.lut[int(r>>2)<<12|int(g>>2)<<6|int(b>>2)]
}

// EncodeGIF converts RGBA frame buffers into a quantised animated GIF using
// only the standard library (Plan 9 palette + a cached nearest-colour LUT).
func EncodeGIF(buffers []*FrameBuffer, delayCentisecs int) ([]byte, error) {
	q := newQuantizer()
	out := &gif.GIF{
		LoopCount: 0,
		Delay:     make([]int, len(buffers)),
		Disposal:  make([]byte, len(buffers)),
	}
	for i, fb := range buffers {
		img := image.NewRGBA(image.Rect(0, 0, fb.W, fb.H))
		copy(img.Pix, fb.Pix)
		pal := image.NewPaletted(img.Rect, palette.Plan9)
		for y := 0; y < fb.H; y++ {
			for x := 0; x < fb.W; x++ {
				off := img.PixOffset(x, y)
				pal.Pix[pal.PixOffset(x, y)] = q.index(
					img.Pix[off], img.Pix[off+1], img.Pix[off+2])
			}
		}
		out.Image = append(out.Image, pal)
		out.Delay[i] = delayCentisecs
		out.Disposal[i] = gif.DisposalBackground
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, out); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
