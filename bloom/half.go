package bloom

import "math"

const maxHalf = 65504.0

// float64ToHalf converts a non-negative, already-clamped value to binary16.
// Rounding is round-to-nearest, ties-to-even, including the subnormal range.
func float64ToHalf(v float64) uint16 {
	if v <= 0 {
		return 0
	}
	if v >= maxHalf {
		return 0x7bff
	}
	if v < 1.0/16777216.0 {
		return 0
	}
	if v < 1.0/16384.0 {
		m := uint32(math.RoundToEven(v * 16777216.0))
		if m >= 1024 {
			return 1 << 10
		}
		return uint16(m)
	}

	frac, exp := math.Frexp(v)
	expField := exp + 14
	mantissa := uint32(math.RoundToEven((frac*2 - 1) * 1024))
	if mantissa == 1024 {
		mantissa = 0
		expField++
	}
	if expField >= 31 {
		return 0x7bff
	}
	return uint16(expField<<10) | uint16(mantissa)
}

func halfToFloat64(h uint16) float64 {
	exp := int(h>>10) & 0x1f
	mantissa := int(h & 0x03ff)
	sign := 1.0
	if h&0x8000 != 0 {
		sign = -1
	}
	switch exp {
	case 0:
		return sign * math.Ldexp(float64(mantissa), -24)
	case 31:
		if mantissa == 0 {
			return sign * math.Inf(1)
		}
		return math.NaN()
	default:
		return sign * math.Ldexp(float64(1024+mantissa), exp-25)
	}
}
