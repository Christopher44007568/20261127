package bloom

import (
	"encoding/binary"
	"fmt"
	"math"
)

type r11g11b10Codec struct{}

const (
	ufloatBias   = 15
	ufloatMaxExp = 30
)

// encodeUnsignedFloat encodes into the shared R11G11B10-style unsigned float.
// The input is already clamped to the channel's representable maximum.
func encodeUnsignedFloat(v float64, mantissaBits int) uint32 {
	if v <= 0 {
		return 0
	}
	minNormal := math.Ldexp(1, 1-ufloatBias)
	if v < minNormal {
		step := math.Ldexp(1, 1-ufloatBias-mantissaBits)
		m := uint32(math.RoundToEven(v / step))
		if m >= uint32(1)<<mantissaBits {
			return 1 << mantissaBits
		}
		return m
	}
	frac, exp := math.Frexp(v)
	expField := exp + ufloatBias - 1
	mantissa := uint32(math.RoundToEven((frac*2 - 1) * float64(uint32(1)<<mantissaBits)))
	if mantissa >= uint32(1)<<mantissaBits {
		mantissa = 0
		expField++
	}
	if expField > ufloatMaxExp {
		expField = ufloatMaxExp
		mantissa = (uint32(1) << mantissaBits) - 1
	}
	return uint32(expField)<<mantissaBits | mantissa
}

func decodeUnsignedFloat(bits uint32, mantissaBits int) float64 {
	mask := (uint32(1) << mantissaBits) - 1
	expField := int(bits >> mantissaBits)
	mantissa := bits & mask
	if expField == 0 {
		return math.Ldexp(float64(mantissa), 1-ufloatBias-mantissaBits)
	}
	if expField == 31 {
		if mantissa == 0 {
			return math.Inf(1)
		}
		return math.NaN()
	}
	return math.Ldexp(1+float64(mantissa)/float64(mask+1), expField-ufloatBias)
}

func (r11g11b10Codec) Encode(im *Image) (*Texture, *QuantizationReport, error) {
	if err := im.Check(); err != nil {
		return nil, nil, err
	}
	q, err := newQuantizer(FormatR11G11B10, im.Width, im.Height)
	if err != nil {
		return nil, nil, err
	}
	stride := im.Width * 4
	tex := &Texture{
		Width: im.Width, Height: im.Height, Format: FormatR11G11B10,
		Stride: stride, Data: make([]byte, stride*im.Height),
	}
	for i, src := range im.Pixels {
		r := q.clamp(src.R, 0)
		g := q.clamp(src.G, 1)
		b := q.clamp(src.B, 2)
		br := encodeUnsignedFloat(r, 6)
		bg := encodeUnsignedFloat(g, 6)
		bb := encodeUnsignedFloat(b, 5)
		packed := br | bg<<11 | bb<<22
		off := (i / im.Width * stride) + (i % im.Width * 4)
		binary.LittleEndian.PutUint32(tex.Data[off:], packed)
		q.addError(src, RGB{
			R: decodeUnsignedFloat(packed&0x7ff, 6),
			G: decodeUnsignedFloat((packed>>11)&0x7ff, 6),
			B: decodeUnsignedFloat((packed>>22)&0x3ff, 5),
		})
	}
	report := q.report()
	tex.Range = report.Range
	return tex, report, nil
}

func (r11g11b10Codec) Decode(tex *Texture) (*Image, error) {
	if err := tex.Check(); err != nil {
		return nil, err
	}
	if tex.Format != FormatR11G11B10 {
		return nil, fmt.Errorf("r11g11b10 codec cannot decode %s", tex.Format)
	}
	im, err := NewImage(tex.Width, tex.Height)
	if err != nil {
		return nil, err
	}
	for y := 0; y < tex.Height; y++ {
		for x := 0; x < tex.Width; x++ {
			off := y*tex.Stride + x*4
			packed := binary.LittleEndian.Uint32(tex.Data[off:])
			im.Set(x, y, RGB{
				R: decodeUnsignedFloat(packed&0x7ff, 6),
				G: decodeUnsignedFloat((packed>>11)&0x7ff, 6),
				B: decodeUnsignedFloat((packed>>22)&0x3ff, 5),
			})
		}
	}
	return im, nil
}
