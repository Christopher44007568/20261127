package demo

import (
	"image"
	"image/color"
	"image/png"
	"os"
)

// Palette groups the demo colors. NRGBA is non-premultiplied alpha.
type Palette struct {
	Paper     color.NRGBA
	PaperAlt  color.NRGBA
	Ink       color.NRGBA
	Grid      color.NRGBA
	EvenOdd   color.NRGBA
	NonZero   color.NRGBA
	NestedEO  color.NRGBA
	NestedNZ  color.NRGBA
	Island    color.NRGBA
	ClipMark  color.NRGBA
	Bad       color.NRGBA
	Good      color.NRGBA
	PanelHead color.NRGBA
}

// DefaultPalette is a color-blind-friendly-ish teal/indigo/amber set.
func DefaultPalette() Palette {
	return Palette{
		Paper:     color.NRGBA{248, 249, 252, 255},
		PaperAlt:  color.NRGBA{228, 232, 240, 255},
		Ink:       color.NRGBA{28, 32, 44, 255},
		Grid:      color.NRGBA{120, 128, 145, 70},
		EvenOdd:   color.NRGBA{91, 92, 214, 255},  // indigo
		NonZero:   color.NRGBA{16, 158, 140, 255}, // teal
		NestedEO:  color.NRGBA{214, 137, 16, 255}, // amber
		NestedNZ:  color.NRGBA{201, 84, 116, 255}, // raspberry
		Island:    color.NRGBA{38, 122, 214, 255}, // blue
		ClipMark:  color.NRGBA{224, 49, 49, 255},  // red
		Bad:       color.NRGBA{224, 49, 49, 255},
		Good:      color.NRGBA{47, 158, 68, 255},
		PanelHead: color.NRGBA{232, 236, 244, 255},
	}
}

// NewPaper returns an opaque image filled with a subtle checkerboard so that
// hole pixels are visually obvious. Checker size is 16 device pixels.
func NewPaper(w, h int, pal Palette) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	const cs = 16
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := pal.Paper
			if ((x/cs)+(y/cs))%2 == 1 {
				c = pal.PaperAlt
			}
			i := img.PixOffset(x, y)
			img.Pix[i+0] = c.R
			img.Pix[i+1] = c.G
			img.Pix[i+2] = c.B
			img.Pix[i+3] = 255
		}
	}
	return img
}

// PaintMask paints fill color onto dst exactly where mask is covered. Hole
// pixels (mask 0) are never touched by the fill, so the main contour cannot
// cover nested holes: there is one mask for the whole path and one paint pass.
func PaintMask(dst *image.NRGBA, m *Mask, fill color.NRGBA) {
	b := dst.Bounds()
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			if m.At(x, y) != 255 {
				continue
			}
			if image.Pt(x, y).In(b) {
				i := dst.PixOffset(x+b.Min.X, y+b.Min.Y)
				dst.Pix[i+0] = fill.R
				dst.Pix[i+1] = fill.G
				dst.Pix[i+2] = fill.B
				dst.Pix[i+3] = 255
			}
		}
	}
}

// PaintMaskAt paints a mask with its origin at (ox, oy) inside dst.
func PaintMaskAt(dst *image.NRGBA, m *Mask, ox, oy int, fill color.NRGBA) {
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			if m.At(x, y) != 255 {
				continue
			}
			px, py := ox+x, oy+y
			if !(image.Pt(px, py).In(dst.Bounds())) {
				continue
			}
			i := dst.PixOffset(px, py)
			dst.Pix[i+0] = fill.R
			dst.Pix[i+1] = fill.G
			dst.Pix[i+2] = fill.B
			dst.Pix[i+3] = 255
		}
	}
}

// MaskImage converts a coverage mask to an 8-bit grayscale image.
func MaskImage(m *Mask) *image.Gray {
	g := image.NewGray(image.Rect(0, 0, m.W, m.H))
	for y := 0; y < m.H; y++ {
		for x := 0; x < m.W; x++ {
			g.SetGray(x, y, color.Gray{Y: m.At(x, y)})
		}
	}
	return g
}

// WritePNG encodes any image as PNG.
func WritePNG(path string, img image.Image) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := png.Encoder{CompressionLevel: png.DefaultCompression}
	if err := enc.Encode(f, img); err != nil {
		return err
	}
	return f.Close()
}

// WriteMaskPNG writes the grayscale fill mask.
func WriteMaskPNG(path string, m *Mask) error {
	return WritePNG(path, MaskImage(m))
}

// SavePNGs is a small helper used by main to surface write errors together.
func SavePNGs(jobs ...struct {
	Path string
	Img  image.Image
}) []error {
	var errs []error
	for _, j := range jobs {
		if err := WritePNG(j.Path, j.Img); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
