package main

import (
	"image"
	"image/color"
)

// Mask is an 8-bit anti-aliased coverage mask. Alpha 0 is a real hole.
type Mask struct {
	alpha *image.Alpha
}

func NewMask(bounds image.Rectangle) *Mask {
	return &Mask{alpha: image.NewAlpha(bounds)}
}

func EmptyMask(bounds image.Rectangle) *Mask {
	return NewMask(bounds)
}

func FullMask(bounds image.Rectangle) *Mask {
	m := NewMask(bounds)
	for i := range m.alpha.Pix {
		m.alpha.Pix[i] = 255
	}
	return m
}

func (m *Mask) Bounds() image.Rectangle {
	return m.alpha.Rect
}

func (m *Mask) CoverageImage() *image.Alpha {
	return m.alpha
}

func (m *Mask) AlphaAt(x, y int) uint8 {
	p := image.Point{X: x, Y: y}
	if !p.In(m.alpha.Rect) {
		return 0
	}
	return m.alpha.Pix[m.alpha.PixOffset(x, y)]
}

func (m *Mask) setAlpha(x, y int, coverage uint8) {
	if image.Pt(x, y).In(m.alpha.Rect) {
		m.alpha.Pix[m.alpha.PixOffset(x, y)] = coverage
	}
}

func (m *Mask) Clone() *Mask {
	clone := NewMask(m.alpha.Rect)
	copy(clone.alpha.Pix, m.alpha.Pix)
	return clone
}

func (m *Mask) ContentBounds() image.Rectangle {
	bounds := m.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y
	found := false
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if m.AlphaAt(x, y) == 0 {
				continue
			}
			found = true
			minX = min(minX, x)
			minY = min(minY, y)
			maxX = max(maxX, x+1)
			maxY = max(maxY, y+1)
		}
	}
	if !found {
		return image.Rect(0, 0, 1, 1)
	}
	return image.Rect(minX, minY, maxX, maxY)
}

// Cropped returns a zero-origin tight mask, convenient for independent PNGs.
func (m *Mask) Cropped(padding int) *Mask {
	bounds := m.Bounds()
	content := m.ContentBounds()
	croppedBounds := image.Rect(
		max(bounds.Min.X, content.Min.X-padding),
		max(bounds.Min.Y, content.Min.Y-padding),
		min(bounds.Max.X, content.Max.X+padding),
		min(bounds.Max.Y, content.Max.Y+padding),
	)
	out := NewMask(image.Rect(0, 0, croppedBounds.Dx(), croppedBounds.Dy()))
	for y := croppedBounds.Min.Y; y < croppedBounds.Max.Y; y++ {
		for x := croppedBounds.Min.X; x < croppedBounds.Max.X; x++ {
			out.setAlpha(x-croppedBounds.Min.X, y-croppedBounds.Min.Y, m.AlphaAt(x, y))
		}
	}
	return out
}

// Inverse returns 255-coverage. Fully covered regions become genuinely empty.
func (m *Mask) Inverse() *Mask {
	out := NewMask(m.alpha.Rect)
	for i, coverage := range m.alpha.Pix {
		out.alpha.Pix[i] = 255 - coverage
	}
	return out
}

// Intersect combines coverage with alpha multiplication, which is the same
// coverage rule used when a source passes through two nested alpha masks.
func Intersect(a, b *Mask) *Mask {
	if a.Bounds() != b.Bounds() {
		panic("Intersect requires equal mask bounds")
	}
	out := NewMask(a.Bounds())
	for i := range out.alpha.Pix {
		av := uint16(a.alpha.Pix[i])
		bv := uint16(b.alpha.Pix[i])
		out.alpha.Pix[i] = uint8((av*bv + 127) / 255)
	}
	return out
}

// Visualize returns fill-colored pixels whose alpha is the mask coverage.
func (m *Mask) Visualize(fill color.NRGBA) *image.NRGBA {
	out := image.NewNRGBA(m.alpha.Rect)
	for y := m.alpha.Rect.Min.Y; y < m.alpha.Rect.Max.Y; y++ {
		for x := m.alpha.Rect.Min.X; x < m.alpha.Rect.Max.X; x++ {
			coverage := m.AlphaAt(x, y)
			i := out.PixOffset(x, y)
			if coverage == 0 {
				continue
			}
			out.Pix[i+0] = fill.R
			out.Pix[i+1] = fill.G
			out.Pix[i+2] = fill.B
			out.Pix[i+3] = coverage
		}
	}
	return out
}

// GrayImage stores coverage as white-on-black, useful as a conventional mask.
func (m *Mask) GrayImage() *image.Gray {
	out := image.NewGray(m.alpha.Rect)
	for i, coverage := range m.alpha.Pix {
		out.Pix[i] = coverage
	}
	return out
}
