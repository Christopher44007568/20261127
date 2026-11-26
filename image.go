package postfx

import (
	"errors"
	"image"
	"image/png"
	"math"
	"os"
)

// Image is a tightly packed linear RGB image. Values are not clamped until PNG
// encoding, so intermediate passes can safely exceed display range.
type Image struct {
	Width  int
	Height int
	Pix    []float64
}

func NewImage(width, height int) *Image {
	if width < 0 || height < 0 {
		panic("postfx: negative image dimensions")
	}
	return &Image{Width: width, Height: height, Pix: make([]float64, width*height*3)}
}

func (img *Image) Clone() *Image {
	dst := NewImage(img.Width, img.Height)
	copy(dst.Pix, img.Pix)
	return dst
}

func (img *Image) At(x, y int) (r, g, b float64) {
	if x < 0 || x >= img.Width || y < 0 || y >= img.Height {
		return 0, 0, 0
	}
	i := (y*img.Width + x) * 3
	return img.Pix[i], img.Pix[i+1], img.Pix[i+2]
}

func (img *Image) Set(x, y int, r, g, b float64) {
	i := (y*img.Width + x) * 3
	img.Pix[i] = r
	img.Pix[i+1] = g
	img.Pix[i+2] = b
}

// AbsDiff returns channel-wise absolute differences.
func AbsDiff(a, b *Image) (*Image, error) {
	if a.Width != b.Width || a.Height != b.Height {
		return nil, errors.New("postfx: image dimensions differ")
	}
	out := NewImage(a.Width, a.Height)
	for i := range out.Pix {
		out.Pix[i] = math.Abs(a.Pix[i] - b.Pix[i])
	}
	return out, nil
}

// MaxDiff is the largest channel difference between two images.
func MaxDiff(a, b *Image) (float64, error) {
	diff, err := AbsDiff(a, b)
	if err != nil {
		return 0, err
	}
	max := 0.0
	for _, value := range diff.Pix {
		if value > max {
			max = value
		}
	}
	return max, nil
}

func (img *Image) WritePNG(path string) error {
	rgba := image.NewNRGBA(image.Rect(0, 0, img.Width, img.Height))
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			r, g, b := img.At(x, y)
			i := y*rgba.Stride + x*4
			rgba.Pix[i+0] = encodeChannel(r)
			rgba.Pix[i+1] = encodeChannel(g)
			rgba.Pix[i+2] = encodeChannel(b)
			rgba.Pix[i+3] = 255
		}
	}
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := png.Encode(file, rgba); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func encodeChannel(value float64) uint8 {
	if math.IsNaN(value) || value <= 0 {
		return 0
	}
	if value >= 1 {
		return 255
	}
	return uint8(math.Round(value * 255))
}
