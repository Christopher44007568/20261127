package main

import (
	"image"
	"image/color"
)

type ClipMode int

const (
	// ClipIntersect keeps only coverage allowed by both parent and mask.
	ClipIntersect ClipMode = iota
	// ClipInverse keeps the parent region outside the new mask.
	ClipInverse
	// ClipReplace ignores the parent clip; useful for independent layers.
	ClipReplace
)

type Canvas struct {
	img       *image.NRGBA
	clipStack []*Mask
}

func NewCanvas(bounds image.Rectangle) *Canvas {
	return &Canvas{img: image.NewNRGBA(bounds)}
}

func (c *Canvas) Image() *image.NRGBA {
	return c.img
}

func (c *Canvas) Bounds() image.Rectangle {
	return c.img.Rect
}

func (c *Canvas) currentClip() *Mask {
	if len(c.clipStack) == 0 {
		return nil
	}
	return c.clipStack[len(c.clipStack)-1]
}

func (c *Canvas) Push(mask *Mask, mode ClipMode) {
	var next *Mask
	parent := c.currentClip()

	switch mode {
	case ClipReplace:
		next = mask.Clone()
	case ClipInverse:
		inverse := mask.Inverse()
		if parent == nil {
			next = inverse
		} else {
			next = Intersect(parent, inverse)
		}
	default:
		if parent == nil {
			next = mask.Clone()
		} else {
			next = Intersect(parent, mask)
		}
	}

	c.clipStack = append(c.clipStack, next)
}

func (c *Canvas) Pop() {
	if len(c.clipStack) == 0 {
		panic("Pop called on empty clip stack")
	}
	c.clipStack = c.clipStack[:len(c.clipStack)-1]
}

func (c *Canvas) WithClip(mask *Mask, mode ClipMode, draw func()) {
	c.Push(mask, mode)
	defer c.Pop()
	draw()
}

type PaintFunc func(x, y int) color.NRGBA

func (c *Canvas) Fill(rect image.Rectangle, paint PaintFunc) {
	area := rect.Intersect(c.img.Rect)
	if area.Empty() {
		return
	}
	clip := c.currentClip()

	for y := area.Min.Y; y < area.Max.Y; y++ {
		for x := area.Min.X; x < area.Max.X; x++ {
			coverage := uint16(255)
			if clip != nil {
				coverage = uint16(clip.AlphaAt(x, y))
				if coverage == 0 {
					continue
				}
			}

			source := paint(x, y)
			sourceAlpha := uint16(source.A) * coverage / 255
			if sourceAlpha == 0 {
				continue
			}

			offset := c.img.PixOffset(x, y)
			destinationAlpha := uint32(c.img.Pix[offset+3])
			if destinationAlpha == 0 {
				c.img.Pix[offset+0] = source.R
				c.img.Pix[offset+1] = source.G
				c.img.Pix[offset+2] = source.B
				c.img.Pix[offset+3] = uint8(sourceAlpha)
				continue
			}

			sa := uint32(sourceAlpha)
			da := destinationAlpha
			outAlpha := sa + da*(255-sa)/255
			if outAlpha == 0 {
				c.img.Pix[offset+0] = 0
				c.img.Pix[offset+1] = 0
				c.img.Pix[offset+2] = 0
				c.img.Pix[offset+3] = 0
				continue
			}

			srcContributionR := uint32(source.R) * sa
			srcContributionG := uint32(source.G) * sa
			srcContributionB := uint32(source.B) * sa
			dstWeight := da * (255 - sa) / 255
			dstContributionR := uint32(c.img.Pix[offset+0]) * dstWeight
			dstContributionG := uint32(c.img.Pix[offset+1]) * dstWeight
			dstContributionB := uint32(c.img.Pix[offset+2]) * dstWeight

			c.img.Pix[offset+0] = uint8((srcContributionR + dstContributionR) / outAlpha)
			c.img.Pix[offset+1] = uint8((srcContributionG + dstContributionG) / outAlpha)
			c.img.Pix[offset+2] = uint8((srcContributionB + dstContributionB) / outAlpha)
			c.img.Pix[offset+3] = uint8(outAlpha)
		}
	}
}

// Cut applies destination-out. Full mask coverage sets RGBA to zero, making
// the cutout a real transparent hole rather than a painted background color.
func (c *Canvas) Cut(mask *Mask) {
	bounds := c.img.Rect
	clip := c.currentClip()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			cut := uint16(mask.AlphaAt(x, y))
			if clip != nil {
				cut = cut * uint16(clip.AlphaAt(x, y)) / 255
			}
			if cut == 0 {
				continue
			}
			keep := 255 - cut
			offset := c.img.PixOffset(x, y)
			for channel := 0; channel < 4; channel++ {
				value := uint16(c.img.Pix[offset+channel])
				c.img.Pix[offset+channel] = uint8(value * keep / 255)
			}
		}
	}
}

func Solid(c color.NRGBA) PaintFunc {
	return func(int, int) color.NRGBA { return c }
}
