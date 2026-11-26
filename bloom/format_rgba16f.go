package bloom

import (
	"encoding/binary"
	"fmt"
)

type rgba16fCodec struct{}

func (rgba16fCodec) Encode(im *Image) (*Texture, *QuantizationReport, error) {
	if err := im.Check(); err != nil {
		return nil, nil, err
	}
	q, err := newQuantizer(FormatRGBA16F, im.Width, im.Height)
	if err != nil {
		return nil, nil, err
	}
	stride := im.Width * 8
	tex := &Texture{
		Width: im.Width, Height: im.Height, Format: FormatRGBA16F,
		Stride: stride, Data: make([]byte, stride*im.Height),
	}
	opaqueAlpha := float64ToHalf(1)
	for i, src := range im.Pixels {
		r := q.clamp(src.R, 0)
		g := q.clamp(src.G, 1)
		b := q.clamp(src.B, 2)
		hr, hg, hb := float64ToHalf(r), float64ToHalf(g), float64ToHalf(b)
		off := (i / im.Width * stride) + (i % im.Width * 8)
		binary.LittleEndian.PutUint16(tex.Data[off:], hr)
		binary.LittleEndian.PutUint16(tex.Data[off+2:], hg)
		binary.LittleEndian.PutUint16(tex.Data[off+4:], hb)
		binary.LittleEndian.PutUint16(tex.Data[off+6:], opaqueAlpha)
		q.addError(src, RGB{
			R: halfToFloat64(hr),
			G: halfToFloat64(hg),
			B: halfToFloat64(hb),
		})
	}
	report := q.report()
	tex.Range = report.Range
	return tex, report, nil
}

func (rgba16fCodec) Decode(tex *Texture) (*Image, error) {
	if err := tex.Check(); err != nil {
		return nil, err
	}
	if tex.Format != FormatRGBA16F {
		return nil, fmt.Errorf("rgba16f codec cannot decode %s", tex.Format)
	}
	im, err := NewImage(tex.Width, tex.Height)
	if err != nil {
		return nil, err
	}
	for y := 0; y < tex.Height; y++ {
		for x := 0; x < tex.Width; x++ {
			off := y*tex.Stride + x*8
			im.Set(x, y, RGB{
				R: halfToFloat64(binary.LittleEndian.Uint16(tex.Data[off:])),
				G: halfToFloat64(binary.LittleEndian.Uint16(tex.Data[off+2:])),
				B: halfToFloat64(binary.LittleEndian.Uint16(tex.Data[off+4:])),
			})
		}
	}
	return im, nil
}
