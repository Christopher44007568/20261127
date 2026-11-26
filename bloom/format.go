package bloom

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
)

type FormatID string

const (
	FormatRGBA16F   FormatID = "rgba16f"
	FormatR11G11B10 FormatID = "r11g11b10"
	FormatSRGB8     FormatID = "srgb8"
)

var ErrUnsupportedFormat = errors.New("unsupported pixel format")

var formatOrder = []FormatID{FormatRGBA16F, FormatR11G11B10, FormatSRGB8}

type ChannelRange struct {
	Min float64 `json:"min"`
	Max float64 `json:"max"`
}

type FormatInfo struct {
	ID            FormatID        `json:"id"`
	Name          string          `json:"name"`
	BytesPerPixel int             `json:"bytes_per_pixel"`
	Samples       int             `json:"samples_per_pixel"`
	Encoding      string          `json:"encoding"`
	Channels      [3]ChannelRange `json:"channels_rgb_target_range"`
}

func AllFormats() []FormatID {
	return append([]FormatID(nil), formatOrder...)
}

func ParseFormat(s string) (FormatID, error) {
	f := FormatID(s)
	if _, err := FormatInfoFor(f); err != nil {
		return "", err
	}
	return f, nil
}

func FormatInfoFor(f FormatID) (FormatInfo, error) {
	switch f {
	case FormatRGBA16F:
		r := ChannelRange{Min: 0, Max: 65504}
		return FormatInfo{
			ID: f, Name: "RGBA16F", BytesPerPixel: 8, Samples: 4,
			Encoding: "IEEE-754 binary16, linear; RGB plus opaque alpha",
			Channels: [3]ChannelRange{r, r, r},
		}, nil
	case FormatR11G11B10:
		return FormatInfo{
			ID: f, Name: "R11G11B10", BytesPerPixel: 4, Samples: 3,
			Encoding: "unsigned 5-bit exponent floats, linear, R/G 6-bit and B 5-bit mantissa",
			Channels: [3]ChannelRange{
				{Min: 0, Max: 65024},
				{Min: 0, Max: 65024},
				{Min: 0, Max: 64512},
			},
		}, nil
	case FormatSRGB8:
		r := ChannelRange{Min: 0, Max: 1}
		return FormatInfo{
			ID: f, Name: "sRGB8", BytesPerPixel: 3, Samples: 3,
			Encoding: "8-bit sRGB transfer, decoded to linear for processing",
			Channels: [3]ChannelRange{r, r, r},
		}, nil
	default:
		return FormatInfo{}, fmt.Errorf("%w: %q", ErrUnsupportedFormat, f)
	}
}

type OverflowChannel struct {
	LowClampCount    uint64  `json:"low_clamp_count"`
	HighClampCount   uint64  `json:"high_clamp_count"`
	CachedLowExcess  float64 `json:"cached_low_excess_sum"`
	CachedHighExcess float64 `json:"cached_high_excess_sum"`
	MaxLowExcess     float64 `json:"max_low_excess"`
	MaxHighExcess    float64 `json:"max_high_excess"`
	NonFiniteCount   uint64  `json:"non_finite_count"`
}

// RangeCache records every value that could not fit in a destination format.
// The excess is kept in the report instead of leaking into a neighboring
// format or silently disappearing.
type RangeCache struct {
	R OverflowChannel `json:"r"`
	G OverflowChannel `json:"g"`
	B OverflowChannel `json:"b"`
}

func (c *OverflowChannel) clamp(v, min, max float64) float64 {
	switch {
	case math.IsNaN(v):
		c.NonFiniteCount++
		return 0
	case math.IsInf(v, 1):
		c.NonFiniteCount++
		return max
	case math.IsInf(v, -1):
		c.NonFiniteCount++
		return min
	}
	if v < min {
		c.LowClampCount++
		excess := min - v
		c.CachedLowExcess += excess
		if excess > c.MaxLowExcess {
			c.MaxLowExcess = excess
		}
		return min
	}
	if v > max {
		c.HighClampCount++
		excess := v - max
		c.CachedHighExcess += excess
		if excess > c.MaxHighExcess {
			c.MaxHighExcess = excess
		}
		return max
	}
	return v
}

func (c *OverflowChannel) merge(o OverflowChannel) {
	c.LowClampCount += o.LowClampCount
	c.HighClampCount += o.HighClampCount
	c.CachedLowExcess += o.CachedLowExcess
	c.CachedHighExcess += o.CachedHighExcess
	c.MaxLowExcess = math.Max(c.MaxLowExcess, o.MaxLowExcess)
	c.MaxHighExcess = math.Max(c.MaxHighExcess, o.MaxHighExcess)
	c.NonFiniteCount += o.NonFiniteCount
}

func (c RangeCache) Merge(o RangeCache) RangeCache {
	c.R.merge(o.R)
	c.G.merge(o.G)
	c.B.merge(o.B)
	return c
}

func (c RangeCache) ClampCount() uint64 {
	return c.R.LowClampCount + c.R.HighClampCount +
		c.G.LowClampCount + c.G.HighClampCount +
		c.B.LowClampCount + c.B.HighClampCount
}

