package studio

import "math"

// RGBA is a linear-light, straight-alpha colour with components in [0, 1].
// Composite images store a premultiplied version of this value.
type RGBA struct {
	R, G, B, A float32
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func clamp01f(v float32) float32 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

func srgbToLinear(v float64) float64 {
	if v <= 0.04045 {
		return v / 12.92
	}
	return math.Pow((v+0.055)/1.055, 2.4)
}

func linearToSRGB(v float64) float64 {
	v = clamp01(v)
	if v <= 0.0031308 {
		return 12.92 * v
	}
	return 1.055*math.Pow(v, 1/2.4) - 0.055
}

// SRGB converts an sRGB component triplet plus alpha into a linear colour.
func SRGB(r, g, b, a float64) RGBA {
	return RGBA{
		R: float32(srgbToLinear(clamp01(r))),
		G: float32(srgbToLinear(clamp01(g))),
		B: float32(srgbToLinear(clamp01(b))),
		A: float32(clamp01(a)),
	}
}

// Hex parses #rrggbb or #rrggbbaa. Alpha defaults to opaque.
func Hex(s string) RGBA {
	if len(s) > 0 && s[0] == '#' {
		s = s[1:]
	}
	if len(s) != 6 && len(s) != 8 {
		panic("studio: colour must be rrggbb or rrggbbaa")
	}
	parse := func(i int) float64 {
		var v int
		for j := i; j < i+2; j++ {
			c := s[j]
			v <<= 4
			switch {
			case c >= '0' && c <= '9':
				v += int(c - '0')
			case c >= 'a' && c <= 'f':
				v += int(c-'a') + 10
			case c >= 'A' && c <= 'F':
				v += int(c-'A') + 10
			default:
				panic("studio: invalid hex colour")
			}
		}
		return float64(v) / 255
	}
	a := 1.0
	if len(s) == 8 {
		a = parse(6)
	}
	return SRGB(parse(0), parse(2), parse(4), a)
}

// Premultiply converts a straight linear colour to its premultiplied form.
func Premultiply(c RGBA) RGBA {
	a := clamp01f(c.A)
	return RGBA{R: c.R * a, G: c.G * a, B: c.B * a, A: a}
}

// Unpremultiply returns the straight colour represented by p. Fully
// transparent pixels deliberately become black so they never leak a fringe.
func Unpremultiply(p RGBA) RGBA {
	a := clamp01f(p.A)
	if a <= 1e-7 {
		return RGBA{}
	}
	return RGBA{
		R: clamp01f(p.R / a),
		G: clamp01f(p.G / a),
		B: clamp01f(p.B / a),
		A: a,
	}
}

func lerp(a, b, t float32) float32 {
	return a + (b-a)*t
}

func lerpRGBA(a, b RGBA, t float32) RGBA {
	t = clamp01f(t)
	return RGBA{
		R: lerp(a.R, b.R, t),
		G: lerp(a.G, b.G, t),
		B: lerp(a.B, b.B, t),
		A: lerp(a.A, b.A, t),
	}
}

func smoothstep(edge0, edge1, x float32) float32 {
	if edge0 == edge1 {
		if x < edge0 {
			return 0
		}
		return 1
	}
	t := clamp01f((x - edge0) / (edge1 - edge0))
	return t * t * (3 - 2*t)
}

func maxf(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// Luminance is the linear-light Rec. 709 luma.
func Luminance(c RGBA) float32 {
	return 0.2126*c.R + 0.7152*c.G + 0.0722*c.B
}
