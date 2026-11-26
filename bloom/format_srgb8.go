package bloom

import (
	"fmt"
	"math"
)

type srgb8Codec struct{}

func linearToSRGB(v float64) float64 {
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1.0/2.4) - 0.055
}

func srgbToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func quantizeSRGB8(v float64) uint8 {
	encoded := linearToSRGB(v)
	q := int(math.RoundToEven(encoded * 255))
	if q < 0 {
		return 0
	}
	if q > 255 {
		return 255
	}
	return uint8(q)
}

func decodeSRGB8(v uint8) float64 {
	return srgbToLinear(float64(v) / 255)
}

func (srgb8Codec) Encode(im *Image) (*Texture, *QuantizationReport, error) {
	if err := im.Check(); err != nil {
		return nil, nil, err
	}
	q, err := newQuantizer(FormatSRGB8, im.Width, im.Height)
	if err != nil {
		return nil, nil, err
	}
	stride := im.Width * 3
	tex := &Texture{
		Width: im.Width, Height: im.Height, Format: FormatSRGB8,
		Stride: stride, Data: make([]byte, stride*im.Height),
	}
	for i, src := range im.Pixels {
		r := q.clamp(src.R, 0)
		g := q.clamp(src.G, 1)
		b := q.clamp(src.B, 2)
		qr, qg, qb := quantizeSRGB8(r), quantizeSRGB8(g), quantizeSRGB8(b)
		off := (i / im.Width * stride) + (i % im.Width * 3)
		tex.Data[off], tex.Data[off+1], tex.Data[off+2] = qr, qg, qb
		q.addError(src, RGB{
			R: decodeSRGB8(qr),
			G: decodeSRGB8(qg),
			B: decodeSRGB8(qb),
		})
	}
	report := q.report()
	tex.Range = report.Range
	return tex, report, nil
}

func (srgb8Codec) Decode(tex *Texture) (*Image, error) {
	if err := tex.Check(); err != nil {
		return nil, err
	}
	if tex.Format != FormatSRGB8 {
		return nil, fmt.Errorf("srgb8 codec cannot decode %s", tex.Format)
	}
	im, err := NewImage(tex.Width, tex.Height)
	if err != nil {
		return nil, err
	}
	for y := 0; y < tex.Height; y++ {
		for x := 0; x < tex.Width; x++ {
			off := y*tex.Stride + x*3
			im.Set(x, y, RGB{
				R: decodeSRGB8(tex.Data[off]),
				G: decodeSRGB8(tex.Data[off+1]),
				B: decodeSRGB8(tex.Data[off+2]),
			})
		}
	}
	return im, nil
}
