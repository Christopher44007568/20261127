package bloom

import (
	"fmt"
	"math"
)

// RGB is a linear-light scene color. Values are deliberately not clamped here:
// clamping belongs to the destination pixel format and is reported there.
type RGB struct {
	R float64 `json:"r"`
	G float64 `json:"g"`
	B float64 `json:"b"`
}

func (c RGB) Add(o RGB) RGB {
	return RGB{c.R + o.R, c.G + o.G, c.B + o.B}
}

func (c RGB) Mul(s float64) RGB {
	return RGB{c.R * s, c.G * s, c.B * s}
}

func (c RGB) MaxComponent() float64 {
	return math.Max(c.R, math.Max(c.G, c.B))
}

func (c RGB) Luma() float64 {
	return 0.2126*c.R + 0.7152*c.G + 0.0722*c.B
}

// Image is an unbounded linear RGBA-less image. Alpha is implicit and always 1.
type Image struct {
	Width  int
	Height int
	Pixels []RGB
}

func NewImage(width, height int) (*Image, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("image dimensions must be positive, got %dx%d", width, height)
	}
	n, ok := checkedPixelCount(width, height)
	if !ok {
		return nil, fmt.Errorf("image dimensions are too large: %dx%d", width, height)
	}
	return &Image{Width: width, Height: height, Pixels: make([]RGB, n)}, nil
}

func checkedPixelCount(width, height int) (int, bool) {
	if width <= 0 || height <= 0 {
		return 0, false
	}
	if width > math.MaxInt/height {
		return 0, false
	}
	return width * height, true
}

func (im *Image) Check() error {
	if im == nil {
		return fmt.Errorf("nil image")
	}
	n, ok := checkedPixelCount(im.Width, im.Height)
	if !ok {
		return fmt.Errorf("invalid image dimensions %dx%d", im.Width, im.Height)
	}
	if len(im.Pixels) != n {
		return fmt.Errorf("image %dx%d has %d pixels, want %d", im.Width, im.Height, len(im.Pixels), n)
	}
	return nil
}

func (im *Image) Clone() *Image {
	out := &Image{Width: im.Width, Height: im.Height, Pixels: make([]RGB, len(im.Pixels))}
	copy(out.Pixels, im.Pixels)
	return out
}

func (im *Image) At(x, y int) RGB {
	return im.Pixels[y*im.Width+x]
}

func (im *Image) Set(x, y int, c RGB) {
	im.Pixels[y*im.Width+x] = c
}

func sameDimensions(a, b *Image) error {
	if a.Width != b.Width || a.Height != b.Height {
		return fmt.Errorf("dimension mismatch: %dx%d != %dx%d", a.Width, a.Height, b.Width, b.Height)
	}
	return nil
}

func absDiff(a, b RGB) RGB {
	return RGB{math.Abs(a.R - b.R), math.Abs(a.G - b.G), math.Abs(a.B - b.B)}
}
