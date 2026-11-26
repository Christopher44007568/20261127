package bloom

import (
	"encoding/json"
	"math"
	"testing"
)

func TestHalfKnownValues(t *testing.T) {
	cases := []struct {
		value float64
		bits  uint16
	}{
		{0, 0x0000},
		{0.5, 0x3800},
		{1, 0x3c00},
		{2, 0x4000},
		{65504, 0x7bff},
	}
	for _, tc := range cases {
		if got := float64ToHalf(tc.value); got != tc.bits {
			t.Fatalf("float64ToHalf(%g) = 0x%04x, want 0x%04x", tc.value, got, tc.bits)
		}
		if got := halfToFloat64(tc.bits); got != tc.value {
			t.Fatalf("halfToFloat64(0x%04x) = %g, want %g", tc.bits, got, tc.value)
		}
	}
}

func TestRGBA16FRangeAndAlpha(t *testing.T) {
	im, err := NewImage(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	im.Pixels[0] = RGB{70000, -2, 0.25}
	tex, report, err := EncodeImage(im, FormatRGBA16F)
	if err != nil {
		t.Fatal(err)
	}
	if report.Range.R.HighClampCount != 1 || report.Range.G.LowClampCount != 1 {
		t.Fatalf("unexpected range report: %+v", report.Range)
	}
	if report.Range.R.MaxHighExcess != 70000-65504 {
		t.Fatalf("cached R excess = %g, want %g", report.Range.R.MaxHighExcess, 70000-65504.0)
	}
	alphaBits := uint16(tex.Data[6]) | uint16(tex.Data[7])<<8
	if alphaBits != 0x3c00 {
		t.Fatalf("alpha half bits = 0x%04x, want 0x3c00", alphaBits)
	}
	decoded, err := DecodeImage(tex)
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Pixels[0].R != 65504 || decoded.Pixels[0].G != 0 || decoded.Pixels[0].B != 0.25 {
		t.Fatalf("unexpected decoded pixel: %+v", decoded.Pixels[0])
	}
}

func TestR11G11B10PrecisionAndLimits(t *testing.T) {
	im, err := NewImage(2, 1)
	if err != nil {
		t.Fatal(err)
	}
	im.Pixels[0] = RGB{1, 2, 0.5}
	im.Pixels[1] = RGB{70000, 70000, 70000}
	tex, report, err := EncodeImage(im, FormatR11G11B10)
	if err != nil {
		t.Fatal(err)
	}
	if tex.Format != FormatR11G11B10 || len(tex.Data) != 8 {
		t.Fatalf("unexpected texture: format=%s bytes=%d", tex.Format, len(tex.Data))
	}
	if report.Range.R.HighClampCount != 1 || report.Range.G.HighClampCount != 1 ||
		report.Range.B.HighClampCount != 1 {
		t.Fatalf("unexpected range report: %+v", report.Range)
	}
	decoded, err := DecodeImage(tex)
	if err != nil {
		t.Fatal(err)
	}
	for channel, pair := range [][2]float64{
		{decoded.Pixels[0].R, 1},
		{decoded.Pixels[0].G, 2},
		{decoded.Pixels[0].B, 0.5},
	} {
		rel := math.Abs(pair[0]-pair[1]) / pair[1]
		if rel > 1.0/64.0 {
			t.Fatalf("channel %d relative error %g is too high", channel, rel)
		}
	}
	if decoded.Pixels[1].R != 65024 || decoded.Pixels[1].G != 65024 || decoded.Pixels[1].B != 64512 {
		t.Fatalf("unexpected clamped maxima: %+v", decoded.Pixels[1])
	}
	if _, err := json.Marshal(report); err != nil {
		t.Fatalf("range report is not JSON-safe: %v", err)
	}
}

func TestR11G11B10KnownPacking(t *testing.T) {
	r := encodeUnsignedFloat(1, 6)
	g := encodeUnsignedFloat(1, 6)
	b := encodeUnsignedFloat(1, 5)
	packed := r | g<<11 | b<<22
	if packed != 0x781e03c0 {
		t.Fatalf("packed 1,1,1 = 0x%08x, want 0x781e03c0", packed)
	}
	if encodeUnsignedFloat(65024, 6) != 0x7bf || encodeUnsignedFloat(64512, 5) != 0x3df {
		t.Fatal("R11G11B10 maximum finite encodings do not match the format")
	}
}

func TestSRGB8ClampAndRoundTrip(t *testing.T) {
	im, err := NewImage(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	im.Pixels[0] = RGB{0.5, 2.0, -1.0}
	tex, report, err := EncodeImage(im, FormatSRGB8)
	if err != nil {
		t.Fatal(err)
	}
	if report.Range.G.HighClampCount != 1 || report.Range.B.LowClampCount != 1 {
		t.Fatalf("unexpected range report: %+v", report.Range)
	}
	decoded, err := DecodeImage(tex)
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(decoded.Pixels[0].R-0.5) > 0.004 {
		t.Fatalf("sRGB midtone error too high: %g", decoded.Pixels[0].R)
	}
	if decoded.Pixels[0].G != 1 || decoded.Pixels[0].B != 0 {
		t.Fatalf("unexpected clamped sRGB values: %+v", decoded.Pixels[0])
	}
}

func TestNonFiniteIsReportedAndJSONSafe(t *testing.T) {
	im, err := NewImage(1, 1)
	if err != nil {
		t.Fatal(err)
	}
	im.Pixels[0] = RGB{math.Inf(1), math.NaN(), math.Inf(-1)}
	_, report, err := EncodeImage(im, FormatRGBA16F)
	if err != nil {
		t.Fatal(err)
	}
	if report.Range.R.NonFiniteCount != 1 || report.Range.G.NonFiniteCount != 1 ||
		report.Range.B.NonFiniteCount != 1 {
		t.Fatalf("non-finite values were not cached in the report: %+v", report.Range)
	}
	if report.Error.NonFiniteChannels != 3 {
		t.Fatalf("non-finite error channel count = %d, want 3", report.Error.NonFiniteChannels)
	}
	if _, err := json.Marshal(report); err != nil {
		t.Fatalf("report is not JSON-safe: %v", err)
	}
}

func TestEncodedFormatsHaveIndependentStorage(t *testing.T) {
	im, err := NewImage(2, 2)
	if err != nil {
		t.Fatal(err)
	}
	for i := range im.Pixels {
		im.Pixels[i] = RGB{0.25, 0.5, 1.0}
	}
	var textures []*Texture
	for _, format := range AllFormats() {
		tex, _, err := EncodeImage(im, format)
		if err != nil {
			t.Fatal(err)
		}
		textures = append(textures, tex)
	}
	for i, tex := range textures {
		for j, other := range textures {
			if i != j && len(tex.Data) > 0 && &tex.Data[0] == &other.Data[0] {
				t.Fatalf("%s and %s share a bitmap buffer", tex.Format, other.Format)
			}
		}
	}
	textures[0].Data[0] ^= 0xff
	if textures[1].Data[0] == textures[0].Data[0] {
		t.Fatal("mutating one format changed another format's bitmap")
	}
}
