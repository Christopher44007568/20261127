package bloom

import (
	"math"
)

func CompareImages(reference, test *Image) (ErrorStats, error) {
	if err := reference.Check(); err != nil {
		return ErrorStats{}, err
	}
	if err := test.Check(); err != nil {
		return ErrorStats{}, err
	}
	if err := sameDimensions(reference, test); err != nil {
		return ErrorStats{}, err
	}
	var acc errorAccumulator
	for i := range reference.Pixels {
		acc.add(reference.Pixels[i], test.Pixels[i])
	}
	return acc.stats(), nil
}

func AbsoluteError(reference, test *Image) (*Image, error) {
	if err := reference.Check(); err != nil {
		return nil, err
	}
	if err := test.Check(); err != nil {
		return nil, err
	}
	if err := sameDimensions(reference, test); err != nil {
		return nil, err
	}
	out, err := NewImage(reference.Width, reference.Height)
	if err != nil {
		return nil, err
	}
	for i := range reference.Pixels {
		out.Pixels[i] = absDiff(reference.Pixels[i], test.Pixels[i])
	}
	return out, nil
}

// ToneMapReinhard returns a display-referred SDR color in [0,1].
func ToneMapReinhard(c RGB) RGB {
	mapChannel := func(v float64) float64 {
		if v <= 0 {
			return 0
		}
		if math.IsInf(v, 1) {
			return 1
		}
		return v / (1 + v)
	}
	return RGB{mapChannel(c.R), mapChannel(c.G), mapChannel(c.B)}
}

func HeatColor(t float64) RGB {
	t = math.Max(0, math.Min(1, t))
	stops := [...]struct {
		at float64
		c  RGB
	}{
		{0.00, RGB{0.02, 0.00, 0.06}},
		{0.18, RGB{0.16, 0.02, 0.46}},
		{0.42, RGB{0.72, 0.04, 0.53}},
		{0.68, RGB{1.00, 0.30, 0.10}},
		{0.86, RGB{1.00, 0.82, 0.18}},
		{1.00, RGB{1.00, 1.00, 0.94}},
	}
	for i := 1; i < len(stops); i++ {
		if t <= stops[i].at {
			span := stops[i].at - stops[i-1].at
			f := (t - stops[i-1].at) / span
			a, b := stops[i-1].c, stops[i].c
			return RGB{
				R: a.R + (b.R-a.R)*f,
				G: a.G + (b.G-a.G)*f,
				B: a.B + (b.B-a.B)*f,
			}
		}
	}
	return stops[len(stops)-1].c
}
