package material

import (
	"math"
)

// Color stores linear-light RGB in the range [0, 1] after tone mapping.
type Color struct {
	R float64 `json:"r"`
	G float64 `json:"g"`
	B float64 `json:"b"`
}

func (c Color) Add(o Color) Color     { return Color{c.R + o.R, c.G + o.G, c.B + o.B} }
func (c Color) Scale(s float64) Color { return Color{c.R * s, c.G * s, c.B * s} }

func (c Color) Mul(o Color) Color {
	return Color{c.R * o.R, c.G * o.G, c.B * o.B}
}

func (c Color) Clamp() Color {
	return Color{clamp01(c.R), clamp01(c.G), clamp01(c.B)}
}

// Luminance uses Rec. 709 coefficients on linear light.
func (c Color) Luminance() float64 {
	return 0.2126*c.R + 0.7152*c.G + 0.0722*c.B
}

// DisplayLuma returns Rec. 709 luma of the gamma-encoded bytes.
func (c Color) DisplayLuma() float64 {
	d := c.ToSRGB()
	return 0.2126*d.R + 0.7152*d.G + 0.0722*d.B
}

func linearToSRGBChannel(x float64) float64 {
	x = clamp01(x)
	if x <= 0.0031308 {
		return 12.92 * x
	}
	return 1.055*math.Pow(x, 1/2.4) - 0.055
}

func sRGBToLinearChannel(x float64) float64 {
	x = clamp01(x)
	if x <= 0.04045 {
		return x / 12.92
	}
	return math.Pow((x+0.055)/1.055, 2.4)
}

// ToSRGB applies the sRGB transfer function without changing the container.
func (c Color) ToSRGB() Color {
	return Color{
		R: linearToSRGBChannel(c.R),
		G: linearToSRGBChannel(c.G),
		B: linearToSRGBChannel(c.B),
	}
}

// ToneMap is the single display transform shared by every tier and chart.
func ToneMap(c Color) Color {
	const exposure = 1.08
	x := Color{c.R * exposure, c.G * exposure, c.B * exposure}
	// Stable Reinhard-style shoulder keeps metal highlights readable.
	return Color{
		R: x.R / (1 + x.R),
		G: x.G / (1 + x.G),
		B: x.B / (1 + x.B),
	}
}

// Display applies the shared tone map and sRGB encoding.
func Display(c Color) Color { return ToneMap(c).ToSRGB() }

// HexColor parses #RRGGBB and returns linear RGB.
func HexColor(hex string) (Color, error) {
	if len(hex) != 7 || hex[0] != '#' {
		return Color{}, mathError("color must be #RRGGBB")
	}
	var v [3]float64
	for i := 0; i < 3; i++ {
		hi, err1 := hexNibble(hex[1+i*2])
		lo, err2 := hexNibble(hex[2+i*2])
		if err1 != nil || err2 != nil {
			return Color{}, mathError("color contains invalid hex digit")
		}
		b := hi<<4 | lo
		v[i] = sRGBToLinearChannel(float64(b) / 255)
	}
	return Color{R: v[0], G: v[1], B: v[2]}, nil
}

func hexNibble(b byte) (byte, error) {
	switch {
	case b >= '0' && b <= '9':
		return b - '0', nil
	case b >= 'a' && b <= 'f':
		return b - 'a' + 10, nil
	case b >= 'A' && b <= 'F':
		return b - 'A' + 10, nil
	default:
		return 0, mathError("bad hex digit")
	}
}

type simpleError string

func (e simpleError) Error() string { return string(e) }
func mathError(s string) error      { return simpleError(s) }

// HueDegrees returns cylindrical hue in degrees for an sRGB-visible color.
func HueDegrees(c Color) float64 {
	r, g, b := c.R, c.G, c.B
	mx := math.Max(r, math.Max(g, b))
	mn := math.Min(r, math.Min(g, b))
	d := mx - mn
	if d < 1e-12 {
		return 0
	}
	var h float64
	switch mx {
	case r:
		h = math.Mod((g-b)/d, 6)
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h
}

// HueDelta returns the shortest absolute hue distance in degrees.
func HueDelta(a, b Color) float64 {
	d := math.Abs(HueDegrees(a) - HueDegrees(b))
	if d > 180 {
		d = 360 - d
	}
	return d
}
