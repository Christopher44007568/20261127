package demo

import (
	"image"
	"image/color"
)

// ZoomOptions controls nearest-neighbor boundary enlargements. Nearest
// neighbor is deliberate: every output block is exactly one device pixel, so
// half-pixel seams or covered hole pixels cannot hide through interpolation.
type ZoomOptions struct {
	Crop  image.Rectangle // source rectangle; zero means whole image
	Scale int             // integer magnification, e.g. 12
	Grid  bool            // draw device-pixel grid lines
	GridC color.NRGBA
}

// ZoomMask enlarges a grayscale mask crop.
func ZoomMask(m *Mask, opt ZoomOptions) *image.NRGBA {
	src := MaskImage(m)
	return ZoomImage(src, opt)
}

// ZoomImage enlarges an image.Image crop with nearest-neighbor sampling.
func ZoomImage(src image.Image, opt ZoomOptions) *image.NRGBA {
	b := src.Bounds()
	crop := opt.Crop
	if crop.Empty() {
		crop = b
	} else {
		crop = crop.Intersect(b)
	}
	s := opt.Scale
	if s < 1 {
		s = 8
	}
	w := crop.Dx() * s
	h := crop.Dy() * s
	dst := image.NewNRGBA(image.Rect(0, 0, w, h))

	for y := 0; y < crop.Dy(); y++ {
		for x := 0; x < crop.Dx(); x++ {
			c := color.NRGBAModel.Convert(src.At(crop.Min.X+x, crop.Min.Y+y)).(color.NRGBA)
			for sy := 0; sy < s; sy++ {
				i := dst.PixOffset(x*s, y*s+sy)
				for sx := 0; sx < s; sx++ {
					j := i + sx*4
					dst.Pix[j+0] = c.R
					dst.Pix[j+1] = c.G
					dst.Pix[j+2] = c.B
					dst.Pix[j+3] = 255
				}
			}
		}
	}

	if opt.Grid {
		gc := opt.GridC
		if gc.A == 0 {
			gc = color.NRGBA{120, 128, 145, 160}
		}
		drawZoomGrid(dst, crop.Dx(), crop.Dy(), s, gc)
	}
	return dst
}

func drawZoomGrid(dst *image.NRGBA, nx, ny, s int, c color.NRGBA) {
	// 1 device-pixel hairline at every cell boundary.
	for x := 0; x <= nx; x++ {
		px := x * s
		for y := 0; y < ny*s; y++ {
			blendPixel(dst, px, y, c)
		}
	}
	for y := 0; y <= ny; y++ {
		py := y * s
		for x := 0; x < nx*s; x++ {
			blendPixel(dst, x, py, c)
		}
	}
}

func blendPixel(dst *image.NRGBA, x, y int, c color.NRGBA) {
	if !image.Pt(x, y).In(dst.Bounds()) {
		return
	}
	i := dst.PixOffset(x, y)
	a := uint32(c.A)
	dst.Pix[i+0] = uint8((uint32(c.R)*a + uint32(dst.Pix[i+0])*(255-a)) / 255)
	dst.Pix[i+1] = uint8((uint32(c.G)*a + uint32(dst.Pix[i+1])*(255-a)) / 255)
	dst.Pix[i+2] = uint8((uint32(c.B)*a + uint32(dst.Pix[i+2])*(255-a)) / 255)
	dst.Pix[i+3] = 255
}

// RectOutline draws a 1-pixel rectangle outline in place.
func RectOutline(dst *image.NRGBA, r image.Rectangle, c color.NRGBA) {
	for x := r.Min.X; x < r.Max.X; x++ {
		blendPixel(dst, x, r.Min.Y, c)
		blendPixel(dst, x, r.Max.Y-1, c)
	}
	for y := r.Min.Y; y < r.Max.Y; y++ {
		blendPixel(dst, r.Min.X, y, c)
		blendPixel(dst, r.Max.X-1, y, c)
	}
}

// CrossMarker draws a small plus marker centered at (cx,cy).
func CrossMarker(dst *image.NRGBA, cx, cy, arm int, c color.NRGBA) {
	for d := -arm; d <= arm; d++ {
		blendPixel(dst, cx+d, cy, c)
		blendPixel(dst, cx, cy+d, c)
	}
}