// ErrorStats is computed against the pre-clamp linear value, so clipping
// energy remains visible rather than being reported as a perfect round trip.
type ErrorStats struct {
	MaxAbs            float64 `json:"max_abs"`
	MeanAbs           float64 `json:"mean_abs"`
	RMSE              float64 `json:"rmse"`
	NonFiniteChannels uint64  `json:"non_finite_channel_values"`
}

type QuantizationReport struct {
	Format FormatID   `json:"format"`
	Width  int        `json:"width"`
	Height int        `json:"height"`
	Range  RangeCache `json:"range_cache"`
	Error  ErrorStats `json:"error"`
}

type errorAccumulator struct {
	count     uint64
	nonFinite uint64
	max       float64
	sumAbs    float64
	sumSq     float64
}

func (a *errorAccumulator) add(original, decoded RGB) {
	for _, d := range []float64{
		math.Abs(original.R - decoded.R),
		math.Abs(original.G - decoded.G),
		math.Abs(original.B - decoded.B),
	} {
		if math.IsNaN(d) || math.IsInf(d, 0) {
			a.nonFinite++
			continue
		}
		a.count++
		a.sumAbs += d
		a.sumSq += d * d
		if d > a.max {
			a.max = d
		}
	}
}

func (a *errorAccumulator) stats() ErrorStats {
	if a.count == 0 {
		return ErrorStats{NonFiniteChannels: a.nonFinite}
	}
	n := float64(a.count)
	return ErrorStats{
		MaxAbs:            a.max,
		MeanAbs:           a.sumAbs / n,
		RMSE:              math.Sqrt(a.sumSq / n),
		NonFiniteChannels: a.nonFinite,
	}
}

type quantizer struct {
	format FormatID
	width  int
	height int
	info   FormatInfo
	ranges RangeCache
	errs   errorAccumulator
}

func newQuantizer(f FormatID, width, height int) (*quantizer, error) {
	info, err := FormatInfoFor(f)
	if err != nil {
		return nil, err
	}
	return &quantizer{format: f, width: width, height: height, info: info}, nil
}

func (q *quantizer) clamp(v float64, channel int) float64 {
	r := q.info.Channels[channel]
	switch channel {
	case 0:
		return q.ranges.R.clamp(v, r.Min, r.Max)
	case 1:
		return q.ranges.G.clamp(v, r.Min, r.Max)
	default:
		return q.ranges.B.clamp(v, r.Min, r.Max)
	}
}

func (q *quantizer) addError(original, decoded RGB) {
	q.errs.add(original, decoded)
}

func (q *quantizer) report() *QuantizationReport {
	return &QuantizationReport{
		Format: q.format,
		Width:  q.width,
		Height: q.height,
		Range:  q.ranges,
		Error:  q.errs.stats(),
	}
}

// Texture is a format-specific encoded bitmap. A Texture never aliases or
// combines storage from another intermediate format.
type Texture struct {
	Width  int        `json:"width"`
	Height int        `json:"height"`
	Format FormatID   `json:"format"`
	Stride int        `json:"stride"`
	Data   []byte     `json:"-"`
	Range  RangeCache `json:"range_cache"`
}

func (t *Texture) Check() error {
	if t == nil {
		return fmt.Errorf("nil texture")
	}
	info, err := FormatInfoFor(t.Format)
	if err != nil {
		return err
	}
	n, ok := checkedPixelCount(t.Width, t.Height)
	if !ok {
		return fmt.Errorf("invalid texture dimensions %dx%d", t.Width, t.Height)
	}
	if t.Stride < t.Width*info.BytesPerPixel {
		return fmt.Errorf("texture stride %d is smaller than %dx%d bytes", t.Stride, t.Width, info.BytesPerPixel)
	}
	want := n / t.Width * t.Stride
	if len(t.Data) < want {
		return fmt.Errorf("texture data has %d bytes, want at least %d", len(t.Data), want)
	}
	return nil
}

func (t *Texture) Clone() *Texture {
	out := *t
	out.Data = append([]byte(nil), t.Data...)
	return &out
}

type Codec interface {
	Encode(*Image) (*Texture, *QuantizationReport, error)
	Decode(*Texture) (*Image, error)
}

func CodecFor(f FormatID) (Codec, error) {
	switch f {
	case FormatRGBA16F:
		return rgba16fCodec{}, nil
	case FormatR11G11B10:
		return r11g11b10Codec{}, nil
	case FormatSRGB8:
		return srgb8Codec{}, nil
	default:
		return nil, fmt.Errorf("%w: %q", ErrUnsupportedFormat, f)
	}
}

func EncodeImage(im *Image, f FormatID) (*Texture, *QuantizationReport, error) {
	if err := im.Check(); err != nil {
		return nil, nil, err
	}
	codec, err := CodecFor(f)
	if err != nil {
		return nil, nil, err
	}
	return codec.Encode(im)
}

func DecodeImage(t *Texture) (*Image, error) {
	if err := t.Check(); err != nil {
		return nil, err
	}
	codec, err := CodecFor(t.Format)
	if err != nil {
		return nil, err
	}
	return codec.Decode(t)
}

func MarshalRangeCache(c RangeCache) (json.RawMessage, error) {
	return json.Marshal(c)
}

func SortedFormatIDs(ids []FormatID) []FormatID {
	out := append([]FormatID(nil), ids...)
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}
